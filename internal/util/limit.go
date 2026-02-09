package util

import (
	"net/http"
	"sync"
	"time"
)

func Limiter(max int, interval time.Duration, fn func(*http.Request) any) func(next http.Handler) http.Handler {
	store := new(sync.Map)

	go func() {
		ticker := time.NewTicker(interval)
		for {
			<-ticker.C
			store.Range(func(key, value any) bool {
				store.Delete(key)
				return true
			})
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			contextVal := fn(r)
			if contextVal == nil {
				next.ServeHTTP(w, r)
				return
			}

			v, ok := store.Load(contextVal)
			if !ok {
				v = 0
			}

			counts := v.(int) + 1
			if counts > max {
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte("too many request sorry"))
				return
			}

			store.Swap(contextVal, counts)
			next.ServeHTTP(w, r)
		})
	}
}
