package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math/rand"
	"net/http"
	"os"
	"shortener/pkg/middleware"
	"shortener/pkg/models/urls"
	"shortener/pkg/response"
	"shortener/proto/blackbox"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/go-playground/validator/v10"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var validate = validator.New(validator.WithRequiredStructEnabled())

type noAuthShortenReq struct {
	Url string `json:"url" validate:"required,url"`
}

type authShortenReq struct {
	Url            string    `json:"url"             validate:"required,url"`
	ExpirationDate time.Time `json:"expiration_date" validate:"required,time"`
}

type shortener struct {
	urls           *urls.Model
	blackboxClient blackbox.BlackboxServiceClient
	redirectorHost string

	producer sarama.AsyncProducer
	topic    string
}

type shortenerOption func(s *shortener) error

func WithBlackboxClient(c blackbox.BlackboxServiceClient) shortenerOption {
	return func(s *shortener) error {
		s.blackboxClient = c
		return nil
	}
}

func WithUrlsModel(u *urls.Model) shortenerOption {
	return func(s *shortener) error {
		s.urls = u
		return nil
	}
}

func WithKafkaProducer(p sarama.AsyncProducer, topic string) shortenerOption {
	return func(s *shortener) error {
		s.producer = p
		s.topic = topic
		return nil
	}
}

func WithRedirectorHost(host string) shortenerOption {
	return func(s *shortener) error {
		s.redirectorHost = host
		return nil
	}
}

func NewShortener(opts ...shortenerOption) (*shortener, error) {
	s := new(shortener)
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

// excludes zero, uppercase 'O', uppercase 'I', and lowercase 'l'
var readerFriendlyCharset = []byte(
	"123456789abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ_-",
)

func generateShortUrl(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = readerFriendlyCharset[rand.Int63()%int64(len(readerFriendlyCharset))]
	}
	return string(b)
}

