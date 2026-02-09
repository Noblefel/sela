package util

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

var empty http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {}

func TestRateLimit(t *testing.T) {
	funcId := func(r *http.Request) any {
		if id := r.Context().Value("user_id"); id != nil {
			return id
		}
		return nil
	}

	rateLimit := Limiter(5, 50*time.Millisecond, funcId)

	var success int

	for range 15 {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("get", "/", nil)
		reqCtx := context.WithValue(req.Context(), "user_id", 1)
		rateLimit(empty).ServeHTTP(rec, req.WithContext(reqCtx))

		if rec.Code == 200 {
			success++
		}
	}

	if success != 5 {
		t.Errorf("successful request should be 5, got %d", success)
	}

	time.Sleep(75 * time.Millisecond) // give time for the tick
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("get", "/", nil)
	reqCtx := context.WithValue(req.Context(), "user_id", 1)
	rateLimit(empty).ServeHTTP(rec, req.WithContext(reqCtx))

	if rec.Code == 429 {
		t.Error("rate limiter should be cleaned up")
	}

	t.Run("with empty context value", func(t *testing.T) {
		var success int
		rateLimit := Limiter(5, 50*time.Millisecond, funcId)

		for range 10 {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("get", "/", nil)
			rateLimit(empty).ServeHTTP(rec, req)

			if rec.Code == 200 {
				success++
			}
		}

		if success != 10 {
			t.Errorf("all 10 request should be successful, got %d", success)
		}
	})

	t.Run("with multiple users", func(t *testing.T) {
		var success int
		rateLimit := Limiter(5, 50*time.Millisecond, funcId)

		for i := range 10 {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("get", "/", nil)
			reqCtx := context.WithValue(req.Context(), "user_id", i)
			rateLimit(empty).ServeHTTP(rec, req.WithContext(reqCtx))

			if rec.Code == 200 {
				success++
			}
		}

		if success != 10 {
			t.Errorf("all 10 request should be successful, got %d", success)
		}
	})
}
