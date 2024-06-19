package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"shortener/pkg/middleware"
	"shortener/pkg/response"
	"shortener/proto/blackbox"

	"github.com/go-playground/validator/v10"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var validate = validator.New(validator.WithRequiredStructEnabled())

type noAuthShortenReq struct {
	Url string `json:"url" validate:"required,url"`
}

type authShortenReq struct {
	Url string `json:"url" validate:"required,url"`
}

type shortener struct {
	blackboxClient blackbox.BlackboxServiceClient
}

type shortenerOption func(s *shortener) error

func WithBlackboxClient(c blackbox.BlackboxServiceClient) shortenerOption {
	return func(s *shortener) error {
		s.blackboxClient = c
		return nil
	}
}

func WithRedis(rdb *redis.Client) shortenerOption {
}

func NewShortener(c blackbox.BlackboxServiceClient) *shortener {
	return &shortener{blackboxClient: c}
}

func (s *shortener) shortenUrl(w http.ResponseWriter, r *http.Request) {
	log.Println("got url shortening request from", r.RemoteAddr)

	_, err := r.Cookie("auth")
	gotDummyCookie := true
	if err != nil {
		switch {
		case errors.Is(err, http.ErrNoCookie):
			gotDummyCookie = false
		default:
			log.Println(
				"caught error during dummy auth cookie processing. error:",
				err,
			)
			res, _ := json.Marshal(&response.Server{
				Status:  response.StatusError,
				Message: "caught error during dummy auth cookie processing",
			})
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(res)
			return
		}
	}

	if !gotDummyCookie {
		s.shortenNoAuth(w, r)
		return
	}

	JWTCookie, err := r.Cookie("JWT")
	if err != nil {
		switch {
		case errors.Is(err, http.ErrNoCookie):
			s.shortenNoAuth(w, r)
		default:
			log.Println(
				"caught error during JWT cookie processing. error:",
				err,
			)
			res, _ := json.Marshal(&response.Server{
				Status:  response.StatusError,
				Message: "caught error during JWT cookie processing",
			})
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(res)
		}
		return
	}

	log.Printf(
		"got shortening request with JWT token from %s. checking token validity.\n",
		r.RemoteAddr,
	)
	isValid, err := s.blackboxClient.ValidateToken(
		context.TODO(),
		&blackbox.ValidateTokenReq{
			Token: JWTCookie.Value,
		},
	)
	if err != nil {
		log.Printf(
			"couldn't validate jwt from %s. reason: %s\n",
			r.RemoteAddr,
			err.Error(),
		)

		res, _ := json.Marshal(&response.Server{
			Status:  response.StatusError,
			Message: "couldn't validate JWT",
		})
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(res)
		return
	}

	if !isValid.GetIsValid() {
		log.Println("got invalid tokin from", r.RemoteAddr)
		res, _ := json.Marshal(&response.Server{
			Status:  response.StatusError,
			Message: "JWT is invalid",
		})
		w.WriteHeader(http.StatusBadRequest)
		w.Write(res)
	}
	log.Println("got valid tokin from", r.RemoteAddr)
	s.shortenAuth(w, r)
}

func (s *shortener) shortenAuth(w http.ResponseWriter, r *http.Request) {
	d := json.NewDecoder(r.Body)
	var form authShortenReq
	if err := d.Decode(&form); err != nil {
		log.Println("couldn't decode. reason:", err)
		res, _ := json.Marshal(&response.Server{
			Status:  response.StatusError,
			Message: "Invalid form",
		})
		w.WriteHeader(http.StatusBadRequest)
		w.Write(res)
		return
	}

	if err := validate.Struct(&form); err != nil {
		res, _ := json.Marshal(&response.Server{
			Status:  response.StatusValidationError,
			Message: "form couldn't pass validation",
		})

		w.WriteHeader(http.StatusBadRequest)
		w.Write(res)
		return
	}

	res, _ := json.Marshal(&response.Server{
		Status:  0,
		Message: "`" + form.Url + "` short with auth",
	})

	w.WriteHeader(200)
	w.Write(res)
}

func (s *shortener) shortenNoAuth(w http.ResponseWriter, r *http.Request) {
	d := json.NewDecoder(r.Body)
	var form noAuthShortenReq
	if err := d.Decode(&form); err != nil {
		log.Println("couldn't decode. reason:", err)
	}

	res, _ := json.Marshal(&response.Server{
		Status:  0,
		Message: "`" + form.Url + "` short no auth",
	})

	w.WriteHeader(200)
	w.Write(res)
}

func handlePreflightRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().
		Add("Access-Control-Allow-Origin", "http://localhost:5173")
	w.Header().Add("Access-Control-Allow-Credentials", "true")
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
}

func main() {
	conn, err := grpc.NewClient(
		"blackbox:8080",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalln("couldn't dial auth service. reason:", err)
	}
	defer conn.Close()

	s := NewShortener(blackbox.NewBlackboxServiceClient(conn))

	http.HandleFunc(
		"OPTIONS /create_short_url",
		http.HandlerFunc(handlePreflightRequest),
	)
	http.HandleFunc(
		"POST /create_short_url",
		middleware.CorsHeaders(http.HandlerFunc(s.shortenUrl)),
	)
	http.ListenAndServe(":8080", nil)
}
