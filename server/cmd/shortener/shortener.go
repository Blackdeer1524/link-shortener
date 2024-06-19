package main

import (
	"context"
	"errors"
	"log"
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
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.NewClient(
		"blackbox:8080",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalln("couldn't dial auth service. error:", err)
	}
	defer conn.Close()

	rdb := redis.NewClient(&redis.Options{Addr: "redis:6379"})
	u, err := urls.New(
		urls.WithPool(context.TODO(), os.Getenv("POSTGRES_DSN")),
		urls.WithRedis(rdb),
	)
	if err != nil {
		log.Fatalln("couldn't instantiate urls model. error:", err)
	}
	defer u.Close()

	conf := sarama.NewConfig()
	conf.Producer.RequiredAcks = sarama.WaitForAll
	conf.Producer.Flush.Frequency = 500 * time.Millisecond
	conf.Producer.Return.Errors = false
	if err = conf.Validate(); err != nil {
		log.Fatalln("invalid kafka config:", err)
	}

	p, err := sarama.NewAsyncProducer(
		strings.Split(os.Getenv("KAFKA_BROKERS"), ","),
		conf,
	)
	if err != nil {
		log.Fatalln("couldn't create kafka producer. error:", err)
	}
	defer p.Close()

	s, err := shortener.New(
		shortener.WithBlackboxClient(blackbox.NewBlackboxServiceClient(conn)),
		shortener.WithUrlsModel(u),
		shortener.WithKafkaProducer(p, os.Getenv("KAFKA_URLS_TOPIC")),
		shortener.WithRedirectorHost(os.Getenv("REDIRECTOR_HOST")),
	)
	if err != nil {
		log.Fatalln("couldn't instantiate shortener. error:", err)
	}

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
	mux.HandleFunc(
		"POST /create_short_url",
		middleware.CorsHeaders(http.HandlerFunc(s.ShortenUrl)),
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
