package redirector

import (
	"context"
	"net/http"
	"net/http/httptest"
	"shortener/pkg/models/urls"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRedirectionOk(t *testing.T) {
	u := NewMockUrls(t)
	shortUrl := "12345"
	u.EXPECT().GetLongUrl(context.TODO(), shortUrl).Return("long_url", nil)

	r, err := New(WithUrlsModel(u))
	assert.Nil(t, err)

	recorder := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "/"+shortUrl, nil)
	assert.Nil(t, err)

	r.Redirect(recorder, req)
	rsp := recorder.Result()

	assert.Equal(t, rsp.StatusCode, http.StatusMovedPermanently)
}

func TestRedirectionUrlTooLong(t *testing.T) {
	u := NewMockUrls(t)
	shortUrl := "1234567"

	r, err := New(WithUrlsModel(u))
	assert.Nil(t, err)

	recorder := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "/"+shortUrl, nil)
	assert.Nil(t, err)

	r.Redirect(recorder, req)
	rsp := recorder.Result()

	assert.Equal(t, rsp.StatusCode, http.StatusNotFound)
}

func TestRedirectionUrlNotFound(t *testing.T) {
	u := NewMockUrls(t)
	u.EXPECT().GetLongUrl(context.TODO(), "other").Return("", urls.ErrNotFound)

	r, err := New(WithUrlsModel(u))
	assert.Nil(t, err)

	recorder := httptest.NewRecorder()

	req, err := http.NewRequest("GET", "/other", nil)
	assert.Nil(t, err)

	r.Redirect(recorder, req)
	rsp := recorder.Result()

	assert.Equal(t, rsp.StatusCode, http.StatusNotFound)
}
