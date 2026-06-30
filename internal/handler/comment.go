package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Noblefel/sela/internal/types"
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

	if _, err := app.db.Exec(queryCount, articleId); err != nil {
		app.error(w, err)
		return
	}

	app.json(w, "Comment saved", map[string]any{"id": id})
}

func (app *Handlers) queryComments(filter string, args ...any) ([]types.Comment, error) {
	var list []types.Comment
	query := `
		SELECT c.id, c.comment, u.id, u.name, u.username, COALESCE(u.avatar, ''),
			c.created_at
		FROM comments c LEFT JOIN users u ON c.user_id = u.id `

	rows, err := app.db.Query(query+filter, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var c types.Comment
		if err = rows.Scan(
			&c.Id, &c.Comment, &c.User.Id, &c.User.Name,
			&c.User.Username, &c.User.Avatar, &c.CreatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}
