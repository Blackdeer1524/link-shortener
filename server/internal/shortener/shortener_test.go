package shortener

import (
	"bytes"
	context "context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"shortener/proto/blackbox"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/IBM/sarama/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbblackbox_mocks "shortener/mocks/shortener/proto/blackbox"
)

func TestShorteningNoAuth(t *testing.T) {
	u := NewMockUrls(t)
	c := pbblackbox_mocks.NewMockBlackboxServiceClient(t)
	conf := sarama.NewConfig()
	conf.Producer.RequiredAcks = sarama.WaitForAll
	conf.Producer.Flush.Frequency = 500 * time.Millisecond
	conf.Producer.Return.Errors = false
	p := mocks.NewAsyncProducer(t, conf).ExpectInputAndSucceed()

	shortener, err := New(
		WithKafkaProducer(p, "topic"),
		WithUrlsModel(u),
		WithRedirectorHost("host"),
		WithBlackboxClient(c),
	)
	assert.Nil(t, err)

	recorder := httptest.NewRecorder()

	body := noAuthShortenReq{Url: "localhost:8080/longlink"}
	marshalledBody, _ := json.Marshal(&body)

	req, err := http.NewRequest(
		"POST",
		"/create_short_url",
		bytes.NewReader(marshalledBody),
	)
	assert.Nil(t, err)

	u.EXPECT().
		CheckExistence(context.TODO(), mock.AnythingOfType("string")).
		Return(false, nil)
	shortener.ShortenUrl(recorder, req)
	rsp := recorder.Result()

	assert.Equal(t, rsp.StatusCode, http.StatusOK)
}

func TestShorteningAuth(t *testing.T) {
	for _, data := range []struct {
		Name       string
		Expiration int
	}{
		{
			Name:       "30 days",
			Expiration: 30,
		},
		{
			Name:       "90 days",
			Expiration: 90,
		},
		{
			Name:       "365 days",
			Expiration: 365,
		},
	} {
		t.Run(data.Name, func(t *testing.T) {
			u := NewMockUrls(t)
			c := pbblackbox_mocks.NewMockBlackboxServiceClient(t)
			conf := sarama.NewConfig()
			conf.Producer.RequiredAcks = sarama.WaitForAll
			conf.Producer.Flush.Frequency = 500 * time.Millisecond
			conf.Producer.Return.Errors = false
			p := mocks.NewAsyncProducer(t, conf).ExpectInputAndSucceed()

			shortener, err := New(
				WithKafkaProducer(p, "topic"),
				WithUrlsModel(u),
				WithRedirectorHost("host"),
				WithBlackboxClient(c),
			)
			assert.Nil(t, err)

			recorder := httptest.NewRecorder()

			body := authShortenReq{
				Url:        "localhost:8080/longlink",
				Expiration: data.Expiration,
			}
			marshalledBody, _ := json.Marshal(&body)

			req, err := http.NewRequest(
				"POST",
				"/create_short_url",
				bytes.NewReader(marshalledBody),
			)

			token := "token"

			jwtCookie := &http.Cookie{
				Name:     "JWT",
				Value:    token,
				Path:     "/",
				MaxAge:   3600,
				Secure:   true,
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
			}

			req.AddCookie(jwtCookie)
			assert.Nil(t, err)

			c.EXPECT().ValidateToken(context.TODO(), &blackbox.ValidateTokenReq{
				Token: token,
			}).Return(&blackbox.ValidateTokenRsp{
				UserId: "id",
			}, nil)
			u.EXPECT().
				CheckExistence(context.TODO(), mock.AnythingOfType("string")).
				Return(false, nil)

			shortener.ShortenUrl(recorder, req)
			rsp := recorder.Result()

			assert.Equal(t, http.StatusOK, rsp.StatusCode)
		})
	}
}

func TestShorteningAuthUnexpectedExpiration(t *testing.T) {
	u := NewMockUrls(t)
	c := pbblackbox_mocks.NewMockBlackboxServiceClient(t)
	conf := sarama.NewConfig()
	conf.Producer.RequiredAcks = sarama.WaitForAll
	conf.Producer.Flush.Frequency = 500 * time.Millisecond
	conf.Producer.Return.Errors = false
	p := mocks.NewAsyncProducer(t, conf)

	shortener, err := New(
		WithKafkaProducer(p, "topic"),
		WithUrlsModel(u),
		WithRedirectorHost("host"),
		WithBlackboxClient(c),
	)
	assert.Nil(t, err)

	recorder := httptest.NewRecorder()

	body := authShortenReq{
		Url:        "localhost:8080/longlink",
		Expiration: 40,
	}
	marshalledBody, _ := json.Marshal(&body)

	req, err := http.NewRequest(
		"POST",
		"/create_short_url",
		bytes.NewReader(marshalledBody),
	)

	token := "token"

	jwtCookie := &http.Cookie{
		Name:     "JWT",
		Value:    token,
		Path:     "/",
		MaxAge:   3600,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}

	req.AddCookie(jwtCookie)
	assert.Nil(t, err)

	c.EXPECT().ValidateToken(context.TODO(), &blackbox.ValidateTokenReq{
		Token: token,
	}).Return(&blackbox.ValidateTokenRsp{
		UserId: "id",
	}, nil)

	shortener.ShortenUrl(recorder, req)
	rsp := recorder.Result()

	assert.Equal(t, http.StatusBadRequest, rsp.StatusCode)
}

func TestShorteningAuthBrokenToken(t *testing.T) {
	u := NewMockUrls(t)
	c := pbblackbox_mocks.NewMockBlackboxServiceClient(t)
	conf := sarama.NewConfig()
	conf.Producer.RequiredAcks = sarama.WaitForAll
	conf.Producer.Flush.Frequency = 500 * time.Millisecond
	conf.Producer.Return.Errors = false
	p := mocks.NewAsyncProducer(t, conf)

	shortener, err := New(
		WithKafkaProducer(p, "topic"),
		WithUrlsModel(u),
		WithRedirectorHost("host"),
		WithBlackboxClient(c),
	)
	assert.Nil(t, err)

	recorder := httptest.NewRecorder()

	body := authShortenReq{
		Url:        "localhost:8080/longlink",
		Expiration: 90,
	}
	marshalledBody, _ := json.Marshal(&body)

	req, err := http.NewRequest(
		"POST",
		"/create_short_url",
		bytes.NewReader(marshalledBody),
	)

	token := "token"

	jwtCookie := &http.Cookie{
		Name:     "JWT",
		Value:    token,
		Path:     "/",
		MaxAge:   3600,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}

	req.AddCookie(jwtCookie)
	assert.Nil(t, err)

	c.EXPECT().ValidateToken(context.TODO(), &blackbox.ValidateTokenReq{
		Token: token,
	}).Return(nil, status.Errorf(
		codes.InvalidArgument,
		"invalid token",
	))

	shortener.ShortenUrl(recorder, req)
	rsp := recorder.Result()

	assert.Equal(t, http.StatusForbidden, rsp.StatusCode)
}
