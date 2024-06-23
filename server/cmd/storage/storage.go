package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"shortener/internal/storage"
	"shortener/pkg/models/urls"
	"shortener/pkg/models/users"
	"strings"
	"syscall"

	"github.com/IBM/sarama"
	"github.com/redis/go-redis/v9"
)

func main() {
	conf := sarama.NewConfig()
	conf.Consumer.Offsets.AutoCommit.Enable = false
	conf.Consumer.Offsets.Initial = sarama.OffsetOldest
	if err := conf.Validate(); err != nil {
		log.Fatalln("invalid kafka config:", err)
	}

	group, err := sarama.NewConsumerGroup(
		strings.Split(os.Getenv("KAFKA_BROKERS"), ","),
		"storage",
		conf,
	)
	if err != nil {
		log.Fatalln("couldn't start consuming kafka topic. error:", err)
	}
	defer group.Close()

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGTERM,
		syscall.SIGINT,
	)
	defer cancel()

	rdb := redis.NewClient(&redis.Options{Addr: "redis:6379"})

	u, err := urls.New(
		urls.WithPool(context.TODO(), os.Getenv("POSTGRES_DSN")),
		urls.WithRedis(rdb),
	)
	if err != nil {
		log.Fatalln("couldn't instantiate urls model. error:", err)
	}
	defer u.Close()

	users, err := users.NewUsers(
		users.WithPool(context.TODO(), os.Getenv("POSTGRES_DSN")),
	)
	if err != nil {
		log.Fatalln("couldn't instantiate users model. error:", err)
	}
	defer users.Close()

	h, err := storage.New(
		storage.WithContext(ctx),
		storage.WithUrlsTopic(os.Getenv("KAFKA_URLS_TOPIC")),
		storage.WithUrlsModel(u),
		storage.WithUsersTopic(os.Getenv("KAFKA_USERS_TOPIC")),
		storage.WithUsersModel(users),
	)
	if err != nil {
		log.Fatalln("couldn't instantiate consumer group's handler. error:", err)
	}

	go func() {
		<-ctx.Done()
		if err = group.Close(); err != nil {
			log.Printf("error occured on group Close():%v\n", err)
		}
	}()

	err = group.Consume(
		ctx,
		[]string{os.Getenv("KAFKA_URLS_TOPIC"), os.Getenv("KAFKA_USERS_TOPIC")},
		h,
	)
	if err != nil {
		log.Fatalln("topics consumption error:", err)
	}
}
