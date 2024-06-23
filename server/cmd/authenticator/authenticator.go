package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"shortener/internal/authenticator"
	"shortener/pkg/middleware"
	"shortener/pkg/models/users"
	"shortener/proto/blackbox"
	"strings"
	"syscall"
	"time"

	"github.com/IBM/sarama"
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
		log.Fatal().Err(err).Msg("couldn't dial blackbox service")
	}
	defer conn.Close()

	box := blackbox.NewBlackboxServiceClient(conn)

	usersModel, err := users.NewUsers(
		users.WithPool(context.TODO(), os.Getenv("POSTGRES_DSN")),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("couldn't instantiate users model")
	}
	defer usersModel.Close()

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

	a, err := authenticator.New(
		authenticator.WithUsersDB(usersModel),
		authenticator.WithBlackboxClient(box),
		authenticator.WithProducer(os.Getenv("KAFKA_USERS_TOPIC"), p),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("couldn't instantiate authenticator")
	}

	stdMiddleware := middleware.RequestTracing(&log)

	mux := http.NewServeMux()
	mux.Handle(
		"OPTIONS /signup",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().
				Add("Access-Control-Allow-Origin", "http://localhost:8001")
			w.Header().Add("Access-Control-Allow-Credentials", "true")
			w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
		}),
	)
	mux.Handle(
		"OPTIONS /login",
		stdMiddleware.ThenFunc(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().
					Add("Access-Control-Allow-Origin", "http://localhost:8001")
				w.Header().Add("Access-Control-Allow-Credentials", "true")
				w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
			}),
		),
	)
	mux.Handle(
		"POST /signup",
		stdMiddleware.Append(middleware.CorsHeaders).ThenFunc(a.Register),
	)
	mux.Handle(
		"POST /login",
		stdMiddleware.
			Append(middleware.CorsHeaders).
			ThenFunc(a.Login),
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

	err = server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal().Err(err).Msg("error during shutdown")
	}
}
