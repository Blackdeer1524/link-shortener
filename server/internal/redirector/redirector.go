package redirector

import (
	"context"
	"errors"
	"net/http"
	"shortener/pkg/models/urls"
	"strings"

	"github.com/rs/zerolog/hlog"
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
	log := hlog.FromRequest(r)

	shortUrl := strings.TrimLeft(r.URL.Path, "/")
	if len(shortUrl) != 5 {
		http.NotFound(w, r)
		return
	}

	log.Info().Msg("trying to get long url from the short one")
	longUrl, err := re.urls.GetLongUrl(context.TODO(), shortUrl)
	if err == nil {
		http.Redirect(w, r, longUrl, http.StatusMovedPermanently)
		return
	}
	if !errors.Is(err, urls.ErrNotFound) {
		log.Error().Err(err).Msg("couldn't get long url")
		http.NotFound(w, r)
	} else {
		log.Info().Err(err).Msg("long url not found in database")
		http.NotFound(w, r)
	}
}
