package authenticator

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"shortener/pkg/middleware"
	"shortener/pkg/models/users"
	"shortener/pkg/responses"
	"shortener/proto/blackbox"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/IBM/sarama/mocks"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	pbblackbox_mocks "shortener/mocks/shortener/proto/blackbox"
)

func TestNewSignUp(t *testing.T) {
	conf := sarama.NewConfig()
	conf.Producer.RequiredAcks = sarama.WaitForAll
	conf.Producer.Flush.Frequency = 500 * time.Millisecond
	conf.Producer.Return.Errors = false
	p := mocks.NewAsyncProducer(t, conf).ExpectInputAndSucceed()

	email := "some@mail.ru"

	uMock := NewMockUsers(t)
	uMock.EXPECT().
		CheckExistence(context.TODO(), email).
		Return(false, nil)

	userId := "id"
	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.MapClaims{
			"sub": userId,
		})
	signedToken, err := token.SignedString([]byte("secret"))

	cMock := pbblackbox_mocks.NewMockBlackboxServiceClient(t)
	cMock.EXPECT().
		IssueToken(context.TODO(), mock.AnythingOfType("*blackbox.IssueTokenReq")).
		Return(&blackbox.IssueTokenRsp{
			Token: signedToken,
		}, nil)

	authenticator, err := New(
		WithProducer("topic", p),
		WithUsersDB(uMock),
		WithBlackboxClient(cMock),
	)
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	req, _ := json.Marshal(&registerRequest{
		Name:     "name",
		Email:    email,
		Password: "password",
	})

	request, err := http.NewRequest("POST", "/signup", bytes.NewReader(req))
	assert.Nil(t, err)

	middleware.CorsHeaders(http.HandlerFunc(authenticator.Register)).
		ServeHTTP(rr, request)

	rsp := rr.Result()

	assert.Equal(t, http.StatusOK, rsp.StatusCode)

	var body responses.Server
	err = json.NewDecoder(rsp.Body).Decode(&body)
	assert.Nil(t, err)
}

func TestSignUpUserAlreadyExists(t *testing.T) {
	conf := sarama.NewConfig()
	conf.Producer.RequiredAcks = sarama.WaitForAll
	conf.Producer.Flush.Frequency = 500 * time.Millisecond
	conf.Producer.Return.Errors = false
	p := mocks.NewAsyncProducer(t, conf)

	email := "some@mail.ru"

	uMock := NewMockUsers(t)
	uMock.EXPECT().
		CheckExistence(context.TODO(), email).
		Return(true, nil)

	cMock := pbblackbox_mocks.NewMockBlackboxServiceClient(t)

	authenticator, err := New(
		WithProducer("topic", p),
		WithUsersDB(uMock),
		WithBlackboxClient(cMock),
	)
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	req, _ := json.Marshal(&registerRequest{
		Name:     "name",
		Email:    email,
		Password: "password",
	})

	request, err := http.NewRequest("POST", "/signup", bytes.NewReader(req))
	assert.Nil(t, err)

	middleware.CorsHeaders(http.HandlerFunc(authenticator.Register)).
		ServeHTTP(rr, request)

	rsp := rr.Result()

	assert.Equal(t, http.StatusConflict, rsp.StatusCode)

	var body responses.Server
	err = json.NewDecoder(rsp.Body).Decode(&body)
	assert.Nil(t, err)
}

func TestLoginSuccess(t *testing.T) {
	conf := sarama.NewConfig()
	conf.Producer.RequiredAcks = sarama.WaitForAll
	conf.Producer.Flush.Frequency = 500 * time.Millisecond
	conf.Producer.Return.Errors = false
	p := mocks.NewAsyncProducer(t, conf)

	userId := "id"
	email := "some@mail.ru"

	uMock := NewMockUsers(t)
	uMock.EXPECT().
		Authenticate(context.TODO(), email, "password").
		Return(userId, nil)

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.MapClaims{
			"sub": userId,
		})
	signedToken, err := token.SignedString([]byte("secret"))

	cMock := pbblackbox_mocks.NewMockBlackboxServiceClient(t)
	cMock.EXPECT().
		IssueToken(context.TODO(), mock.AnythingOfType("*blackbox.IssueTokenReq")).
		Return(&blackbox.IssueTokenRsp{
			Token: signedToken,
		}, nil)

	authenticator, err := New(
		WithProducer("topic", p),
		WithUsersDB(uMock),
		WithBlackboxClient(cMock),
	)
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	req, _ := json.Marshal(&loginRequest{
		Email:    email,
		Password: "password",
	})

	request, err := http.NewRequest("POST", "/login", bytes.NewReader(req))
	assert.Nil(t, err)

	middleware.CorsHeaders(http.HandlerFunc(authenticator.Login)).
		ServeHTTP(rr, request)

	rsp := rr.Result()

	assert.Equal(t, http.StatusOK, rsp.StatusCode)

	var body responses.Server
	err = json.NewDecoder(rsp.Body).Decode(&body)
	assert.Nil(t, err)
}

func TestLoginFailedCredentials(t *testing.T) {
	conf := sarama.NewConfig()
	conf.Producer.RequiredAcks = sarama.WaitForAll
	conf.Producer.Flush.Frequency = 500 * time.Millisecond
	conf.Producer.Return.Errors = false
	p := mocks.NewAsyncProducer(t, conf)

	email := "some@mail.ru"

	uMock := NewMockUsers(t)
	uMock.EXPECT().
		Authenticate(context.TODO(), email, "password").
		Return("", users.ErrWrongCredentials)

	cMock := pbblackbox_mocks.NewMockBlackboxServiceClient(t)

	authenticator, err := New(
		WithProducer("topic", p),
		WithUsersDB(uMock),
		WithBlackboxClient(cMock),
	)
	assert.Nil(t, err)

	rr := httptest.NewRecorder()
	req, _ := json.Marshal(&loginRequest{
		Email:    email,
		Password: "password",
	})

	request, err := http.NewRequest("POST", "/login", bytes.NewReader(req))
	assert.Nil(t, err)

	middleware.CorsHeaders(http.HandlerFunc(authenticator.Login)).
		ServeHTTP(rr, request)

	rsp := rr.Result()

	assert.Equal(t, http.StatusForbidden, rsp.StatusCode)

	var body responses.Server
	err = json.NewDecoder(rsp.Body).Decode(&body)
	assert.Nil(t, err)
}
