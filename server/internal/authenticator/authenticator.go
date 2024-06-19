package authenticator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"shortener/pkg/models/users"
	"shortener/pkg/responses"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
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
	users Users

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
	log.Println("got new registration request from", r.RemoteAddr)

	d := json.NewDecoder(r.Body)
	var regReq registerRequest
	if err := d.Decode(&regReq); err != nil {
		log.Println(
			"couldn't parse registration request from",
			r.RemoteAddr,
			". error:",
			err.Error(),
		)

		pkg, _ := json.Marshal(&responses.Server{
			Message: "Couldn't parse registration request",
		})

		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write(pkg)
		return
	}

	if err := validate.Struct(&regReq); err != nil {
		log.Println(
			"invalid registration form",
			r.RemoteAddr,
			". error:",
			err.Error(),
		)

		pkg, _ := json.Marshal(&responses.Server{
			Message: "invalid registration form from",
		})

		w.WriteHeader(http.StatusBadRequest)
		w.Write(pkg)
		return
	}

	hashedPwd, err := bcrypt.GenerateFromPassword(
		[]byte(regReq.Password),
		hashCost,
	)
	if err != nil {
		log.Printf(
			"couldn't hash password. error: %v\n",
			err,
		)
		pkg, _ := json.Marshal(&responses.Server{
			Message: "couldn't register user",
		})

		w.WriteHeader(http.StatusInternalServerError)
		w.Write(pkg)
		return
	}

	userId := uuid.New().String()
	p, err := json.Marshal(&responses.Authenticator{
		Id:             userId,
		Name:           regReq.Name,
		Email:          regReq.Email,
		HashedPassword: string(hashedPwd),
	})
	if err != nil {
		panic(err)
	}

	exists, err := a.users.CheckExistence(context.TODO(), regReq.Email)
	if err != nil {
		log.Printf("couldn't query db. error: %v", err)
		pkg, _ := json.Marshal(&responses.Server{
			Message: "couldn't create new user. try again later",
		})

		w.WriteHeader(http.StatusInternalServerError)
		w.Write(pkg)
		return
	}

	if exists {
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

	signedToken, err := a.blackboxClient.IssueToken(
		context.TODO(),
		&pbblackbox.IssueTokenReq{
			UserId: userId,
		},
	)
	if err != nil {
		s, ok := status.FromError(err)
		log.Println(
			"couldnt't sign token for",
			r.RemoteAddr,
			". error:",
			err.Error(),
		)
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
			log.Printf("unknown code from grpc: %v", s.Code())
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
}

func (a *Authentitor) Login(w http.ResponseWriter, r *http.Request) {
	log.Println("got login request from", r.RemoteAddr)

	d := json.NewDecoder(r.Body)
	var loginForm loginRequest
	if err := d.Decode(&loginForm); err != nil {
		log.Println(
			"couldn't parse login request from",
			r.RemoteAddr,
			". error:",
			err.Error(),
		)

		pkg, _ := json.Marshal(&responses.Server{
			Message: "Couldn't parse login request",
		})

		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write(pkg)
		return
	}

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

	userId, err := a.users.Authenticate(
		context.TODO(),
		loginForm.Email,
		loginForm.Password,
	)
	if err != nil {
		log.Println(
			"wrong email or password for login attempt from",
			r.RemoteAddr,
			". error:",
			err.Error(),
		)

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

	signedToken, err := a.blackboxClient.IssueToken(
		context.TODO(),
		&pbblackbox.IssueTokenReq{
			UserId: userId,
		},
	)
	if err != nil {
		log.Println(
			"couldnt't sign token on login attempt for",
			r.RemoteAddr,
			". error:",
			err.Error(),
		)
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

	log.Println(
		"successful login from",
		r.RemoteAddr,
	)
}
