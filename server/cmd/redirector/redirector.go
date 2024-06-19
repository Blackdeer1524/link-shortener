package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"shortener/internal/redirector"
	"shortener/pkg/models/urls"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
)

func main() {
	rdb := redis.NewClient(&redis.Options{Addr: "redis:6379"})

	u, err := urls.New(
		urls.WithRedis(rdb),
		urls.WithPool(context.TODO(), os.Getenv("POSTGRES_DSN")),
	)
	if err != nil {
		log.Fatalln("couldn't instantiate urls model. error:", err)
	}
	defer u.Close()

	re, err := redirector.New(redirector.WithUrlsModel(u))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", re.Redirect)

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
			log.Printf("error occured on Shutdown():%v\n", err)
		}
	}()

	err = server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("error during shutdown:%v\n", err)
	}
}