func (s *shortener) shortenUrl(w http.ResponseWriter, r *http.Request) {
	log.Println("got url shortening request from", r.RemoteAddr)

	_, err := r.Cookie("auth")
	gotDummyCookie := true
	if err != nil {
		switch {
		case errors.Is(err, http.ErrNoCookie):
			gotDummyCookie = false
		default:
			log.Println(
				"caught error during dummy auth cookie processing. error:",
				err,
			)
			res, _ := json.Marshal(&response.Server{
				Status:  response.StatusError,
				Message: "caught error during dummy auth cookie processing",
			})
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(res)
			return
		}
	}

	if !gotDummyCookie {
		s.shortenNoAuth(w, r)
		return
	}

	JWTCookie, err := r.Cookie("JWT")
	if err != nil {
		switch {
		case errors.Is(err, http.ErrNoCookie):
			s.shortenNoAuth(w, r)
		default:
			log.Println(
				"caught error during JWT cookie processing. error:",
				err,
			)
			res, _ := json.Marshal(&response.Server{
				Status:  response.StatusError,
				Message: "caught error during JWT cookie processing",
			})
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(res)
		}
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

	if validationError := tokenInfo.GetError(); validationError != "" {
		log.Printf(
			"invalid jwt from %s. reason: %s\n",
			r.RemoteAddr,
			validationError,
		)

		res, _ := json.Marshal(&response.Server{
			Status:  response.StatusValidationError,
			Message: "couldn't validate JWT",
		})
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(res)
		return
	}

	log.Printf(
		"got valid tokin from %s with user id %s\n ",
		r.RemoteAddr,
		tokenInfo.GetUserId(),
	)

	s.shortenAuth(tokenInfo.GetUserId(), w, r)
}

func (s *shortener) shortenAuth(
	from string,
	w http.ResponseWriter,
	r *http.Request,
) {
	d := json.NewDecoder(r.Body)
	var form authShortenReq
	if err := d.Decode(&form); err != nil {
		log.Println("couldn't decode auth shortening req. reason:", err)
		res, _ := json.Marshal(&response.Server{
			Status:  response.StatusValidationError,
			Message: "Bad shortening form",
		})

		w.WriteHeader(http.StatusBadRequest)
		w.Write(res)
		return
	}

	var shortUrl string
	for {
		shortUrl = generateShortUrl(5)
		// TODO: ctx deadline check
		exists, err := s.urls.CheckExistence(context.TODO(), shortUrl)
		if err != nil {
			log.Println("couldn't check existence of short url. reason:", err)
			continue
		}
		if !exists {
			break
		}
	}

	m, _ := json.Marshal(&response.Shortener{
		From:           from,
		ShortUrl:       shortUrl,
		LongUrl:        form.Url,
		ExpirationDate: time.Now().Add(time.Hour * 24 * 30),
	})

	s.producer.Input() <- &sarama.ProducerMessage{
		Topic: s.topic,
		Value: sarama.ByteEncoder(m),
	}

	res, _ := json.Marshal(&response.Server{
		Status:  response.StatusOK,
		Message: s.redirectorHost + "/" + shortUrl,
	})

	w.WriteHeader(http.StatusOK)
	w.Write(res)
}

func (s *shortener) shortenNoAuth(w http.ResponseWriter, r *http.Request) {
	d := json.NewDecoder(r.Body)
	var form noAuthShortenReq
	if err := d.Decode(&form); err != nil {
		log.Println("couldn't decode no auth shortening req. reason:", err)
		res, _ := json.Marshal(&response.Server{
			Status:  response.StatusValidationError,
			Message: "Bad shortening form",
		})

		w.WriteHeader(http.StatusBadRequest)
		w.Write(res)
		return
	}

	var shortUrl string
	for {
		shortUrl = generateShortUrl(5)
		// TODO: ctx deadline check
		exists, err := s.urls.CheckExistence(context.TODO(), shortUrl)
		if err != nil {
			log.Println("couldn't check existence of short url. reason:", err)
			continue
		}
		if !exists {
			break
		}
	}

	m, _ := json.Marshal(&response.Shortener{
		From:           "",
		ShortUrl:       shortUrl,
		LongUrl:        form.Url,
		ExpirationDate: time.Now().Add(time.Hour * 24 * 30),
	})

	s.producer.Input() <- &sarama.ProducerMessage{
		Topic: s.topic,
		Value: sarama.ByteEncoder(m),
	}

	res, _ := json.Marshal(&response.Server{
		Status:  response.StatusOK,
		Message: s.redirectorHost + "/" + shortUrl,
	})

	w.WriteHeader(http.StatusOK)
	w.Write(res)
}

func handlePreflightRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().
		Add("Access-Control-Allow-Origin", "http://localhost:5173")
	w.Header().Add("Access-Control-Allow-Credentials", "true")
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
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

	rdb := redis.NewClient(&redis.Options{Addr: "redis:6379"})
	u, err := urls.New(
		urls.WithPool(context.TODO(), os.Getenv("POSTGRES_DSN")),
		urls.WithRedis(rdb),
	)
	if err != nil {
		log.Fatalln("couldn't instantiate urls model. reason:", err)
	}

	conf := sarama.NewConfig()
	conf.Producer.RequiredAcks = sarama.WaitForAll
	conf.Producer.Flush.Frequency = 3 * time.Second
	conf.Producer.Return.Errors = false // TODO: handle errors (Dead Letter Queue?)

	p, err := sarama.NewAsyncProducer(
		strings.Split(os.Getenv("KAFKA_BROKERS"), ","),
		conf,
	)
	if err != nil {
		log.Fatalln("couldn't kafka producer. reason:", err)
	}

	s, err := NewShortener(
		WithBlackboxClient(blackbox.NewBlackboxServiceClient(conn)),
		WithUrlsModel(u),
		WithKafkaProducer(p, os.Getenv("KAFKA_STORAGE_TOPIC")),
		WithRedirectorHost(os.Getenv("REDIRECTOR_HOST")),
	)
	if err != nil {
		log.Fatalln("couldn't instantiate shortener. reason:", err)
	}


	http.HandleFunc(
		"OPTIONS /create_short_url",
		http.HandlerFunc(handlePreflightRequest),
	)
	http.HandleFunc(
		"POST /create_short_url",
		middleware.CorsHeaders(http.HandlerFunc(s.shortenUrl)),
	)
	http.ListenAndServe(":8080", nil)
}
