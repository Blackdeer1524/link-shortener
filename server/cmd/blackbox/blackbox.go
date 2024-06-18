package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os/signal"
	"shortener/proto/blackbox"
	"syscall"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
)

type BlackboxServiceImpl struct {
	blackbox.UnimplementedBlackboxServiceServer
	secret string
	ctx    context.Context
}

type serviceOption func(*BlackboxServiceImpl) error

func WithSecret(secret string) serviceOption {
	return func(s *BlackboxServiceImpl) error {
		s.secret = secret
		return nil
	}
}

func WithContext(ctx context.Context) serviceOption {
	return func(s *BlackboxServiceImpl) error {
		s.ctx = ctx
		return nil
	}
}

func NewBlackboxServiceImpl(
	opts ...serviceOption,
) (*BlackboxServiceImpl, error) {
	s := new(BlackboxServiceImpl)
	for _, opt := range opts {
		opt(s)
	}

	if s.secret == "" {
		return nil, fmt.Errorf("no secret provided")
	}

	if s.ctx == nil {
		return nil, fmt.Errorf("no context provided")
	}

	return s, nil
}

func (s *BlackboxServiceImpl) IssueToken(
	ctx context.Context,
	r *blackbox.IssueTokenReq,
) (*blackbox.IssueTokenRsp, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.MapClaims{
			"sub": r.GetUserId(),
		})

	signedToken, err := token.SignedString([]byte(s.secret))
	if err != nil {
		return nil, err
	}
	res := &blackbox.IssueTokenRsp{
		Token: signedToken,
	}

	return res, err
}

func (s *BlackboxServiceImpl) ValidateToken(
	ctx context.Context,
	r *blackbox.ValidateTokenReq,
) (*blackbox.ValidateTokenRsp, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	token := r.GetToken()

	_, err := jwt.Parse(
		token,
		func(*jwt.Token) (interface{}, error) {
			return s.secret, nil
		},
		jwt.WithValidMethods([]string{"HS256"}),
	)
	if err != nil {
		return nil, err
	}

	res := &blackbox.ValidateTokenRsp{
		Token: true,
	}
	return res, nil
}

func main() {
	s := grpc.NewServer()

	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalln("couldn't start listening for connections. reason: ", err)
	}

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGTERM,
		syscall.SIGINT,
	)
	defer cancel()

	service, err := NewBlackboxServiceImpl(
		WithSecret("some secret key"),
		WithContext(ctx),
	)
	if err != nil {
		log.Fatalln("couldn't instantiate service impl. reason: ", err)
	}
	blackbox.RegisterBlackboxServiceServer(s, service)

	go func() {
		<-ctx.Done()
		s.GracefulStop()
	}()

	if err := s.Serve(lis); err != nil {
		log.Fatalln("fatal serve error: ", err)
	}
}
