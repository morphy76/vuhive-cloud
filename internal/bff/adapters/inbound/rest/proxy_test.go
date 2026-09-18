package rest_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morphy76/vuhive-cloud/internal/bff/adapters/inbound/rest"
	"github.com/morphy76/vuhive-cloud/internal/bff/adapters/outbound/circuitbreaker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestControlPlaneProxy_CircuitBreaker(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("proxies successfully when circuit is closed", func(t *testing.T) {
		cb := circuitbreaker.New(circuitbreaker.Config{
			Name: "proxy-cb",
		})

		baseTransport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "/api/v1/suites", req.URL.Path)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{"suites":[]}`)),
				Header:     make(http.Header),
			}, nil
		})

		transport := circuitbreaker.NewResilientTransport(baseTransport, cb, 2*time.Second)
		proxyHandler := rest.NewControlPlaneProxy("http://controlplane:8080", transport)

		router := gin.New()
		router.GET("/api/bff/v1/suites", proxyHandler)

		req := httptest.NewRequest(http.MethodGet, "/api/bff/v1/suites", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `{"suites":[]}`)
	})

	t.Run("short-circuits with 503 when circuit breaker is open", func(t *testing.T) {
		cb := circuitbreaker.New(circuitbreaker.Config{
			Name:         "proxy-cb-failing",
			MaxRequests:  1,
			Timeout:      100 * time.Millisecond,
			FailureRatio: 0.5,
			MinRequests:  2,
		})

		// Trip circuit open
		done1, err := cb.Allow()
		require.NoError(t, err)
		done1(false)

		done2, err := cb.Allow()
		require.NoError(t, err)
		done2(false)

		assert.True(t, cb.IsOpen())

		baseTransport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			t.Fatal("upstream transport should not be called when circuit breaker is open")
			return nil, nil
		})

		transport := circuitbreaker.NewResilientTransport(baseTransport, cb, 2*time.Second)
		proxyHandler := rest.NewControlPlaneProxy("http://controlplane:8080", transport)

		router := gin.New()
		router.GET("/api/bff/v1/suites", proxyHandler)

		req := httptest.NewRequest(http.MethodGet, "/api/bff/v1/suites", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
		assert.Contains(t, rec.Body.String(), `"upstream service circuit open"`)
	})
}
