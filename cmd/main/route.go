package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/Noblefel/sela/internal/types"
	"github.com/Noblefel/sela/internal/util"
)

func route() http.Handler {
	mux := http.NewServeMux()

	handle(mux, "GET /a/{slug}", app.ArticleShow)
	handle(mux, "POST /articles", app.ArticlePost, auth)
	handle(mux, "GET /articles/create", app.ArticleCreate, auth)
	handle(mux, "GET /articles/{id}/edit", app.ArticleEdit, auth)
	handle(mux, "POST /articles/{id}/update", app.ArticleUpdate, auth)
	handle(mux, "POST /articles/{id}/delete", app.ArticleDelete, auth)
	handle(mux, "GET /search", app.Search)

	handle(mux, "GET /me", app.Me, auth)
	handle(mux, "GET /u/{username}", app.UserProfile)
	handle(mux, "GET /users/{id}/edit", app.UserEdit, auth)
	handle(mux, "POST /users/{id}/update", app.UserUpdate, auth)
	handle(mux, "POST /users/{id}/delete", app.UserDelete, auth)
	handle(mux, "GET /settings", app.Settings, auth)

	handle(mux, "GET /auth", app.Auth)
	handle(mux, "/auth/force", app.AuthForceLogin)
	handle(mux, "/auth/google", app.AuthGoogle)
	handle(mux, "/auth/callback", app.AuthGoogleCallback)
	handle(mux, "GET /auth/logout", app.AuthLogout, auth)

	handle(mux, "POST /auth/reset-email", app.AuthResetStart, auth, strict)
	handle(mux, "GET /auth/reset-email/{token}", app.AuthResetPage, auth)
	handle(mux, "POST /auth/reset-email/{token}", app.AuthReset, auth)
	handle(mux, "/{$}", app.Home)

	mux.Handle("/images/{dir}/{prefix}/{prefix2}/{name}",
		http.StripPrefix("/images", http.HandlerFunc(app.Image)))

	mux.Handle("/static/",
		http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	return global(mux)
}

func global(mux http.Handler) http.Handler {
	cors := http.NewCrossOriginProtection()
	mux = refresh(mux)
	mux = session.LoadAndSave(mux)
	mux = cors.Handler(mux)
	mux = catch(mux)
	mux = throttle(mux)
	return mux
}

func catch(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				msg := fmt.Sprintln("RECOVERED:", err)
				http.Error(w, msg, http.StatusInternalServerError)
				log.Println(msg)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func refresh(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok := session.Get(r.Context(), "auth").(types.Auth)
		if ok && u.ShouldRefresh() {
			query := "SELECT name, username, COALESCE(avatar, '') FROM users WHERE id = $1"
			if err := conn.QueryRow(query, u.Id).Scan(&u.Name, &u.Username, &u.Avatar); err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized) // unlikely
				return
			}
			u.LastRefresh = time.Now()
			session.Put(r.Context(), "auth", u)
		}
		next.ServeHTTP(w, r)
	})
}

func auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if session.Get(r.Context(), "auth") == nil {
			session.Put(r.Context(), "error", "you need to login first")
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		limit(next).ServeHTTP(w, r)
	})
}

var limit = util.Limiter(10, 7*time.Second, func(r *http.Request) any {
	if auth, ok := session.Get(r.Context(), "auth").(types.Auth); ok {
		return auth.Id
	}
	return nil
})

var strict = util.Limiter(1, 1*time.Minute, func(r *http.Request) any {
	if auth, ok := session.Get(r.Context(), "auth").(types.Auth); ok {
		return auth.Id
	}
	return nil
})

var throttle = util.Limiter(20, 3*time.Second, func(r *http.Request) any {
	if ip := r.Header.Get("x-real-ip"); ip != "" {
		return ip
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return nil
	}
	return ip
})

func handle(mux *http.ServeMux, path string, fn http.HandlerFunc, mw ...func(http.Handler) http.Handler) {
	final := http.Handler(fn)
	for i := len(mw) - 1; i >= 0; i-- {
		final = mw[i](final)
	}
	mux.Handle(path, final)
}
