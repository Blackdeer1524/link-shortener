package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"shortener-server/pkg/middleware"
	"shortener-server/pkg/response"
)

type shortenReq struct {
	Url string `json:"url"`
}

func shortenAuthUrl(w http.ResponseWriter, r *http.Request) {
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
	log.Println("got cookie: ", cookie.Value)

	d := json.NewDecoder(r.Body)
	var form shortenReq
	err = d.Decode(&form)
	if err != nil {
		log.Println("couldn't decode. reason: ", err)
	}

	res, _ := json.Marshal(&response.Server{
		Status:  0,
		Message: "`" + form.Url + "` but shorter",
	})

	w.WriteHeader(200)
	w.Write(res)
}

func main() {
	http.HandleFunc(
		"OPTIONS /",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().
				Add("Access-Control-Allow-Origin", "http://localhost:5173")
			w.Header().Add("Access-Control-Allow-Credentials", "true")
			w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
		}),
	)
	http.HandleFunc(
		"POST /", middleware.CorsHeaders(http.HandlerFunc(shortenAuthUrl)),
	)
	http.ListenAndServe(":8080", nil)
}
