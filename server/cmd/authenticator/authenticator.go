package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"shortener/pkg/middleware"
	"shortener/pkg/models"
	"shortener/pkg/response"
	"shortener/proto/blackbox"

	"github.com/go-playground/validator/v10"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type App struct {
	users          *models.Users // TODO: replace with an interface
	blackboxClient blackbox.BlackboxServiceClient
}

type AppOption func(*App) error

func WithUsersDB(users *models.Users) AppOption {
	return func(a *App) error {
		a.users = users
		return nil
	}
}

func WithBlackboxClient(c blackbox.BlackboxServiceClient) AppOption {
	return func(a *App) error {
		a.blackboxClient = c
		return nil
	}
}

func NewApp(opts ...AppOption) (*App, error) {
	a := new(App)
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

	return a, nil
}

type RegisterRequest struct {
	Name            string `json:"name"             validate:"required,gt=0,lt=300"`
	Email           string `json:"email"            validate:"required,email"`
	Password        string `json:"password"         validate:"required,gte=8,lt=64"`
	ConfirmPassword string `json:"confirm_password" validate:"required,eqfield=Password"`
}

var validate = validator.New(validator.WithRequiredStructEnabled())

func (a *App) register(w http.ResponseWriter, r *http.Request) {
	log.Println("got new registration request from", r.RemoteAddr)

	d := json.NewDecoder(r.Body)
	var regReq RegisterRequest
	if err := d.Decode(&regReq); err != nil {
		log.Println(
			"couldn't parse registration request from",
			r.RemoteAddr,
			". reason:",
			err.Error(),
		)

		pkg, _ := json.Marshal(&response.Server{
			Status:  response.StatusError,
			Message: "Couldn't parse registration request",
		})

		w.WriteHeader(http.StatusBadRequest)
		w.Write(pkg)
		return
	}

	if err := validate.Struct(&regReq); err != nil {
		log.Println(
			"invalid registration form from",
			r.RemoteAddr,
			". reason:",
			err.Error(),
		)

		pkg, _ := json.Marshal(&response.Server{
			Status:  response.StatusValidationError,
			Message: "invalid registration form from",
		})

		w.WriteHeader(http.StatusBadRequest)
		w.Write(pkg)
		return
	}

	uuid, err := a.users.Insert(regReq.Name, regReq.Email, regReq.Password)
	if err != nil {
		log.Println(
			"couldnt't insert new user for",
			r.RemoteAddr,
			". reason:",
			err.Error(),
		)
		if errors.Is(err, models.ErrAlreadyExists) {
			pkg, _ := json.Marshal(&response.Server{
				Status:  response.StatusValidationError,
				Message: "User already exists",
			})
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(pkg)
		} else {
			pkg, _ := json.Marshal(&response.Server{
				Status:  response.StatusError,
				Message: "couldn't create new user. Try again later",
			})

			w.WriteHeader(http.StatusInternalServerError)
			w.Write(pkg)
		}
		return
	}

	signedToken, err := a.blackboxClient.IssueToken(
		context.TODO(),
		&blackbox.IssueTokenReq{
			UserId: uuid,
		},
	)
	if err != nil {
		log.Println(
			"couldnt't sign token for",
			r.RemoteAddr,
			". reason:",
			err.Error(),
		)
		pkg, _ := json.Marshal(&response.Server{
			Status:  response.StatusError,
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
	w.WriteHeader(200)

	pkg, _ := json.Marshal(&response.Server{
		Status:  response.StatusOK,
		Message: "success",
	})

	w.Write(pkg)

	log.Printf(
		"successfully registered user `%s` from %s\n",
		regReq.Email,
		r.RemoteAddr,
	)
}

type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required,gte=8,lt=64"`
}

func (a *App) login(w http.ResponseWriter, r *http.Request) {
	log.Println("got login request from", r.RemoteAddr)

	d := json.NewDecoder(r.Body)
	loginForm := new(LoginRequest)
	if err := d.Decode(loginForm); err != nil {
		log.Println(
			"couldn't parse login request from",
			r.RemoteAddr,
			". reason:",
			err.Error(),
		)

		pkg, _ := json.Marshal(&response.Server{
			Status:  response.StatusError,
			Message: "Couldn't parse login request",
		})

		w.WriteHeader(http.StatusBadRequest)
		w.Write(pkg)
		return
	}

	if err := validate.Struct(loginForm); err != nil {
		log.Println(
			"invalid login form from",
			r.RemoteAddr,
			". reason:",
			err.Error(),
		)

		pkg, _ := json.Marshal(&response.Server{
			Status:  response.StatusValidationError,
			Message: "invalid login form",
		})

		w.WriteHeader(http.StatusBadRequest)
		w.Write(pkg)
		return
	}

	userId, err := a.users.Authenticate(loginForm.Email, loginForm.Password)
	if err != nil {
		log.Println(
			"wrong email or password for login attempt from",
			r.RemoteAddr,
			". reason:",
			err.Error(),
		)

		pkg, _ := json.Marshal(&response.Server{
			Status:  response.StatusValidationError,
			Message: "wrong email or password",
		})

		w.WriteHeader(http.StatusBadRequest)
		w.Write(pkg)
		return
	}

	signedToken, err := a.blackboxClient.IssueToken(
		context.TODO(),
		&blackbox.IssueTokenReq{
			UserId: userId,
		},
	)
	if err != nil {
		log.Println(
			"couldnt't sign token on login attempt for",
			r.RemoteAddr,
			". reason:",
			err.Error(),
		)
		pkg, _ := json.Marshal(&response.Server{
			Status:  response.StatusError,
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
	w.WriteHeader(200)

	pkg, _ := json.Marshal(&response.Server{
		Status:  response.StatusOK,
		Message: "success",
	})

	w.Write(pkg)

	log.Println(
		"successful login from",
		r.RemoteAddr,
	)
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

	c := blackbox.NewBlackboxServiceClient(conn)
	usersModel := &models.Users{}
	app, err := NewApp(WithUsersDB(usersModel), WithBlackboxClient(c))
	if err != nil {
		panic(err)
	}

	http.HandleFunc(
		"OPTIONS /register",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().
				Add("Access-Control-Allow-Origin", "http://localhost:5173")
			w.Header().Add("Access-Control-Allow-Credentials", "true")
			w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
		}),
	)

	http.HandleFunc(
		"OPTIONS /login",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().
				Add("Access-Control-Allow-Origin", "http://localhost:5173")
			w.Header().Add("Access-Control-Allow-Credentials", "true")
			w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
		}),
	)

	http.HandleFunc(
		"POST /register",
		middleware.CorsHeaders(http.HandlerFunc(app.register)),
	)
	http.HandleFunc(
		"POST /login",
		middleware.CorsHeaders(http.HandlerFunc(app.login)),
	)
	http.ListenAndServe(":8080", nil)
}
