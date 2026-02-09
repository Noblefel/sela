package main

import (
	"net"
	"net/http"
	"time"

	"github.com/Noblefel/sela/internal/types"
	"github.com/Noblefel/sela/internal/util"
)

func route() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /a/{slug}", app.ArticleShow)
	mux.HandleFunc("POST /articles", auth(app.ArticlePost))
	mux.HandleFunc("GET /articles/create", auth(app.ArticleCreate))
	mux.HandleFunc("GET /articles/{id}/edit", auth(app.ArticleEdit))
	mux.HandleFunc("POST /articles/{id}/update", auth(app.ArticleUpdate))
	mux.HandleFunc("POST /articles/{id}/delete", auth(app.ArticleDelete))
	mux.HandleFunc("GET /search", app.Search)

	mux.HandleFunc("GET /u/{username}", app.UserProfile)
	mux.HandleFunc("GET /users/{id}/edit", auth(app.UserEdit))
	mux.HandleFunc("POST /users/{id}/update", auth(app.UserUpdate))
	mux.HandleFunc("POST /users/{id}/delete", auth(app.UserDelete))

	mux.HandleFunc("/auth", app.Auth)
	mux.HandleFunc("/auth/force", app.AuthForceLogin)
	mux.HandleFunc("/auth/google", app.AuthGoogle)
	mux.HandleFunc("/auth/callback", app.AuthGoogleCallback)
	mux.HandleFunc("/auth/logout", auth(app.AuthLogout))
	mux.HandleFunc("/{$}", app.Home)

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
	mux = throttle(mux)
	return mux
}

func refresh(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
	}
}

func auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// session.Put(r.Context(), "auth", types.Auth{Id: 1}) // Debug mode
		if session.Get(r.Context(), "auth") == nil {
			session.Put(r.Context(), "error", "you need to login first")
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		ratelimit(next).ServeHTTP(w, r)
	}
}

var ratelimit = util.Limiter(10, 7*time.Second, func(r *http.Request) any {
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
