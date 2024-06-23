package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"shortener/internal/redirector"
	"shortener/pkg/middleware"
	"shortener/pkg/models/urls"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).
		With().
		Timestamp().
		Logger()

	rdb := redis.NewClient(&redis.Options{Addr: "redis:6379"})
	u, err := urls.New(
		urls.WithRedis(rdb),
		urls.WithPool(context.TODO(), os.Getenv("POSTGRES_DSN")),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("couldn't instantiate urls model")
	}
	defer u.Close()

	re, err := redirector.New(redirector.WithUrlsModel(u))
	if err != nil {
		log.Fatal().Err(err).Msg("couldn't instantiate redirector")
	}

	c := middleware.RequestTracing(&log)

	mux := http.NewServeMux()
	mux.Handle("GET /", c.ThenFunc(re.Redirect))
	server := http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
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
