package authenticator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"shortener/pkg/models/users"
	"shortener/pkg/responses"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbblackbox "shortener/proto/blackbox"
)

type Users interface {
	Authenticate(
		ctx context.Context,
		email string,
		password string,
	) (string, error)

	CheckExistence(ctx context.Context, email string) (bool, error)
}

type Authentitor struct {
	logger         *zerolog.Logger
	users          Users
	blackboxClient pbblackbox.BlackboxServiceClient

	topic    string
	producer sarama.AsyncProducer
}

type authenticatorOption func(*Authentitor) error

func WithUsersDB(users Users) authenticatorOption {
	return func(a *Authentitor) error {
		a.users = users
		return nil
	}
}

func WithBlackboxClient(
	c pbblackbox.BlackboxServiceClient,
) authenticatorOption {
	return func(a *Authentitor) error {
		a.blackboxClient = c
		return nil
	}
}

func WithProducer(
	topic string,
	producer sarama.AsyncProducer,
) authenticatorOption {
	return func(a *Authentitor) error {
		a.topic = topic
		a.producer = producer
		return nil
	}
}

func New(opts ...authenticatorOption) (*Authentitor, error) {
	a := new(Authentitor)
	for _, opt := range opts {
		err := opt(a)
		if err != nil {
			return nil, err
		}
	}
	if a.users == nil {
		return nil, fmt.Errorf("no Users model provided")
	}
	if a.blackboxClient == nil {
		return nil, fmt.Errorf("no blackbox client provided")
	}
	if a.topic == "" {
		return nil, fmt.Errorf("no topic provided")
	}
	if a.producer == nil {
		return nil, fmt.Errorf("no producer provided")
	}

	return a, nil
}

const hashCost = 12

