package shortener

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"shortener/pkg/responses"
	"shortener/proto/blackbox"
	"time"

	"github.com/IBM/sarama"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Urls interface {
	CheckExistence(ctx context.Context, shortUrl string) (bool, error)
}

type Shortener struct {
	urls           Urls
	blackboxClient blackbox.BlackboxServiceClient
	redirectorHost string

	producer sarama.AsyncProducer
	topic    string
}

type shortenerOption func(s *Shortener) error

func WithBlackboxClient(c blackbox.BlackboxServiceClient) shortenerOption {
	return func(s *Shortener) error {
		s.blackboxClient = c
		return nil
	}
}

func WithUrlsModel(u Urls) shortenerOption {
	return func(s *Shortener) error {
		s.urls = u
		return nil
	}
}

func WithKafkaProducer(p sarama.AsyncProducer, topic string) shortenerOption {
	return func(s *Shortener) error {
		s.producer = p
		s.topic = topic
		return nil
	}
}

func WithRedirectorHost(host string) shortenerOption {
	return func(s *Shortener) error {
		s.redirectorHost = host
		return nil
	}
}

func New(opts ...shortenerOption) (*Shortener, error) {
	s := new(Shortener)
	for _, opt := range opts {
		if err := opt(s); err != nil {
			return nil, err
		}
	}

	if s.urls == nil {
		return nil, errors.New("no urls models provided")
	}

	if s.blackboxClient == nil {
		return nil, errors.New("no blackbox client provided")
	}

	if s.producer == nil {
		return nil, errors.New("no kafka producer provided")
	}

	if s.redirectorHost == "" {
		return nil, errors.New("no redirector host provided")
	}

	return s, nil
}

func (s *Shortener) ShortenUrl(w http.ResponseWriter, r *http.Request) {
	log.Println("got url shortening request from", r.RemoteAddr)

	JWTCookie, err := r.Cookie("JWT")
	gotJWT := true
	if err != nil {
		switch {
		case errors.Is(err, http.ErrNoCookie):
			gotJWT = false
		default:
			log.Println(
				"caught error during JWT cookie processing. error:",
				err,
			)
			res, _ := json.Marshal(&responses.Server{
				Message: "caught error during JWT cookie processing",
			})
			w.WriteHeader(http.StatusUnprocessableEntity)
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
	tokenInfo, err := s.blackboxClient.ValidateToken(
		context.TODO(),
		&blackbox.ValidateTokenReq{
			Token: JWTCookie.Value,
		},
	)
	if err != nil {
		log.Printf(
			"couldn't validate jwt from %s. error: %s\n",
			r.RemoteAddr,
			err.Error(),
		)
		s, ok := status.FromError(err)
		if !ok {
			res, _ := json.Marshal(&responses.Server{
				Message: "couldn't validate JWT",
			})
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(res)
			return
		}

		switch s.Code() {
		case codes.DeadlineExceeded:
			res, _ := json.Marshal(&responses.Server{
				Message: "deadline exceeded",
			})
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write(res)
		case codes.InvalidArgument:
			res, _ := json.Marshal(&responses.Server{
				Message: "invalid JWT",
			})
			w.WriteHeader(http.StatusForbidden)
			w.Write(res)
		default:
			log.Printf("unknown code from grpc: %v", s.Code())

			res, _ := json.Marshal(&responses.Server{
				Message: "couldn't validate JWT",
			})
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(res)
		}
		return
	}

	log.Printf(
		"got valid tokin from %s with user id %s\n ",
		r.RemoteAddr,
		tokenInfo.GetUserId(),
	)

	s.shortenAuth(tokenInfo.GetUserId(), w, r)
}

func (s *Shortener) shortenAuth(
	from string,
	w http.ResponseWriter,
	r *http.Request,
) {
	d := json.NewDecoder(r.Body)
	var form authShortenReq
	if err := d.Decode(&form); err != nil {
		log.Println("couldn't decode auth shortening req. error:", err)
		res, _ := json.Marshal(&responses.Server{
			Message: "Bad shortening form",
		})

		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write(res)
		return
	}

	if err := validate.Struct(&form); err != nil {
		log.Println(
			"invalid url shortening form (auth) from",
			r.RemoteAddr,
			". error:",
			err.Error(),
		)

		pkg, _ := json.Marshal(&responses.Server{
			Message: "invalid shortening form",
		})

		w.WriteHeader(http.StatusBadRequest)
		w.Write(pkg)
		return
	}

	var shortUrl string
	for {
		shortUrl = generateShortUrl(5)
		exists, err := s.urls.CheckExistence(context.TODO(), shortUrl)
		if err != nil {
			log.Println("couldn't check existence of short url. error:", err)
			continue
		}
		if !exists {
			break
		}
	}

	m, _ := json.Marshal(&responses.Shortener{
		From:     from,
		ShortUrl: shortUrl,
		LongUrl:  form.Url,
		ExpirationDate: time.Now().
			Add(time.Hour * 24 * time.Duration(form.Expiration)),
	})

	s.producer.Input() <- &sarama.ProducerMessage{
		Topic: s.topic,
		Value: sarama.ByteEncoder(m),
	}

	res, _ := json.Marshal(&responses.Server{
		Message: s.redirectorHost + "/" + shortUrl,
	})

	w.WriteHeader(http.StatusOK)
	w.Write(res)
}

func (s *Shortener) shortenNoAuth(w http.ResponseWriter, r *http.Request) {
	d := json.NewDecoder(r.Body)
	var form noAuthShortenReq
	if err := d.Decode(&form); err != nil {
		log.Println("couldn't decode no auth shortening req. error:", err)
		res, _ := json.Marshal(&responses.Server{
			Message: "Bad shortening form",
		})

		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write(res)
		return
	}

	if err := validate.Struct(&form); err != nil {
		log.Println(
			"invalid url shortening form from",
			r.RemoteAddr,
			". error:",
			err.Error(),
		)

		pkg, _ := json.Marshal(&responses.Server{
			Message: "invalid shortening form",
		})

		w.WriteHeader(http.StatusBadRequest)
		w.Write(pkg)
		return
	}

	var shortUrl string
	for {
		shortUrl = generateShortUrl(5)
		exists, err := s.urls.CheckExistence(context.TODO(), shortUrl)
		if err != nil {
			log.Println("couldn't check existence of short url. error:", err)
			continue
		}
		if !exists {
			break
		}
	}

	// uuid of an anonymous user
	const dummyUUID = "db092ed4-306a-4d4f-be5f-fd2f1487edbe"
	m, _ := json.Marshal(&responses.Shortener{
		From:           dummyUUID,
		ShortUrl:       shortUrl,
		LongUrl:        form.Url,
		ExpirationDate: time.Now().Add(time.Hour * 24 * 30),
	})

	s.producer.Input() <- &sarama.ProducerMessage{
		Topic: s.topic,
		Value: sarama.ByteEncoder(m),
	}

	res, _ := json.Marshal(&responses.Server{
		Message: s.redirectorHost + "/" + shortUrl,
	})

	w.WriteHeader(http.StatusOK)
	w.Write(res)
}
