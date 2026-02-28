package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"os"
	"strings"
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
		auth.LastRefresh = time.Now()

		if err := app.db.QueryRow(`
			INSERT INTO users (email, username, name, avatar)
			VALUES ($1, $2, $3, $4) RETURNING id`,
			user.Email,
			auth.Username,
			auth.Name,
			auth.Avatar,
		).Scan(&auth.Id); err != nil {
			app.error(w, err)
			return
		}

		util.Background(func() {
			b, _ := os.ReadFile("emails/welcome.html")
			app.mailer.Send(user.Email, "Thanks for joining", b)
		})

		app.session.RenewToken(r.Context())
		app.session.Put(r.Context(), "auth", auth)
		app.session.Put(r.Context(), "success", "account registered")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if err != nil {
		app.error(w, err)
		return
	}

	auth.LastRefresh = time.Now()
	app.session.RenewToken(r.Context())
	app.session.Put(r.Context(), "auth", auth)
	app.session.Put(r.Context(), "success", "welcome back")
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
	app.session.Put(r.Context(), "success", "logged in (debug)")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *Handlers) AuthLogout(w http.ResponseWriter, r *http.Request) {
	app.session.Destroy(r.Context())
	app.session.RenewToken(r.Context())
	app.session.Put(r.Context(), "success", "logged out")
	app.back(w, r)
}

// to generate tokens and redirect to verification page
func (app *Handlers) AuthResetStart(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")

	if _, err := mail.ParseAddress(email); err != nil {
		app.session.Put(r.Context(), "error", "email is malformed")
		app.back(w, r)
		return
	}

	auth := app.auth(r)

	user, err := app.queryUser("WHERE email = $1", email)
	if err == sql.ErrNoRows {
		token := fmt.Sprintf("%s-%d", util.RandomString(10), time.Now().UnixNano())
		code := util.RandomString(6)

		if _, err = app.db.Exec(
			"INSERT INTO reset_emails (token, user_id, code, email) VALUES ($1, $2, $3, $4)",
			token, auth.Id, code, email,
		); err != nil {
			app.error(w, err)
			return
		}

		util.Background(func() {
			b, _ := os.ReadFile("emails/reset_code.html")
			html := strings.ReplaceAll(string(b), "[CODE]", code)
			app.mailer.Send(email, "Email reset verification code", []byte(html))
		})

		http.Redirect(w, r, "/auth/reset-email/"+token, http.StatusSeeOther)
	} else if err != nil {
		app.error(w, err)
	} else if user.Id == auth.Id {
		http.Error(w, "You already own this email", http.StatusBadRequest)
	} else {
		app.session.Put(r.Context(), "error", "Email already used by others")
		app.back(w, r)
	}
}

func (app *Handlers) AuthResetPage(w http.ResponseWriter, r *http.Request) {
	var reset types.ResetEmail
	query := "SELECT user_id, email, created_at FROM reset_emails WHERE token = $1"

	if err := app.db.QueryRow(query, r.PathValue("token")).Scan(
		&reset.UserId,
		&reset.Email,
		&reset.CreatedAt,
	); err != nil {
		app.error(w, err)
		return
	}

	if reset.UserId != app.auth(r).Id {
		http.Error(w, "no permission", http.StatusForbidden)
		return
	}

	if time.Since(reset.CreatedAt) > (5 * time.Minute) {
		query := "DELETE FROM reset_emails WHERE user_id = $1 AND created_at <= NOW() - INTERVAL '5 minutes'"
		if _, err := app.db.Exec(query, reset.UserId); err != nil {
			log.Println("error deleting expired links")
		}
		http.Error(w, "link expired", http.StatusForbidden)
		return
	}

	app.view(w, r, "auth_reset", map[string]any{"email": reset.Email})
}

func (app *Handlers) AuthReset(w http.ResponseWriter, r *http.Request) {
	var reset types.ResetEmail
	token := r.PathValue("token")
	query := "SELECT user_id, email, code, created_at FROM reset_emails WHERE token = $1"

	if err := app.db.QueryRow(query, token).Scan(
		&reset.UserId,
		&reset.Email,
		&reset.Code,
		&reset.CreatedAt,
	); err != nil {
		app.error(w, err)
		return
	}

	if reset.UserId != app.auth(r).Id {
		http.Error(w, "no permission", http.StatusForbidden)
		return
	}

	if reset.Code != r.FormValue("code") {
		app.session.Put(r.Context(), "error", "Incorrect code")
		app.back(w, r)
		return
	}

	_, err := app.db.Exec("UPDATE users SET email = $2 WHERE id = $1", reset.UserId, reset.Email)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			app.session.Put(r.Context(), "error", "email already used")
			app.back(w, r)
			return
		}
		app.error(w, err)
		return
	}

	app.view(w, r, "auth_reset_success", map[string]any{})
}
