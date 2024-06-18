package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"shortener-server/pkg/middleware"
	"shortener-server/pkg/models"
	"shortener-server/pkg/response"

	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
)

type App struct {
	secret string
	users  *models.Users // TODO: replace with an interface
}

type AppOption func(*App) error

func WithSecret(secret string) AppOption {
	return func(a *App) error {
		a.secret = secret
		return nil
	}
}

func WithUsersDB(users *models.Users) AppOption {
	return func(a *App) error {
		a.users = users
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
	if a.secret == "" {
		return nil, fmt.Errorf("no secret key provided")
	}
	if a.users == nil {
		return nil, fmt.Errorf("no Users model provided")
	}

	return a, nil
}

type RegisterRequest struct {
	Name            string `json:"name"             validate:"required,gt=0,lt=300"`
	Email           string `json:"login"            validate:"required,email"`
	Password        string `json:"password"         validate:"required,gte=8,lt=64"`
	ConfirmPassword string `json:"confirm_password" validate:"required,eqfield=Password"`
}

var validate = validator.New(validator.WithRequiredStructEnabled())

func (a *App) register(w http.ResponseWriter, r *http.Request) {
	log.Println("got new registration request from ", r.RemoteAddr)

	d := json.NewDecoder(r.Body)
	var regReq RegisterRequest
	if err := d.Decode(&regReq); err != nil {
		log.Println(
			"couldn't parse registration request from ",
			r.RemoteAddr,
			". reason: ",
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
			"invalid registration form from ",
			r.RemoteAddr,
			". reason: ",
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
			"couldnt't insert new user for ",
			r.RemoteAddr,
			". reason: ",
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

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.MapClaims{
			"sub": uuid,
		})

	signedTocken, err := token.SignedString([]byte(a.secret))
	if err != nil {
		log.Println(
			"couldnt't sign token for ",
			r.RemoteAddr,
			". reason: ",
			err.Error(),
		)
		pkg, _ := json.Marshal(&response.Server{
			Status:  response.StatusError,
			Message: "couldn't create user token",
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
		SameSite: http.SameSiteStrictMode,
	}

	jwtCookie := http.Cookie{
		Name:     "JWT",
		Value:    signedTocken,
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
}

func main() {
	usersModel := &models.Users{}
	app, err := NewApp(WithSecret("some secret"), WithUsersDB(usersModel))
	if err != nil {
		panic(err)
	}

	http.HandleFunc(
		"POST /register",
		middleware.CorsHeaders(http.HandlerFunc(app.register)),
	)
	http.ListenAndServe(":8080", nil)
}
