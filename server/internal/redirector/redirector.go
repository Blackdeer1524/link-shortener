package redirector

import (
	"context"
	"errors"
	"log"
	"net/http"
	"shortener/pkg/models/urls"
	"strings"
)

type Urls interface {
	GetLongUrl(ctx context.Context, shortUrl string) (string, error)
}

type Redirector struct {
	urls Urls
}

type redirectorOption func(r *Redirector) error

func WithUrlsModel(u Urls) redirectorOption {
	return func(r *Redirector) error {
		r.urls = u
		return nil
	}
}

func New(opts ...redirectorOption) (*Redirector, error) {
	r := new(Redirector)
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

func (re *Redirector) Redirect(w http.ResponseWriter, r *http.Request) {
	shortUrl := strings.TrimLeft(r.URL.Path, "/")
	log.Printf("got redirect request from %s: %s\n", r.RemoteAddr, shortUrl)
	if len(shortUrl) != 5 {
		http.NotFound(w, r)
		return
	}

	longUrl, err := re.urls.GetLongUrl(context.TODO(), shortUrl)
	if err == nil {
		w.Header().Add("Cache-Control", "no-cache")
		http.Redirect(w, r, longUrl, http.StatusMovedPermanently)
		return
	}
	if !errors.Is(err, urls.ErrNotFound) {
		log.Printf(
			"couldn't get long url for %s from db. error: %v",
			shortUrl,
			err,
		)
	}
	http.NotFound(w, r)
}
