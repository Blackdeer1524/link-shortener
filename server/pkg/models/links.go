package models

import (
	"context"
	"fmt"
	"log"
	"shortener/pkg/response"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Links struct {
	pool *pgxpool.Pool
}

type linksOption func(l *Links) error

func WithPool(pool *pgxpool.Pool) linksOption {
	return func(l *Links) error {
		l.pool = pool
		return nil
	}
}

func NewLinks(opts ...linksOption) (*Links, error) {
	l := new(Links)
	for _, opt := range opts {
		if err := opt(l); err != nil {
			return nil, err
		}
	}
	if l.pool == nil {
		return nil, fmt.Errorf("no pool provided")
	}
	return l, nil
}

func (l *Links) Insert(info []*response.Shortener) {
	batch := pgx.Batch{}

	for _, linkInfo := range info {
		batch.Queue(
			`INSERT INTO Links(From, LongLink, ShortLink, ExparationDate) VALUES ($1, $2, $3, $4)`,
			linkInfo.From,
			linkInfo.LongLink,
			linkInfo.ShortLink,
			linkInfo.ExpirationDate,
		)
	}

	res := l.pool.SendBatch(context.Background(), &batch)
	for _, linkInfo := range info {
		if _, err := res.Exec(); err != nil {
			log.Printf(
				"error occured during insert of %s (%s). reason: %s\n",
				linkInfo.LongLink,
				linkInfo.ShortLink,
				err.Error(),
			)
		}
	}

	defer res.Close()
}
