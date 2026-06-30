package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Noblefel/sela/internal/types"
)

func (app *Handlers) ArticleDraftPostJSON(w http.ResponseWriter, r *http.Request) {
	var form types.FormArticleDraft

	if err := json.NewDecoder(r.Body).Decode(&form); err != nil {
		app.error(w, err)
		return
	}

	if msg := form.Validate(); msg != "" {
		app.error(w, fmt.Errorf(msg))
		return
	}

	var id int

	if err := app.db.QueryRow(`
		INSERT INTO article_drafts (user_id, title, excerpt, content)
		VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, '')) RETURNING id`,
		app.auth(r).Id, form.Title, form.Excerpt, form.Content,
	).Scan(&id); err != nil {
		app.error(w, err)
		return
	}

	app.json(w, "article saved into draft", map[string]any{"id": id})
}

func (app *Handlers) ArticleDraftUseJSON(w http.ResponseWriter, r *http.Request) {
	draft, err := app.queryArticleDraft("WHERE id = $1", r.PathValue("id"))
	if err != nil {
		app.error(w, err)
		return
	}

	if !draft.Authorize(app.auth(r)) {
		http.Error(w, "no permission", http.StatusForbidden)
		return
	}

	app.json(w, "draft loaded", map[string]any{"draft": draft})
}

func (app *Handlers) ArticleDraftDeleteJSON(w http.ResponseWriter, r *http.Request) {
	draft, err := app.queryArticleDraft("WHERE id = $1", r.PathValue("id"))
	if err != nil {
		app.error(w, err)
		return
	}

	if !draft.Authorize(app.auth(r)) {
		http.Error(w, "no permission", http.StatusForbidden)
		return
	}

	if _, err := app.db.Exec("DELETE FROM article_drafts WHERE id = $1", draft.Id); err != nil {
		app.error(w, err)
		return
	}

	app.json(w, "draft deleted", nil)
}

func (app *Handlers) queryArticleDraft(filter string, args ...any) (*types.ArticleDraft, error) {
	draft := new(types.ArticleDraft)
	query := `
		SELECT id, user_id, title, COALESCE(excerpt, ''), COALESCE(content,'') 
		FROM article_drafts `

	return draft, app.db.QueryRow(query+filter, args...).Scan(
		&draft.Id,
		&draft.UserId,
		&draft.Title,
		&draft.Excerpt,
		&draft.Content,
	)
}

func (app *Handlers) queryArticleDrafts(filter string, args ...any) ([]types.ArticleDraft, error) {
	var list []types.ArticleDraft
	query := "SELECT id, title, COALESCE(excerpt, '') FROM article_drafts "

	rows, err := app.db.Query(query+filter, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var a types.ArticleDraft
		if err = rows.Scan(&a.Id, &a.Title, &a.Excerpt); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}
