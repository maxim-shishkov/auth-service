package http_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	transporthttp "github.com/maxim-shishkov/auth-service/internal/transport/http"
)

func TestRateLimiter(t *testing.T) {
	limiter := transporthttp.NewRateLimiter(2, time.Minute)

	require.True(t, limiter.Allow("127.0.0.1"))
	require.True(t, limiter.Allow("127.0.0.1"))

	require.False(t, limiter.Allow("127.0.0.1"))
}
