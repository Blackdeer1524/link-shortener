package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"shortener/pkg/models/urls"

	"github.com/redis/go-redis/v9"
)

type redirector struct {
	rdb  *redis.Client
	urls *urls.Model
}

type redirectorOption func(r *redirector) error

func WithUrlsModel(u *urls.Model) redirectorOption {
	return func(r *redirector) error {
		r.urls = u
		return nil
	}
}

func NewRedirector(opts ...redirectorOption) (*redirector, error) {
	r := new(redirector)
	for _, opt := range opts {
		if err := opt(r); err != nil {
			return nil, err
		}
	}
	if r.urls == nil {
		return nil, errors.New("no urls model provided")
	}
	return r, nil
}

func (re *redirector) redirect(w http.ResponseWriter, r *http.Request) {
	shortUrl := r.URL.Path
	if len(shortUrl) != 5 {
		http.NotFound(w, r)
		return
	}

	log.Printf(
		"got new shortening request from %s: %s\n",
		r.RemoteAddr,
		shortUrl,
	)

	longUrl, err := re.urls.GetLongUrl(context.TODO(), shortUrl)
	if err == nil {
		http.Redirect(w, r, longUrl, http.StatusMovedPermanently)
		return
	}
	if !errors.Is(err, urls.ErrNotFound) {
		log.Printf(
			"couldn't get long url for %s from db. reason: %v",
			shortUrl,
			err,
		)
	}
	http.NotFound(w, r)
}

func main() {
	rdb := redis.NewClient(&redis.Options{Addr: "redis:6379"})

	u, err := urls.New(
		urls.WithRedis(rdb),
		urls.WithPool(context.TODO(), os.Getenv("POSTGRES_DSN")),
	)
	if err != nil {
		log.Fatalln("couldn't instantiate urls model. reason:", err)
	}

	re, err := NewRedirector(WithUrlsModel(u))

	http.HandleFunc("GET /", re.redirect)
}
