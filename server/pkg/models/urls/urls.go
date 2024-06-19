package urls

import (
	"context"
	"errors"
	"fmt"
	"log"
	"shortener/pkg/response"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Model struct {
	pool *pgxpool.Pool
	rdb  *redis.Client
}

type urlsOption func(u *Model) error

func WithPool(ctx context.Context, dsn string) urlsOption {
	return func(l *Model) error {
		pool, err := pgxpool.New(ctx, dsn)
		if err != nil {
			return err
		}

		if err := pool.Ping(ctx); err != nil {
			return err
		}

		l.pool = pool
		return nil
	}
}

func WithRedis(rdb *redis.Client) urlsOption {
	return func(u *Model) error {
		u.rdb = rdb
		return nil
	}
}

func New(opts ...urlsOption) (*Model, error) {
	u := new(Model)
	for _, opt := range opts {
		if err := opt(u); err != nil {
			return nil, err
		}
	}
	if u.pool == nil {
		return nil, fmt.Errorf("no pool provided")
	}
	if u.rdb == nil {
		return nil, errors.New("no redis client provided")
	}

	return u, nil
}

func (u *Model) CheckExistence(
	ctx context.Context,
	shortUrl string,
) (bool, error) {
	var res bool

	err := u.rdb.Get(ctx, shortUrl).Err()
	if err == nil {
		return true, nil
	} else if !errors.Is(err, redis.Nil) {
		log.Println("couldn't get result from redis. reason:", err)
	}

	err = u.pool.QueryRow(
		ctx,
		`SELECT EXISTS(SELECT 1 FROM Urls WHERE ShortUrl = $1)`,
		shortUrl,
	).Scan(&res)
	return res, err
}

var ErrNotFound = errors.New("url not found")

func (u *Model) GetLongUrl(
	ctx context.Context,
	shortUrl string,
) (string, error) {
	cacheRes := u.rdb.Get(ctx, shortUrl)
	if cacheRes.Err() == nil {
		longUrl, err := cacheRes.Result()
		if err == nil {
			return longUrl, nil
		} else {
			log.Printf("couldn't extract value from redis result. reason:", err)
		}
	} else if cacheRes.Err() != redis.Nil {
		log.Println("couldn't get value by key from redis. reason:", cacheRes.Err())
	}

	var longUrl string
	err := u.pool.QueryRow(
		ctx,
		`SELECT LongUrl from Urls where Urls.ShortUrl = $1`,
		shortUrl,
	).Scan(&longUrl)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}

	return longUrl, err
}

func (u *Model) Insert(ctx context.Context, rr []*response.Shortener) {
	batch := pgx.Batch{}

	for _, urlInfo := range rr {
		batch.Queue(
			`INSERT INTO Urls(ShortUrl, LongUrl, UserId, ExpirationDate) VALUES ($1, $2, $3, $4)`,
			urlInfo.ShortUrl,
			urlInfo.LongUrl,
			urlInfo.From,
			urlInfo.ExpirationDate,
		)
	}

	res := u.pool.SendBatch(ctx, &batch)
	for _, urlInfo := range rr {
		if _, err := res.Exec(); err != nil {
			log.Printf(
				"error occured during insert of %s (%s). reason: %s\n",
				urlInfo.LongUrl,
				urlInfo.ShortUrl,
				err.Error(),
			)
		} else {
			err := u.rdb.Set(ctx, urlInfo.ShortUrl, urlInfo.LongUrl, time.Hour*24).Err()
			if err != nil {
				log.Println("coulnd't put short url into cache. reason:", err)
			}
		}
	}

	defer res.Close()
}

func (u *Model) Close() {
	u.pool.Close()
}
