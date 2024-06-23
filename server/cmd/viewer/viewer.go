package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"shortener/internal/viewer"
	"shortener/pkg/middleware"
	"shortener/pkg/models/urls"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pbblackbox "shortener/proto/blackbox"
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
	c := pbblackbox.NewBlackboxServiceClient(conn)
	log.Info().Msg("instantiated blackbox client")

	rdb := redis.NewClient(&redis.Options{Addr: "redis:6379"})
	u, err := urls.New(
		urls.WithPool(context.TODO(), os.Getenv("POSTGRES_DSN")),
		urls.WithRedis(rdb),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("couldn't instantiate urls model")
	}
	log.Info().Msg("instantiated urls model")

	v, err := viewer.New(
		viewer.WithUrls(u),
		viewer.WithBlackboxClient(c),
		viewer.WithRedirectorHost(os.Getenv("REDIRECTOR_HOST")),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("couldn't instantiate urls model")
	}
	log.Info().Msg("instantiated viewer service")

	m := middleware.RequestTracing(&log)
	mux := http.NewServeMux()
	mux.HandleFunc(
		"OPTIONS /history",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().
				Add("Access-Control-Allow-Origin", "http://localhost:8001")
			w.Header().Add("Access-Control-Allow-Credentials", "true")
		}),
	)
	mux.Handle(
		"GET /history",
		m.Append(middleware.CorsHeaders).ThenFunc(v.HandleHistory),
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
			log.Err(err).Msg("error occured on Shutdown()")
		}
	}()

	log.Info().Msg("started listening")
	err = server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal().Err(err).Msg("error during shutdown")
	}
}
