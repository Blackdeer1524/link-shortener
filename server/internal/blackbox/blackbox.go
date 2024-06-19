package blackbox

import (
	"context"
	"fmt"
	"log"
	"shortener/proto/blackbox"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type BlackboxServiceImpl struct {
	blackbox.UnimplementedBlackboxServiceServer
	secret string
}

type serviceOption func(*BlackboxServiceImpl) error

func WithSecret(secret string) serviceOption {
	return func(s *BlackboxServiceImpl) error {
		s.secret = secret
		return nil
	}
}

func New(
	opts ...serviceOption,
) (*BlackboxServiceImpl, error) {
	s := new(BlackboxServiceImpl)
	for _, opt := range opts {
		opt(s)
	}

	if s.secret == "" {
		return nil, fmt.Errorf("no secret provided")
	}

	return s, nil
}

func (s *BlackboxServiceImpl) IssueToken(
	ctx context.Context,
	r *blackbox.IssueTokenReq,
) (*blackbox.IssueTokenRsp, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.Errorf(codes.DeadlineExceeded, "deadline exceeded")
	}

	if r.GetUserId() == "" {
		return nil, status.Errorf(
			codes.FailedPrecondition,
			"UserId not provided",
		)
	}

	log.Println("issuing JWT for", r.GetUserId())

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.MapClaims{
			"sub": r.GetUserId(),
		})

	signedToken, err := token.SignedString([]byte(s.secret))
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			"couldn't sign a token: %v",
			err,
		)
	}

	res := &blackbox.IssueTokenRsp{
		Token: signedToken,
	}
	return res, nil
}

func (s *BlackboxServiceImpl) ValidateToken(
	ctx context.Context,
	r *blackbox.ValidateTokenReq,
) (*blackbox.ValidateTokenRsp, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.Errorf(codes.DeadlineExceeded, "deadline exceeded")
	}

	token := r.GetToken()

	parsedToken, err := jwt.Parse(
		token,
		func(*jwt.Token) (interface{}, error) {
			return []byte(s.secret), nil
		},
		jwt.WithValidMethods([]string{"HS256"}),
	)
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"invalid token",
		)
	}

	sub, err := parsedToken.Claims.GetSubject()
	if err != nil || sub == "" {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"couldn't get sub claim from JWT",
		)
	}

	res := &blackbox.ValidateTokenRsp{
		UserId: sub,
	}
	return res, nil
}
