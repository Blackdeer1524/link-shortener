package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"shortener/pkg/responses"
	"time"

	"github.com/IBM/sarama"
	"github.com/rs/zerolog"
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

	log *zerolog.Logger
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

func WithLogger(l *zerolog.Logger) groupHandlerOption {
	return func(h *GroupHandler) error {
		h.log = l
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
	if g.log == nil {
		return nil, fmt.Errorf("no logger provided")
	}

	return g, nil
}

func (h *GroupHandler) Setup(sess sarama.ConsumerGroupSession) error {
	h.log.Info().
		Str("member_id", sess.MemberID()).
		Str("claims", fmt.Sprintf("%v", sess.Claims())).
		Msg("began consumption")
	return nil
}

func (h *GroupHandler) Cleanup(sess sarama.ConsumerGroupSession) error {
	h.log.Info().
		Str("member_id", sess.MemberID()).
		Str("claims", fmt.Sprintf("%v", sess.Claims())).
		Msg("began cleanup")
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

	h.log.Info().Msg("waiting for url messages")
	var messageBatch []*sarama.ConsumerMessage
	for {
		select {
		case <-h.ctx.Done():
			h.log.Info().Msg("got cancellation signal in urls handler")
			return nil
		case mes, isOpen := <-claim.Messages():
			if !isOpen {
				h.log.Info().Msg("url messages channel had been closed")
				return nil
			}
			h.log.Info().Msg("got url message")
			messageBatch = append(messageBatch, mes)
		case <-ticker.C:
			if len(messageBatch) == 0 {
				continue
			}
			h.log.Info().
				Int("batch_size", len(messageBatch)).
				Msg("processing url messages batch")
			var rr []*responses.Shortener
			for _, mes := range messageBatch {
				var rsp responses.Shortener
				if err := json.Unmarshal(mes.Value, &rsp); err != nil {
					h.log.Error().Err(err).Msg("couldn't unmarshal url message")
					continue
				}
				rr = append(rr, &rsp)
			}
			h.log.Info().Msg("began inserting message batch into database")
			h.urls.Insert(context.TODO(), rr)
			for _, mes := range messageBatch {
				sess.MarkMessage(mes, "")
			}
			h.log.Info().Msg("marked message batch as processed")
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

	h.log.Info().Msg("waiting for new users messages")
	var messageBatch []*sarama.ConsumerMessage
	for {
		select {
		case <-h.ctx.Done():
			h.log.Info().Msg("got cancellation signal in new users handler")
			return nil
		case mes, isOpen := <-claim.Messages():
			if !isOpen {
				h.log.Info().Msg("users messages channel had been closed")
				return nil
			}
			h.log.Info().Msg("got new users message")
			messageBatch = append(messageBatch, mes)
		case <-ticker.C:
			if len(messageBatch) == 0 {
				continue
			}
			h.log.Info().
				Int("batch_size", len(messageBatch)).
				Msg("processing new users messages batch")
			var rr []*responses.Authenticator
			for _, mes := range messageBatch {
				var rsp responses.Authenticator
				if err := json.Unmarshal(mes.Value, &rsp); err != nil {
					h.log.Error().
						Err(err).
						Msg("couldn't unmarshal new user message")
					continue
				}
				rr = append(rr, &rsp)
			}
			h.log.Info().Msg("began inserting message batch into database")
			h.users.Insert(context.TODO(), rr)
			for _, mes := range messageBatch {
				sess.MarkMessage(mes, "")
			}
			h.log.Info().Msg("marked message batch as processed")
			messageBatch = messageBatch[:0]
		}
	}
}
