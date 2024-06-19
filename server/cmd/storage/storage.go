package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"shortener/pkg/models/urls"
	"shortener/pkg/response"
	"strings"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	"github.com/redis/go-redis/v9"
)

type groupHandler struct {
	ctx  context.Context
	urls *urls.Model // TODO: replace with interface
}

type groupHandlerOption func(*groupHandler) error

func WithContext(ctx context.Context) groupHandlerOption {
	return func(h *groupHandler) error {
		h.ctx = ctx
		return nil
	}
}

func WithUrlsModel(l *urls.Model) groupHandlerOption {
	return func(h *groupHandler) error {
		h.urls = l
		return nil
	}
}

func NewGroupHandler(opts ...groupHandlerOption) (*groupHandler, error) {
	g := new(groupHandler)
	for _, opt := range opts {
		if err := opt(g); err != nil {
			return nil, err
		}
	}
	if g.ctx == nil {
		return nil, fmt.Errorf("no context provided")
	}
	return g, nil
}

func (h *groupHandler) Setup(sess sarama.ConsumerGroupSession) error {
	log.Printf("%s is consuming %v\n", sess.MemberID(), sess.Claims())
	return nil
}

func (h *groupHandler) Cleanup(sess sarama.ConsumerGroupSession) error {
	log.Printf("%s is began cleanup\n", sess.MemberID())
	sess.Commit()
	return nil
}

func (h *groupHandler) ConsumeClaim(
	sess sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim,
) error {
	ticker := time.NewTicker(time.Millisecond * 300)
	defer ticker.Stop()

	var messageBatch []*sarama.ConsumerMessage
	for {
		select {
		case <-h.ctx.Done():
			return nil
		case mes, isOpen := <-claim.Messages():
			if !isOpen {
				return nil
			}
			messageBatch = append(messageBatch, mes)
		case <-ticker.C:
			var rr []*response.Shortener
			for _, mes := range messageBatch {
				var rsp response.Shortener
				if err := json.Unmarshal(mes.Value, &rsp); err != nil {
					log.Println("couldn't unmarshal message. reason:", err)
					continue
				}
				rr = append(rr, &rsp)
			}
			h.urls.Insert(context.TODO(), rr)
			for _, mes := range messageBatch {
				sess.MarkMessage(mes, "")
			}
			messageBatch = messageBatch[:0]
		}
	}
}

func main() {
	conf := sarama.NewConfig()
	conf.Consumer.Offsets.AutoCommit.Enable = false
	conf.Consumer.Offsets.Initial = sarama.OffsetOldest

	group, err := sarama.NewConsumerGroup(
		strings.Split(os.Getenv("KAFKA_BROKERS"), ","),
		"storage",
		conf,
	)
	if err != nil {
		log.Fatalln("couldn't start consuming kafka topic. reason:", err)
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
		log.Fatalln("couldn't instantiate urls model. reason:", err)
	}

	h, err := NewGroupHandler(WithContext(ctx), WithUrlsModel(u))
	if err != nil {
		log.Fatalln("couldn't create consumer group's handler. reason:", err)
	}

	go func() {
		<-ctx.Done()
		group.Close()
	}()

	if err := group.Consume(ctx, []string{os.Getenv("KAFKA_STORAGE_TOPIC")}, h); err != nil {
		log.Fatalln("topics consumption error:", err)
	}
}
