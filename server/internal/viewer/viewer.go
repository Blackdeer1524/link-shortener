package viewer

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"shortener/pkg/domain"
	"shortener/pkg/responses"
	"shortener/proto/blackbox"

	"github.com/rs/zerolog/hlog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbblackbox "shortener/proto/blackbox"
)

type Urls interface {
	History(ctx context.Context, userId string) ([]*domain.UrlInfo, error)
}

type Viewer struct {
	redirectorHost string
	urls           Urls
	blackboxClient pbblackbox.BlackboxServiceClient
}

type viewerOption func(*Viewer) error

func WithUrls(u Urls) viewerOption {
	return func(v *Viewer) error {
		v.urls = u
		return nil
	}
}

func WithBlackboxClient(c pbblackbox.BlackboxServiceClient) viewerOption {
	return func(v *Viewer) error {
		v.blackboxClient = c
		return nil
	}
}

func WithRedirectorHost(host string) viewerOption {
	return func(v *Viewer) error {
		v.redirectorHost = host
		return nil
	}
}

func New(opts ...viewerOption) (*Viewer, error) {
	v := new(Viewer)
	for _, opt := range opts {
		if err := opt(v); err != nil {
			return nil, err
		}
	}
	if v.blackboxClient == nil {
		return nil, errors.New("no blackbox client provided")
	}
	if v.urls == nil {
		return nil, errors.New("no urls model provided")
	}
	if v.redirectorHost == "" {
		return nil, errors.New("no redirector host provided")
	}

	return v, nil
}

func (v *Viewer) HandleHistory(w http.ResponseWriter, r *http.Request) {
	log := hlog.FromRequest(r)

	log.Info().Msg("got new history request")
	JWTCookie, err := r.Cookie("JWT")
	gotJWT := true
	if err != nil {
		switch {
		case errors.Is(err, http.ErrNoCookie):
			gotJWT = false
		default:
			log.Error().
				Err(err).
				Msg("caught error during JWT cookie processing")
			res, _ := json.Marshal(&responses.Server{
				Message: "caught error during JWT cookie processing",
			})
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write(res)
			return
		}
	}

	if !gotJWT {
		log.Info().Msg("unauthenticated user tried to get shortening history")
		res, _ := json.Marshal(&responses.Server{
			Message: "no JWT cookie provided",
		})
		w.WriteHeader(http.StatusPreconditionFailed)
		w.Write(res)
		return
	}

	log.Info().Msg("validating JWT")
	tokenInfo, err := v.blackboxClient.ValidateToken(
		context.TODO(),
		&blackbox.ValidateTokenReq{
			Token: JWTCookie.Value,
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("couldn't validate jwt")
		s, ok := status.FromError(err)
		if !ok {
			res, _ := json.Marshal(&responses.Server{
				Message: "couldn't validate JWT",
			})
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(res)
			return
		}

		switch s.Code() {
		case codes.DeadlineExceeded:
			res, _ := json.Marshal(&responses.Server{
				Message: "deadline exceeded",
			})
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write(res)
		case codes.InvalidArgument:
			res, _ := json.Marshal(&responses.Server{
				Message: "invalid JWT",
			})
			w.WriteHeader(http.StatusForbidden)
			w.Write(res)
		default:
			log.Printf("unknown code from grpc: %v", s.Code())

			res, _ := json.Marshal(&responses.Server{
				Message: "couldn't validate JWT",
			})
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(res)
		}
		return
	}

	log.Info().Msg("getting shortening history")
	history, err := v.urls.History(context.TODO(), tokenInfo.GetUserId())
	if err != nil {
		log.Error().Err(err).Msg("couldn't get history")

		res, _ := json.Marshal(&responses.Server{
			Message: "couldn't get history",
		})
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(res)
		return
	}

	for i := range history {
		history[i].ShortUrl = v.redirectorHost + "/" + history[i].ShortUrl
	}

	res, _ := json.Marshal(&history)
	w.WriteHeader(http.StatusOK)
	w.Write(res)
}
