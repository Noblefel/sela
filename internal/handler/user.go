package handler

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/Noblefel/sela/internal/types"
)

func (app *Handlers) Me(w http.ResponseWriter, r *http.Request) {
	r.SetPathValue("username", app.auth(r).Username)
	app.UserProfile(w, r)
}

func (app *Handlers) UserProfile(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimPrefix(r.PathValue("username"), "@")
	user, err := app.queryUser("WHERE username = $1", username)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			app.view(w, r, "user_404", map[string]any{})
			return
		}
		app.error(w, err)
		return
	}

	var (
		pagination = types.NewPagination(r.URL.Query())
		queryList  = "WHERE a.user_id = $1 ORDER BY a.created_at DESC "
		queryTotal = "SELECT COUNT(a.id) FROM articles a WHERE a.user_id = $1"
	)

	// TODO: change this redundant JOIN query
	articles, err := app.queryArticles(r, queryList+pagination.Query(), user.Id)
	if err != nil {
		app.error(w, err)
		return
	}

	if err := app.db.QueryRow(queryTotal, user.Id).Scan(&pagination.Total); err != nil {
		app.error(w, err)
		return
	}

	app.view(w, r, "user_profile", map[string]any{
		"user":       user,
		"articles":   articles,
		"pagination": pagination.WithPages(),
	})
}

func (app *Handlers) UserEdit(w http.ResponseWriter, r *http.Request) {
	user, err := app.queryUser("WHERE id = $1", r.PathValue("id"))
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			app.view(w, r, "user_404", map[string]any{})
			return
		}
		app.error(w, err)
		return
	}

	if !user.Authorize(app.auth(r)) {
		http.Error(w, "no permission", http.StatusForbidden)
		return
	}

	app.view(w, r, "user_edit", map[string]any{"user": user})
}

func (app *Handlers) UserUpdate(w http.ResponseWriter, r *http.Request) {
	form := types.FormUser{
		Name:     r.FormValue("name"),
		Username: r.FormValue("username"),
		Bio:      r.FormValue("bio"),
	}

	if msg := form.Validate(); msg != "" {
		app.session.Put(r.Context(), "error", msg)
		app.session.Put(r.Context(), "form", form)
		app.back(w, r)
		return
	}

	user, err := app.queryUser("WHERE id = $1", r.PathValue("id"))
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			app.view(w, r, "user_404", map[string]any{})
			return
		}
		app.error(w, err)
		return
	}

	auth := app.auth(r)

	if !user.Authorize(auth) {
		http.Error(w, "no permission", http.StatusForbidden)
		return
	}

	path, err := app.upload(r, "avatar", "avatars")
	if err != nil {
		app.error(w, err)
		return
	}

	if _, err = app.db.Exec(`
		UPDATE users SET 
			name = $2, username = $3, bio = NULLIF($4, ''),  
			avatar = COALESCE(NULLIF($5, ''), avatar), updated_at = NOW()
		WHERE id = $1`,
		user.Id,
		form.Name,
		form.Username,
		form.Bio,
		path,
	); err != nil {
		app.remove(path)
		if strings.Contains(err.Error(), "duplicate") {
			app.session.Put(r.Context(), "error", "username already taken")
			app.session.Put(r.Context(), "form", form)
			app.back(w, r)
			return
		}
		app.error(w, err)
		return
	}

	if path != "" {
		auth.Avatar = path
		app.remove(user.Avatar)
	}

	auth.Name = form.Name
	auth.Username = form.Username
	auth.LastRefresh = time.Now()
	app.session.Put(r.Context(), "auth", auth)
	app.session.Put(r.Context(), "success", "profile updated")
	http.Redirect(w, r, "/u/@"+form.Username, http.StatusSeeOther)
}

func (app *Handlers) UserDelete(w http.ResponseWriter, r *http.Request) {
	user, err := app.queryUser("WHERE id = $1", r.PathValue("id"))
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			app.view(w, r, "user_404", map[string]any{})
			return
		}
		app.error(w, err)
		return
	}

	if !user.Authorize(app.auth(r)) {
		http.Error(w, "no permission", http.StatusForbidden)
		return
	}

	if _, err = app.db.Exec("DELETE from users WHERE id = $1", user.Id); err != nil {
		app.error(w, err)
		return
	}

	app.remove(user.Avatar)
	app.session.Destroy(r.Context())
	app.session.RenewToken(r.Context())
	app.session.Put(r.Context(), "success", "account deleted")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *Handlers) Settings(w http.ResponseWriter, r *http.Request) {
	user, err := app.queryUser("WHERE id = $1", app.auth(r).Id)
	if err != nil {
		app.error(w, err) // unlikely
		return
	}

	app.view(w, r, "user_settings", map[string]any{"user": user})
}

func (app *Handlers) SettingsPrivacy(w http.ResponseWriter, r *http.Request) {
	var (
		favorite = r.FormValue("show_favorites") == "true"
		comments = r.FormValue("show_comments") == "true"
		query    = "UPDATE users SET profile_favorites_show = $2, profile_comments_show = $3 WHERE id = $1"
	)

	if _, err := app.db.Exec(query, app.auth(r).Id, favorite, comments); err != nil {
		app.error(w, err)
		return
	}

	app.session.Put(r.Context(), "success", "settings updated")
	app.back(w, r)
}

func (app *Handlers) MeFavorite(w http.ResponseWriter, r *http.Request) {
	r.SetPathValue("username", app.auth(r).Username)
	app.UserFavorite(w, r)
}

func (app *Handlers) UserFavorite(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimPrefix(r.PathValue("username"), "@")
	user, err := app.queryUser("WHERE username = $1", username)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			app.view(w, r, "user_404", map[string]any{})
			return
		}
		app.error(w, err)
		return
	}

	if !user.ShowFavorites(app.auth(r)) {
		http.Error(w, "no permission", http.StatusForbidden)
		return
	}

	var (
		pagination = types.NewPagination(r.URL.Query())
		queryList  = "WHERE al.user_id = $1 ORDER BY a.created_at DESC "
		queryTotal = "SELECT COUNT(user_id) FROM article_likes WHERE user_id = $1"
	)

	articles, err := app.queryArticles(r, queryList+pagination.Query(), user.Id)
	if err != nil {
		app.error(w, err)
		return
	}

	if err := app.db.QueryRow(queryTotal, user.Id).Scan(&pagination.Total); err != nil {
		app.error(w, err)
		return
	}

	app.view(w, r, "user_profile", map[string]any{
		"user":       user,
		"articles":   articles,
		"pagination": pagination.WithPages(),
	})
}

func (app *Handlers) queryUser(filter string, args ...any) (*types.User, error) {
	user := new(types.User)

	query := `
		SELECT id, email, username, name, COALESCE(bio, ''), COALESCE(avatar, ''), 
			profile_favorites_show, profile_comments_show,
			created_at, updated_at 
		FROM users `

	return user, app.db.QueryRow(query+filter, args...).Scan(
		&user.Id,
		&user.Email,
		&user.Username,
		&user.Name,
		&user.Bio,
		&user.Avatar,
		&user.ProfileFavoritesShow,
		&user.ProfileCommentsShow,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
}
