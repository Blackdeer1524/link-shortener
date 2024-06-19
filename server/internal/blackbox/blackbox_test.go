package blackbox

import (
	"context"
	"log"
	"net"
	"shortener/proto/blackbox"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	pbblackbox "shortener/proto/blackbox"
)

const secret = "secret"

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func init() {
	lis = bufconn.Listen(bufSize)
	s := grpc.NewServer()

	service, err := New(WithSecret(secret))
	if err != nil {
		panic(err)
	}

	pbblackbox.RegisterBlackboxServiceServer(s, service)
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()
}

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func TestIssueToken(t *testing.T) {
	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx,
		"bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := pbblackbox.NewBlackboxServiceClient(conn)

	userId := "id"
	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.MapClaims{
			"sub": userId,
		})
	signedToken, err := token.SignedString([]byte(secret))

	res, err := client.IssueToken(context.Background(), &blackbox.IssueTokenReq{
		UserId: userId,
	})
	assert.Nil(t, err)
	assert.Equal(t, signedToken, res.GetToken())
}

func TestValidateToken(t *testing.T) {
	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx,
		"bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := pbblackbox.NewBlackboxServiceClient(conn)

	userId := "id"
	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.MapClaims{
			"sub": userId,
		})
	signedToken, err := token.SignedString([]byte(secret))

	res, err := client.ValidateToken(
		context.Background(),
		&blackbox.ValidateTokenReq{
			Token: signedToken,
		},
	)

	assert.Nil(t, err)
	assert.Equal(t, userId, res.GetUserId())
}

func TestValidateTokenFailNoSub(t *testing.T) {
	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx,
		"bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := pbblackbox.NewBlackboxServiceClient(conn)

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.MapClaims{})
	signedToken, err := token.SignedString([]byte(secret))

	res, err := client.ValidateToken(
		context.Background(),
		&blackbox.ValidateTokenReq{
			Token: signedToken,
		},
	)
	assert.Nil(t, res)

	pberr, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, pberr.Code())
}

func TestValidateTokenFailInvalidToken(t *testing.T) {
	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx,
		"bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := pbblackbox.NewBlackboxServiceClient(conn)

	signedToken := "fake_token"

	res, err := client.ValidateToken(
		context.Background(),
		&blackbox.ValidateTokenReq{
			Token: signedToken,
		},
	)

	assert.Nil(t, res)
	pberr, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, pberr.Code())
}