func (a *Authentitor) Register(w http.ResponseWriter, r *http.Request) {
	log := hlog.FromRequest(r)

	log.Info().Msg("decoding request body")
	d := json.NewDecoder(r.Body)
	var regForm registerRequest
	if err := d.Decode(&regForm); err != nil {
		log.Error().Err(err).Msg("couldn't parse registration request from")

		pkg, _ := json.Marshal(&responses.Server{
			Message: "Couldn't parse registration request",
		})

		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write(pkg)
		return
	}

	log.Info().Msg("validating parsed request body")
	if err := validate.Struct(&regForm); err != nil {
		log.Error().Err(err).Msg("invalid registration form")

		pkg, _ := json.Marshal(&responses.Server{
			Message: "invalid registration form from",
		})

		w.WriteHeader(http.StatusBadRequest)
		w.Write(pkg)
		return
	}

	tmp := log.With().Str("email", regForm.Email).Logger()
	log = &tmp

	log.Info().Msg("generating password hash")
	hashedPwd, err := bcrypt.GenerateFromPassword(
		[]byte(regForm.Password),
		hashCost,
	)
	if err != nil {
		log.Error().
			Err(err).
			Msg("couldn't hash password")
		pkg, _ := json.Marshal(&responses.Server{
			Message: "couldn't register user",
		})

		w.WriteHeader(http.StatusInternalServerError)
		w.Write(pkg)
		return
	}

	userId := uuid.New().String()
	p, _ := json.Marshal(&responses.Authenticator{
		Id:             userId,
		Name:           regForm.Name,
		Email:          regForm.Email,
		HashedPassword: string(hashedPwd),
	})

	log.Info().
		Err(err).
		Msg("checking whether this email has already been taken")
	exists, err := a.users.CheckExistence(context.TODO(), regForm.Email)
	if err != nil {
		log.Error().
			Err(err).
			Msg("couldn't query database")
		pkg, _ := json.Marshal(&responses.Server{
			Message: "couldn't instantiate new user. try again later",
		})

		w.WriteHeader(http.StatusInternalServerError)
		w.Write(pkg)
		return
	}

	if exists {
		log.Info().
			Msg("user already exists")
		pkg, _ := json.Marshal(&responses.Server{
			Message: "user already exists",
		})
		w.WriteHeader(http.StatusConflict)
		w.Write(pkg)
		return
	}

	a.producer.Input() <- &sarama.ProducerMessage{
		Topic: a.topic,
		Value: sarama.ByteEncoder(p),
	}
	log.Info().Msg("sent registration request to Storage service")

	log.Info().Msg("issuing JWT")
	signedToken, err := a.blackboxClient.IssueToken(
		context.TODO(),
		&pbblackbox.IssueTokenReq{
			UserId: userId,
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("couldnt't issue token")
		s, ok := status.FromError(err)
		if !ok {
			pkg, _ := json.Marshal(&responses.Server{
				Message: "couldn't issue JWT",
			})
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(pkg)
			return
		}

		switch s.Code() {
		case codes.Internal:
			pkg, _ := json.Marshal(&responses.Server{
				Message: "couldn't issue JWT",
			})
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(pkg)
		case codes.FailedPrecondition:
			pkg, _ := json.Marshal(&responses.Server{
				Message: "couldn't issue JWT",
			})
			// NOTE: precondition failure lies on other service, not the user. Hence, not 412 status code.
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(pkg)
		case codes.DeadlineExceeded:
			pkg, _ := json.Marshal(&responses.Server{
				Message: "deadline exceeded",
			})
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write(pkg)
		default:
			log.Error().
				Uint32("grpc_code", uint32(s.Code())).
				Msg("unknown code from grpc")
			pkg, _ := json.Marshal(&responses.Server{
				Message: "couldn't issue JWT",
			})
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(pkg)
		}
		return
	}

	authCookie := http.Cookie{
		Name:     "auth",
		Value:    "pass",
		Path:     "/",
		MaxAge:   3600,
		Secure:   true,
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	}

	jwtCookie := http.Cookie{
		Name:     "JWT",
		Value:    signedToken.GetToken(),
		Path:     "/",
		MaxAge:   3600,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}

	http.SetCookie(w, &authCookie)
	http.SetCookie(w, &jwtCookie)
	w.WriteHeader(http.StatusOK)

	pkg, _ := json.Marshal(&responses.Server{
		Message: "success",
	})

	w.Write(pkg)
	log.Info().Str("user_id", userId).Msg("wriring response")
}

func (a *Authentitor) Login(w http.ResponseWriter, r *http.Request) {
	log := hlog.FromRequest(r)

	log.Info().Msg("parsing request body")
	d := json.NewDecoder(r.Body)
	var loginForm loginRequest
	if err := d.Decode(&loginForm); err != nil {
		log.Error().Err(err).Msg("couldn't parse login request")

		pkg, _ := json.Marshal(&responses.Server{
			Message: "Couldn't parse login request",
		})

		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write(pkg)
		return
	}

	log.Info().Msg("validating login form")
	if err := validate.Struct(&loginForm); err != nil {
		log.Println(
			"invalid login form from",
			r.RemoteAddr,
			". error:",
			err.Error(),
		)

		pkg, _ := json.Marshal(&responses.Server{
			Message: "invalid login form",
		})

		w.WriteHeader(http.StatusBadRequest)
		w.Write(pkg)
		return
	}

	log.Info().Str("email", loginForm.Email).Msg("trying to authenticate")
	userId, err := a.users.Authenticate(
		context.TODO(),
		loginForm.Email,
		loginForm.Password,
	)
	if err != nil {
		log.Error().
			Err(err).
			Msg("wrong email or password for login attempt from")

		if errors.Is(err, users.ErrWrongCredentials) {
			pkg, _ := json.Marshal(&responses.Server{
				Message: "wrong email or password",
			})

			w.WriteHeader(http.StatusForbidden)
			w.Write(pkg)
		} else {
			pkg, _ := json.Marshal(&responses.Server{
				Message: "couldn't check credentials",
			})

			w.WriteHeader(http.StatusInternalServerError)
			w.Write(pkg)
		}
		return
	}

	log.Info().Msg("Issuing JWT")
	signedToken, err := a.blackboxClient.IssueToken(
		context.TODO(),
		&pbblackbox.IssueTokenReq{
			UserId: userId,
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("couldn't sign token on login attempt for")
		pkg, _ := json.Marshal(&responses.Server{
			Message: "couldn't issue user token",
		})
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(pkg)
		return
	}

	authCookie := http.Cookie{
		Name:     "auth",
		Value:    "pass",
		Path:     "/",
		MaxAge:   3600,
		Secure:   true,
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	}

	jwtCookie := http.Cookie{
		Name:     "JWT",
		Value:    signedToken.GetToken(),
		Path:     "/",
		MaxAge:   3600,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}

	http.SetCookie(w, &authCookie)
	http.SetCookie(w, &jwtCookie)
	w.WriteHeader(http.StatusOK)

	pkg, _ := json.Marshal(&responses.Server{
		Message: "success",
	})

	w.Write(pkg)

	log.Info().Msg("successful login")
}
