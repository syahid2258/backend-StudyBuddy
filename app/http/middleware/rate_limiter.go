package middleware

import (
	"net/http"
	"sync"
	"time"

	httpcontract "github.com/goravel/framework/contracts/http"
)

// ipBucket holds per-IP request counts within a time window.
type ipBucket struct {
	count     int
	windowEnd time.Time
}

// RateLimiter is a simple in-memory sliding-window rate limiter per IP.
// Closes FINDING-007 and FINDING-021 (No Rate Limiting / No Account Lockout).
type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*ipBucket
	maxReqs  int
	window   time.Duration
	httpCode int
}

// NewRateLimiter creates a new rate limiter.
// maxReqs: maximum number of requests allowed within window.
// window: duration of the time window (e.g., time.Minute).
func NewRateLimiter(maxReqs int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		buckets:  make(map[string]*ipBucket),
		maxReqs:  maxReqs,
		window:   window,
		httpCode: http.StatusTooManyRequests,
	}
	// Background cleanup goroutine
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			rl.mu.Lock()
			now := time.Now()
			for ip, b := range rl.buckets {
				if now.After(b.windowEnd) {
					delete(rl.buckets, ip)
				}
			}
			rl.mu.Unlock()
		}
	}()
	return rl
}

func (r *RateLimiter) Signature() string {
	return "rate_limiter"
}

func (r *RateLimiter) Handle(ctx httpcontract.Context) {
	ip := ctx.Request().Ip()
	if ip == "" {
		ip = "unknown"
	}

	r.mu.Lock()
	now := time.Now()
	b, exists := r.buckets[ip]
	if !exists || now.After(b.windowEnd) {
		r.buckets[ip] = &ipBucket{count: 1, windowEnd: now.Add(r.window)}
		r.mu.Unlock()
		ctx.Request().Next()
		return
	}
	b.count++
	count := b.count
	r.mu.Unlock()

	if count > r.maxReqs {
		ctx.Response().Header("Retry-After", "60")
		ctx.Response().Json(r.httpCode, httpcontract.Json{
			"status":  "error",
			"message": "Terlalu banyak permintaan. Silakan coba lagi nanti.",
		}).Abort()
		return
	}
	ctx.Request().Next()
}

// Preconfigured limiters
var (
	// AuthLimiter: 10 attempts per minute per IP for login/register endpoints
	AuthLimiter = NewRateLimiter(10, time.Minute)

	// AILimiter: 30 requests per minute per IP for AI-heavy endpoints
	AILimiter = NewRateLimiter(30, time.Minute)

	// GeneralLimiter: 120 requests per minute for general API
	GeneralLimiter = NewRateLimiter(120, time.Minute)
)
