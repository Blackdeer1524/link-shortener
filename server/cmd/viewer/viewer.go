package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"shortener/internal/viewer"
	"shortener/pkg/middleware"
	"shortener/pkg/models/urls"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pbblackbox "shortener/proto/blackbox"
)

func main() {
	conn, err := grpc.NewClient(
		"blackbox:8080",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalln("couldn't dial blackbox service. error:", err)
	}
	defer conn.Close()
	c := pbblackbox.NewBlackboxServiceClient(conn)

	rdb := redis.NewClient(&redis.Options{Addr: "redis:6379"})

	u, err := urls.New(
		urls.WithPool(context.TODO(), os.Getenv("POSTGRES_DSN")),
		urls.WithRedis(rdb),
	)
	if err != nil {
		log.Fatalln("couldn't create urls model. error:", err)
	}

	v, err := viewer.New(
		viewer.WithUrls(u),
		viewer.WithBlackboxClient(c),
		viewer.WithRedirectorHost(os.Getenv("REDIRECTOR_HOST")),
	)
	if err != nil {
		log.Fatalln("couldn't create urls model. error:", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc(
		"OPTIONS /history",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().
				Add("Access-Control-Allow-Origin", "http://localhost:8001")
			w.Header().Add("Access-Control-Allow-Credentials", "true")
		}),
	)
	mux.HandleFunc(
		"GET /history",
		middleware.CorsHeaders(http.HandlerFunc(v.HandleHistory)),
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
			log.Printf("error occured on Shutdown():%v\n", err)
		}
	}()

	err = server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("error during shutdown:%v\n", err)
	}
}
