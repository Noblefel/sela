package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Noblefel/sela/internal/types"
	"github.com/Noblefel/sela/internal/util"
	"golang.org/x/oauth2"
)

var googleEndpoint = oauth2.Endpoint{
	AuthURL:       "https://accounts.google.com/o/oauth2/auth",
	TokenURL:      "https://oauth2.googleapis.com/token",
	DeviceAuthURL: "https://oauth2.googleapis.com/device/code",
	AuthStyle:     oauth2.AuthStyleInParams,
}

func (app *Handlers) Auth(w http.ResponseWriter, r *http.Request) {
	if app.auth(r) != nil {
		app.session.Put(r.Context(), "error", "you are logged in")
		app.back(w, r)
		return
	}
	app.view(w, r, "auth", map[string]any{})
}

func (app *Handlers) AuthGoogle(w http.ResponseWriter, r *http.Request) {
	if app.auth(r) != nil {
		app.back(w, r)
		return
	}

	state := util.RandomString(16)

	http.SetCookie(w, &http.Cookie{
		Name:     "state",
		Value:    state,
		HttpOnly: true,
	})

	c := oauth2.Config{
		ClientID:     app.config.GoogleClientId,
		ClientSecret: app.config.GoogleClientSecret,
		RedirectURL:  app.config.GoogleCallbackURL,
		Endpoint:     googleEndpoint,
		Scopes:       []string{"email", "profile"},
	}

	to := c.AuthCodeURL(state)
	http.Redirect(w, r, to, http.StatusTemporaryRedirect)
}

func (app *Handlers) AuthGoogleCallback(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("state")
	if err != nil || cookie.Value != r.FormValue("state") {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	c := oauth2.Config{
		ClientID:     app.config.GoogleClientId,
		ClientSecret: app.config.GoogleClientSecret,
		RedirectURL:  app.config.GoogleCallbackURL,
		Endpoint:     googleEndpoint,
		Scopes:       []string{"email", "profile"},
	}

	token, err := c.Exchange(context.Background(), r.FormValue("code"))
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	link := "https://www.googleapis.com/oauth2/v2/userinfo?access_token="

	resp, err := c.Client(r.Context(), token).Get(link + token.AccessToken)
	if err != nil {
		app.error(w, err)
		return
	}
	defer resp.Body.Close()

	var user types.GoogleUser

	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		app.error(w, err)
		return
	}

	var auth types.Auth
	query := "SELECT id, name, username, avatar FROM users WHERE email = $1"

	err = app.db.QueryRow(query, user.Email).Scan(
		&auth.Id,
		&auth.Name,
		&auth.Username,
		&auth.Avatar,
	)
	if err == sql.ErrNoRows {
		auth.Name = user.Name
		auth.Username = util.Slug(user.Name) + "-" + util.RandomString(4)
		auth.Avatar = user.Picture
		err = app.db.QueryRow(`
			INSERT INTO users (email, username, name, avatar)
			VALUES ($1, $2, $3, $4) RETURNING id`,
			user.Email,
			auth.Username,
			auth.Name,
			auth.Avatar,
		).Scan(&auth.Id)
	}
	if err != nil {
		app.error(w, err)
		return
	}

	auth.LastRefresh = time.Now()
	app.session.RenewToken(r.Context())
	app.session.Put(r.Context(), "auth", auth)
	app.session.Put(r.Context(), "success", "logged in")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *Handlers) AuthForceLogin(w http.ResponseWriter, r *http.Request) {
	var auth types.Auth
	query := "SELECT id, name, username, COALESCE(avatar, '') FROM users WHERE id = $1"

	if err := app.db.QueryRow(query, r.FormValue("id")).Scan(
		&auth.Id,
		&auth.Name,
		&auth.Username,
		&auth.Avatar,
	); err != nil {
		app.error(w, err)
		return
	}

	auth.LastRefresh = time.Now()
	app.session.Put(r.Context(), "auth", auth)
	app.session.Put(r.Context(), "success", "logged in")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *Handlers) AuthLogout(w http.ResponseWriter, r *http.Request) {
	app.session.Destroy(r.Context())
	app.session.RenewToken(r.Context())
	app.session.Put(r.Context(), "success", "logged out")
	app.back(w, r)
}
