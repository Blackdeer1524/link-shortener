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

	"google.golang.org/grpc"
)

type shortenReq struct {
	Url string `json:"url" validate:"required,url"`
}

type shortener struct {
	blackboxClient blackbox.BlackboxServiceClient
}

func NewShortener(c blackbox.BlackboxServiceClient) *shortener {
	return &shortener{blackboxClient: c}
}

func (s *shortener) shortenUrl(w http.ResponseWriter, r *http.Request) {
	log.Println("got url shortening request from ", r.RemoteAddr)

	cookie, err := r.Cookie("JWT")
	gotJWT := true
	if err != nil {
		switch {
		case errors.Is(err, http.ErrNoCookie):
			gotJWT = false
		default:
			log.Println(
				"caught error during JWT cookie processing. error: ",
				err,
			)
			res, _ := json.Marshal(&response.Server{
				Status:  response.StatusError,
				Message: "caught error during JWT cookie processing",
			})
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(res)
			return
		}
	}

	if !gotJWT {
		s.shortenNoAuth(w, r)
		return
	}

	log.Printf(
		"got shortening request with JWT token from %s. checking token validity.\n",
		r.RemoteAddr,
	)
	isValid, err := s.blackboxClient.ValidateToken(
		context.TODO(),
		&blackbox.ValidateTokenReq{
			Token: cookie.Value,
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

	if isValid.GetIsValid() {
		log.Println("got valid tokin from ", r.RemoteAddr)
		s.shortenAuth(w, r)
	} else {
		log.Println("got invalid tokin from ", r.RemoteAddr)
		res, _ := json.Marshal(&response.Server{
			Status:  response.StatusError,
			Message: "JWT is invalid",
		})
		w.WriteHeader(http.StatusBadRequest)
		w.Write(res)
	}
}

func (s *shortener) shortenAuth(w http.ResponseWriter, r *http.Request) {
}

func (s *shortener) shortenNoAuth(w http.ResponseWriter, r *http.Request) {
	d := json.NewDecoder(r.Body)
	var form shortenReq
	if err := d.Decode(&form); err != nil {
		log.Println("couldn't decode. reason: ", err)
	}

	res, _ := json.Marshal(&response.Server{
		Status:  0,
		Message: "`" + form.Url + "` but shorter",
	})

	w.WriteHeader(200)
	w.Write(res)
}

func handlePreflightReques(w http.ResponseWriter, r *http.Request) {
	w.Header().
		Add("Access-Control-Allow-Origin", "http://localhost:5173")
	w.Header().Add("Access-Control-Allow-Credentials", "true")
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
}

func main() {
	conn, err := grpc.Dial("blackbox:8080")
	if err != nil {
		log.Fatalln("couldn't dial auth service. reason: ", err)
	}
	defer conn.Close()

	s := NewShortener(blackbox.NewBlackboxServiceClient(conn))

	http.HandleFunc("OPTIONS /", http.HandlerFunc(handlePreflightReques))
	http.HandleFunc(
		"POST /",
		middleware.CorsHeaders(http.HandlerFunc(s.shortenUrl)),
	)
	http.ListenAndServe(":8080", nil)
}
