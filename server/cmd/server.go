package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
)

type Response struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

func corsHeaders(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Add("Access-Control-Allow-Credentials", "true")
		next.ServeHTTP(w, r)
	}
}

func getCookieHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("JWT")
	if err != nil {
		switch {
		case errors.Is(err, http.ErrNoCookie):
			http.Error(w, "cookie not found", http.StatusBadRequest)
		default:
			log.Println(err)
			http.Error(w, "server error", http.StatusInternalServerError)
		}
		return
	}

	w.Write([]byte(cookie.Value))
}

func register(w http.ResponseWriter, r *http.Request) {
	log.Println("got new request")
	t := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.MapClaims{
			"iss": "my-auth-server",
			"sub": "client",
		})

	str, err := t.SignedString([]byte("some secret key"))
	if err != nil {
		pkg, _ := json.Marshal(&Response{
			Status: 1,
			Message: fmt.Sprintf(
				"couldn't create token. error: %v\n",
				err,
			),
		})
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(pkg)
		return
	}

	cookie := http.Cookie{
		Name:     "JWT",
		Value:    str,
		Path:     "/",
		MaxAge:   3600,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}

	http.SetCookie(w, &cookie)
	w.WriteHeader(200)

	pkg, _ := json.Marshal(&Response{
		Status:  0,
		Message: "set jwt in cookie",
	})

	w.Write(pkg)
}

func main() {
	http.HandleFunc(
		"/register",
		corsHeaders(http.HandlerFunc(register)),
	)
	http.HandleFunc(
		"/cookie",
		corsHeaders(http.HandlerFunc(getCookieHandler)),
	)
	http.Handle("/", corsHeaders(http.HandlerFunc(register)))
	http.ListenAndServe(":8080", nil)
}
