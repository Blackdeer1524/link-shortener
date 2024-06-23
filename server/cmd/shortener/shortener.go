package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"shortener/internal/shortener"
	"shortener/pkg/middleware"
	"shortener/pkg/models/urls"
	"shortener/proto/blackbox"
	"strings"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).
		With().
		Timestamp().
		Logger()

	conn, err := grpc.NewClient(
		"blackbox:8080",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("couldn't dial auth service")
	}
	log.Info().Msg("successfully instantiated blackbox client")
	defer conn.Close()

	rdb := redis.NewClient(&redis.Options{Addr: "redis:6379"})

	u, err := urls.New(
		urls.WithPool(context.TODO(), os.Getenv("POSTGRES_DSN")),
		urls.WithRedis(rdb),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("couldn't instantiate urls model")
	}
	defer u.Close()
	log.Info().Msg("successfully instantiated user model")

	conf := sarama.NewConfig()
	conf.Producer.RequiredAcks = sarama.WaitForAll
	conf.Producer.Flush.Frequency = 500 * time.Millisecond
	conf.Producer.Return.Errors = false
	if err = conf.Validate(); err != nil {
		log.Fatal().Err(err).Msg("invalid kafka config")
	}
	p, err := sarama.NewAsyncProducer(
		strings.Split(os.Getenv("KAFKA_BROKERS"), ","),
		conf,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("couldn't instantiate kafka producer")
	}
	defer p.Close()
	log.Info().Msg("successfully instantiated topic producer")

	log.Info().Msg("instantiating shortener")
	s, err := shortener.New(
		shortener.WithUrlsModel(u),
		shortener.WithBlackboxClient(blackbox.NewBlackboxServiceClient(conn)),
		shortener.WithKafkaProducer(p, os.Getenv("KAFKA_URLS_TOPIC")),
		shortener.WithRedirectorHost(os.Getenv("REDIRECTOR_HOST")),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("couldn't instantiate shortener")
	}
	log.Info().Msg("successfully instantiated shortener")

	mux := http.NewServeMux()
	mux.HandleFunc(
		"OPTIONS /create_short_url",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().
				Add("Access-Control-Allow-Origin", "http://localhost:8001")
			w.Header().Add("Access-Control-Allow-Credentials", "true")
			w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
		}),
	)

	m := middleware.RequestTracing(&log)
	mux.Handle(
		"POST /create_short_url",
		m.Append(middleware.CorsHeaders).ThenFunc(s.ShortenUrl),
	)

	server := http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  time.Minute,
	}

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGTERM,
		syscall.SIGINT,
	)
	defer cancel()

	go func() {
		<-ctx.Done()
		err = server.Shutdown(context.TODO())
		if err != nil {
			log.Error().Err(err).Msg("error occured on Shutdown()")
		}
	}()

	log.Info().Msg("listening for connections")
	err = server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal().Err(err).Msg("error during shutdown")
	}
}
