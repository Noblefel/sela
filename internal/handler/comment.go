package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Noblefel/sela/internal/types"
	"github.com/Noblefel/sela/internal/util"
)

func (app *Handlers) CommentPostJSON(w http.ResponseWriter, r *http.Request) {
	var form types.FormComment

	if err := json.NewDecoder(r.Body).Decode(&form); err != nil {
		app.error(w, err)
		return
	}

	if msg := form.Validate(); msg != "" {
		app.error(w, fmt.Errorf(msg))
		return
	}

	var (
		queryStore   = "INSERT INTO comments (user_id, article_id, comment) VALUES ($1,$2,$3) RETURNING id"
		queryCount   = "UPDATE articles SET comments = (comments + 1) WHERE id = $1"
		articleId, _ = strconv.Atoi(r.PathValue("id"))
		id           int
	)

	if err := app.db.QueryRow(queryStore, app.auth(r).Id, articleId, form.Comment).Scan(&id); err != nil {
		app.error(w, err)
		return
	}

	util.Background(func() {
		if _, err := app.db.Exec(queryCount, articleId); err != nil {
			app.error(w, err)
			return
		}
	})

	app.json(w, "Comment saved", map[string]any{"id": id})
}

func (app *Handlers) CommentDeleteJSON(w http.ResponseWriter, r *http.Request) {
	comment, err := app.queryComment("WHERE id = $1", r.PathValue("id"))
	if err != nil {
		app.error(w, err)
		return
	}

	if !comment.Authorize(app.auth(r)) {
		http.Error(w, "no permission", http.StatusForbidden)
		return
	}

	if _, err = app.db.Exec("DELETE FROM comments WHERE id = $1", comment.Id); err != nil {
		app.error(w, err)
		return
	}

	util.Background(func() {
		if _, err := app.db.Exec(
			"UPDATE articles SET comments = (comments - 1) WHERE id = $1",
			comment.ArticleId,
		); err != nil {
			app.error(w, err)
			return
		}
	})

	app.json(w, "Comment deleted", nil)
}

func (app *Handlers) queryComment(filter string, args ...any) (*types.Comment, error) {
	comment := new(types.Comment)

	query := `SELECT id, user_id, article_id, comment, created_at FROM comments `

	return comment, app.db.QueryRow(query+filter, args...).Scan(
		&comment.Id,
		&comment.UserId,
		&comment.ArticleId,
		&comment.Comment,
		&comment.CreatedAt,
	)
}

func (app *Handlers) queryComments(filter string, args ...any) ([]types.Comment, error) {
	var list []types.Comment

	// TODO: optimize query with conditional joins.
	// i still have no idea how to do it ergonomically
	query := `
		SELECT c.id, c.comment, u.id, u.name, u.username, COALESCE(u.avatar, ''),
			c.created_at, a.title, a.user_id, a.slug
		FROM comments c 
		LEFT JOIN users u ON c.user_id = u.id 
		LEFT JOIN articles a ON c.article_id = a.id `

	rows, err := app.db.Query(query+filter, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var c types.Comment
		if err = rows.Scan(
			&c.Id, &c.Comment, &c.UserId, &c.User.Name,
			&c.User.Username, &c.User.Avatar, &c.CreatedAt,
			&c.Article.Title, &c.Article.UserId, &c.Article.Slug,
		); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}
