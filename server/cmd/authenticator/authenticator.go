package main

import (
	"context"
	"errors"
	"log"
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
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

	c := blackbox.NewBlackboxServiceClient(conn)

	usersModel, err := users.NewUsers(
		users.WithPool(context.TODO(), os.Getenv("POSTGRES_DSN")),
	)
	if err != nil {
		log.Fatalln("couldn't instantiate users model. error:", err)
	}
	defer usersModel.Close()

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

	authenticator, err := authenticator.New(
		authenticator.WithUsersDB(usersModel),
		authenticator.WithBlackboxClient(c),
		authenticator.WithProducer(os.Getenv("KAFKA_USERS_TOPIC"), p),
	)
	if err != nil {
		log.Fatalln("couldn't create authenticator. error:", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc(
		"OPTIONS /signup",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().
				Add("Access-Control-Allow-Origin", "http://localhost:8001")
			w.Header().Add("Access-Control-Allow-Credentials", "true")
			w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
		}),
	)
	mux.HandleFunc(
		"OPTIONS /login",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().
				Add("Access-Control-Allow-Origin", "http://localhost:8001")
			w.Header().Add("Access-Control-Allow-Credentials", "true")
			w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
		}),
	)
	mux.HandleFunc(
		"POST /signup",
		middleware.CorsHeaders(http.HandlerFunc(authenticator.Register)),
	)
	mux.HandleFunc(
		"POST /login",
		middleware.CorsHeaders(http.HandlerFunc(authenticator.Login)),
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
