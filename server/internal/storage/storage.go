package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"shortener/pkg/responses"
	"time"

	"github.com/IBM/sarama"
)

type Urls interface {
	Insert(ctx context.Context, rr []*responses.Shortener)
}

type Users interface {
	Insert(ctx context.Context, rr []*responses.Authenticator)
}

type GroupHandler struct {
	ctx context.Context

	urlsTopic  string
	usersTopic string

	users Users
	urls  Urls
}

type groupHandlerOption func(*GroupHandler) error

func WithContext(ctx context.Context) groupHandlerOption {
	return func(h *GroupHandler) error {
		h.ctx = ctx
		return nil
	}
}

func WithUrlsModel(u Urls) groupHandlerOption {
	return func(h *GroupHandler) error {
		h.urls = u
		return nil
	}
}

func WithUsersModel(u Users) groupHandlerOption {
	return func(h *GroupHandler) error {
		h.users = u
		return nil
	}
}

func WithUrlsTopic(t string) groupHandlerOption {
	return func(h *GroupHandler) error {
		h.urlsTopic = t
		return nil
	}
}

func WithUsersTopic(t string) groupHandlerOption {
	return func(h *GroupHandler) error {
		h.usersTopic = t
		return nil
	}
}

func New(opts ...groupHandlerOption) (*GroupHandler, error) {
	g := new(GroupHandler)
	for _, opt := range opts {
		if err := opt(g); err != nil {
			return nil, err
		}
	}
	if g.ctx == nil {
		return nil, fmt.Errorf("no context provided")
	}
	if g.urlsTopic == "" {
		return nil, fmt.Errorf("no urls topic provided")
	}
	if g.usersTopic == "" {
		return nil, fmt.Errorf("no users topic provided")
	}
	if g.urls == nil {
		return nil, fmt.Errorf("no urls model provided")
	}
	if g.users == nil {
		return nil, fmt.Errorf("no users model provided")
	}

	return g, nil
}

func (h *GroupHandler) Setup(sess sarama.ConsumerGroupSession) error {
	log.Printf("%s is consuming %v\n", sess.MemberID(), sess.Claims())
	return nil
}

func (h *GroupHandler) Cleanup(sess sarama.ConsumerGroupSession) error {
	log.Printf("%s began cleanup %v\n", sess.MemberID(), sess.Claims())
	sess.Commit()
	return nil
}

func (h *GroupHandler) ConsumeClaim(
	sess sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim,
) error {
	if claim.Topic() == h.usersTopic {
		return h.handleUsers(sess, claim)
	} else if claim.Topic() == h.urlsTopic {
		return h.handleUrls(sess, claim)
	} else {
		return fmt.Errorf("unknown topic: %s", claim.Topic())
	}
}

func (h *GroupHandler) handleUrls(
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
			if len(messageBatch) == 0 {
				continue
			}
			var rr []*responses.Shortener
			for _, mes := range messageBatch {
				var rsp responses.Shortener
				if err := json.Unmarshal(mes.Value, &rsp); err != nil {
					log.Println("couldn't unmarshal message. error:", err)
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

func (h *GroupHandler) handleUsers(
	sess sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim,
) error {
	ticker := time.NewTicker(time.Millisecond * 500)
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
			if len(messageBatch) == 0 {
				continue
			}
			var rr []*responses.Authenticator
			for _, mes := range messageBatch {
				var rsp responses.Authenticator
				if err := json.Unmarshal(mes.Value, &rsp); err != nil {
					log.Println("couldn't unmarshal message. error:", err)
					continue
				}
				rr = append(rr, &rsp)
			}
			h.users.Insert(context.TODO(), rr)
			for _, mes := range messageBatch {
				sess.MarkMessage(mes, "")
			}
			messageBatch = messageBatch[:0]
		}
	}
}
