package viewer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"shortener/pkg/domain"
	"shortener/proto/blackbox"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbblackbox_mock "shortener/mocks/shortener/proto/blackbox"
)

func TestViewerWrongToken(t *testing.T) {
	u := NewMockUrls(t)

	c := pbblackbox_mock.NewMockBlackboxServiceClient(t)
	c.EXPECT().
		ValidateToken(context.TODO(), mock.AnythingOfType("*blackbox.ValidateTokenReq")).
		Return(nil, status.Errorf(
			codes.InvalidArgument,
			"invalid token",
		))
	v, err := New(
		WithUrls(u),
		WithBlackboxClient(c),
		WithRedirectorHost("host"),
	)
	assert.Nil(t, err)

	recorder := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/", nil)
	assert.Nil(t, err)

	jwtCookie := http.Cookie{
		Name:     "JWT",
		Value:    "invalid token",
		Path:     "/",
		MaxAge:   3600,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}

	r.AddCookie(&jwtCookie)

	v.HandleHistory(recorder, r)
	rsp := recorder.Result()
	assert.Equal(t, http.StatusForbidden, rsp.StatusCode)
}

func TestViewerNoCookie(t *testing.T) {
	u := NewMockUrls(t)
	c := pbblackbox_mock.NewMockBlackboxServiceClient(t)

	v, err := New(
		WithUrls(u),
		WithBlackboxClient(c),
		WithRedirectorHost("host"),
	)
	assert.Nil(t, err)

	recorder := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/", nil)
	assert.Nil(t, err)

	v.HandleHistory(recorder, r)
	rsp := recorder.Result()
	assert.Equal(t, http.StatusPreconditionFailed, rsp.StatusCode)
}

func TestViewerSuccess(t *testing.T) {
	u := NewMockUrls(t)
	exDate := time.Now()
	u.EXPECT().History(context.TODO(), "id").Return([]*domain.UrlInfo{
		{
			ShortUrl:       "short",
			LongUrl:        "long",
			ExpirationDate: exDate,
		},
	}, nil)
	c := pbblackbox_mock.NewMockBlackboxServiceClient(t)

	c.EXPECT().
		ValidateToken(context.TODO(), mock.AnythingOfType("*blackbox.ValidateTokenReq")).
		Return(&blackbox.ValidateTokenRsp{
			UserId: "id",
		}, nil)
	v, err := New(
		WithUrls(u),
		WithBlackboxClient(c),
		WithRedirectorHost("host"),
	)
	assert.Nil(t, err)

	recorder := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/", nil)
	assert.Nil(t, err)

	jwtCookie := http.Cookie{
		Name:     "JWT",
		Value:    "token",
		Path:     "/",
		MaxAge:   3600,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}

	r.AddCookie(&jwtCookie)

	v.HandleHistory(recorder, r)
	rsp := recorder.Result()
	assert.Equal(t, http.StatusOK, rsp.StatusCode)

	var res []*domain.UrlInfo

	err = json.NewDecoder(rsp.Body).Decode(&res)
	assert.Nil(t, err)

	assert.Equal(t, len(res), 1)
}
