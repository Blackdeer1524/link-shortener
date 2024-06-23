package main

import (
	"context"
	"os"
	"os/signal"
	"shortener/internal/storage"
	"shortener/pkg/models/urls"
	"shortener/pkg/models/users"
	"strings"
	"syscall"

	"github.com/IBM/sarama"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).
		With().
		Timestamp().
		Logger()

	conf := sarama.NewConfig()
	conf.Consumer.Offsets.AutoCommit.Enable = false
	conf.Consumer.Offsets.Initial = sarama.OffsetOldest
	if err := conf.Validate(); err != nil {
		log.Fatal().Err(err).Msg("invalid kafka config")
	}
	group, err := sarama.NewConsumerGroup(
		strings.Split(os.Getenv("KAFKA_BROKERS"), ","),
		"storage",
		conf,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("couldn't start consuming kafka topic")
	}
	defer group.Close()
	log.Info().Msg("successfully instantiated topic consumer group")

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
		log.Fatal().Msg("couldn't instantiate urls model")
	}
	defer u.Close()
	log.Info().Msg("successfully instantiated urls model")

	users, err := users.NewUsers(
		users.WithPool(context.TODO(), os.Getenv("POSTGRES_DSN")),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("couldn't instantiate users model")
	}
	defer users.Close()
	log.Info().Msg("successfully instantiated users model")

	h, err := storage.New(
		storage.WithLogger(&log),
		storage.WithContext(ctx),
		storage.WithUrlsTopic(os.Getenv("KAFKA_URLS_TOPIC")),
		storage.WithUrlsModel(u),
		storage.WithUsersTopic(os.Getenv("KAFKA_USERS_TOPIC")),
		storage.WithUsersModel(users),
	)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("couldn't instantiate consumer group's handler. error:")
	}
	log.Info().Msg("successfully instantiated group handler")

	go func() {
		<-ctx.Done()
		if err = group.Close(); err != nil {
			log.Error().Err(err).Msg("error occured on group Close()")
		}
	}()

	err = group.Consume(
		ctx,
		[]string{os.Getenv("KAFKA_URLS_TOPIC"), os.Getenv("KAFKA_USERS_TOPIC")},
		h,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("group.Consume() exited with an error")
	}
}
