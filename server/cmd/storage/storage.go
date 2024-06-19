package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/signal"
	"shortener/pkg/models"
	"shortener/pkg/response"
	"syscall"
	"time"

	"github.com/IBM/sarama"
)

type groupHandler struct {
	ctx   context.Context
	links *models.Links // TODO: replace with interface
}

type groupHandlerOption func(*groupHandler) error

func WithContext(ctx context.Context) groupHandlerOption {
	return func(h *groupHandler) error {
		h.ctx = ctx
		return nil
	}
}


func WithLinksModel(l *models.Links) groupHandlerOption {
	return func(h *groupHandler) error {
		h.ctx = ctx
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
	return nil
}

func (h *groupHandler) Cleanup(sess sarama.ConsumerGroupSession) error {
	sess.Commit()
	return nil
}

func (h *groupHandler) ConsumeClaim(
	sess sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim,
) error {
	ticker := time.NewTicker(time.Millisecond * 300)
	defer ticker.Stop()

	var messageBatch []*response.Shortener

	for {
		select {
		case mes, isOpen := <-claim.Messages():
			if !isOpen {
				return nil
			}
			var rsp response.Shortener
			if err := json.Unmarshal(mes.Value, &rsp); err != nil {
				log.Println("couldn't unmarshal message. reason:", err)
				continue
			}
			messageBatch = append(messageBatch, &rsp)
		case <-ticker.C:

		case <-h.ctx.Done():
			return nil
		}
	}
}

func main() {
	conf := sarama.NewConfig()
	conf.Consumer.Offsets.AutoCommit.Enable = false
	conf.Consumer.Offsets.Initial = sarama.OffsetOldest

	group, err := sarama.NewConsumerGroup(
		[]string{"kafka:9092"},
		"storage",
		conf,
	)
	defer group.Close()

	if err != nil {
		log.Fatalln("couldn't start consuming kafka topic. reason:", err)
	}

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGTERM,
		syscall.SIGINT,
	)
	defer cancel()

	h, err := NewGroupHandler(WithContext(ctx))
	if err != nil {
		log.Fatalln("couldn't create consumer group's handler. reason:", err)
	}

	group.Consume(ctx, []string{"urls"}, h)
}
