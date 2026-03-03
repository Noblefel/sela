package handler

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/Noblefel/sela/internal/types"
)

func (app *Handlers) ArticleShow(w http.ResponseWriter, r *http.Request) {
	article, err := app.queryArticle("WHERE slug = $1", r.PathValue("slug"))
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			app.view(w, r, "article_404", map[string]any{})
			return
		}
		app.error(w, err)
		return
	}

	var (
		queryAuthor = "SELECT username, name, COALESCE(avatar, '') FROM users WHERE id = $1"
		queryLiked  = "SELECT EXISTS(SELECT FROM article_likes WHERE article_id = $1 AND user_id = $2)"
		auth        = app.auth(r)
	)

	if err := app.db.QueryRow(queryAuthor, article.UserId).Scan(
		&article.User.Username,
		&article.User.Name,
		&article.User.Avatar,
	); err != nil {
		app.error(w, err)
		return
	}

	if auth != nil {
		if err := app.db.QueryRow(queryLiked, article.Id, auth.Id).Scan(&article.Liked); err != nil {
			app.error(w, err)
			return
		}
	}

	app.view(w, r, "article_show", map[string]any{"article": article})
}

func (app *Handlers) ArticleCreate(w http.ResponseWriter, r *http.Request) {
	app.view(w, r, "article_create", map[string]any{})
}

func (app *Handlers) ArticlePost(w http.ResponseWriter, r *http.Request) {
	form := types.FormArticle{
		Title:   r.FormValue("title"),
		Excerpt: r.FormValue("excerpt"),
		Content: r.FormValue("content"),
	}

	if msg := form.Validate(); msg != "" {
		app.session.Put(r.Context(), "error", msg)
		app.session.Put(r.Context(), "form", form)
		app.back(w, r)
		return
	}

	authId := app.auth(r).Id

	path, err := app.upload(r, "image", "articles")
	if err != nil {
		app.error(w, err)
		return
	}

	var articleId int

	if err = app.db.QueryRow(`
		INSERT INTO articles (user_id, title, slug, excerpt, content, image) 
		VALUES ($1, $2, $3, NULLIF($4, ''), $5, NULLIF($6, '')) RETURNING id`,
		authId,
		form.Title,
		form.Slug,
		form.Excerpt,
		form.Content,
		path,
	).Scan(&articleId); err != nil {
		app.remove(path)
		if strings.Contains(err.Error(), "duplicate") {
			app.session.Put(r.Context(), "error", "title already taken")
			app.session.Put(r.Context(), "form", form)
			app.back(w, r)
			return
		}
		app.error(w, err)
		return
	}

	app.session.Put(r.Context(), "success", "article saved")
	http.Redirect(w, r, "/a/"+form.Slug, http.StatusSeeOther)
}

func (app *Handlers) ArticleEdit(w http.ResponseWriter, r *http.Request) {
	article, err := app.queryArticle("WHERE id = $1", r.PathValue("id"))
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			app.view(w, r, "article_404", map[string]any{})
			return
		}
		app.error(w, err)
		return
	}

	if !article.Authorize(app.auth(r)) {
		http.Error(w, "no permission", http.StatusForbidden)
		return
	}

	app.view(w, r, "article_edit", map[string]any{"article": article})
}

func (app *Handlers) ArticleUpdate(w http.ResponseWriter, r *http.Request) {
	form := types.FormArticle{
		Title:   r.FormValue("title"),
		Excerpt: r.FormValue("excerpt"),
		Content: r.FormValue("content"),
	}

	if msg := form.Validate(); msg != "" {
		app.session.Put(r.Context(), "error", msg)
		app.session.Put(r.Context(), "form", form)
		app.back(w, r)
		return
	}

	article, err := app.queryArticle("WHERE id = $1", r.PathValue("id"))
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			app.view(w, r, "article_404", map[string]any{})
			return
		}
		app.error(w, err)
		return
	}

	if !article.Authorize(app.auth(r)) {
		http.Error(w, "no permission", http.StatusForbidden)
		return
	}

	path, err := app.upload(r, "image", "articles")
	if err != nil {
		app.error(w, err)
		return
	}

	if _, err = app.db.Exec(`
		UPDATE articles SET 
			title = $2, slug = $3, excerpt = NULLIF($4, ''), content = $5, 
			image = COALESCE(NULLIF($6, ''), image), updated_at = NOW()
		WHERE id = $1`,
		article.Id,
		form.Title,
		form.Slug,
		form.Excerpt,
		form.Content,
		path,
	); err != nil {
		app.remove(path)
		if strings.Contains(err.Error(), "duplicate") {
			app.session.Put(r.Context(), "error", "title already taken")
			app.session.Put(r.Context(), "form", form)
			app.back(w, r)
			return
		}
		app.error(w, err)
		return
	}

	// image deletion is merged with delete handler because in this handler
	// i can't detect if user just deletes image without uploading a new one.
	app.session.Put(r.Context(), "success", "article saved")
	http.Redirect(w, r, "/a/"+form.Slug, http.StatusSeeOther)
}

func (app *Handlers) ArticleDelete(w http.ResponseWriter, r *http.Request) {
	article, err := app.queryArticle("WHERE id = $1", r.PathValue("id"))
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			app.view(w, r, "article_404", map[string]any{})
			return
		}
		app.error(w, err)
		return
	}

	if !article.Authorize(app.auth(r)) {
		http.Error(w, "no permission", http.StatusForbidden)
		return
	}

	if r.URL.Query().Has("image-only") {
		if _, err = app.db.Exec("UPDATE articles SET image = NULL WHERE id = $1", article.Id); err != nil {
			app.error(w, err)
			return
		}
		app.session.Put(r.Context(), "success", "image deleted")
		app.remove(article.Image)
		app.back(w, r)
		return
	}

	if _, err = app.db.Exec("DELETE from articles WHERE id = $1", article.Id); err != nil {
		app.error(w, err)
		return
	}
	app.remove(article.Image)
	app.session.Put(r.Context(), "success", "article deleted")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *Handlers) ArticleLikeToggle(w http.ResponseWriter, r *http.Request) {
	article, err := app.queryArticle("WHERE id = $1", r.PathValue("id"))
	if err != nil {
		app.error(w, err)
		return
	}

	var (
		authId     = app.auth(r).Id
		queryExist = "SELECT EXISTS(SELECT FROM article_likes WHERE article_id = $1 AND user_id = $2)"
		queryLikes string
		queryCount string
		hasLiked   bool
	)

	if err := app.db.QueryRow(queryExist, article.Id, authId).Scan(&hasLiked); err != nil {
		app.error(w, err)
		return
	}

	if hasLiked {
		queryLikes = "DELETE FROM article_likes WHERE article_id = $1 AND user_id = $2"
		queryCount = "UPDATE articles SET likes = (likes - 1) WHERE id = $1"
	} else {
		queryLikes = "INSERT INTO article_likes (article_id, user_id) VALUES ($1, $2)"
		queryCount = "UPDATE articles SET likes = (likes + 1) WHERE id = $1"
	}

	if _, err = app.db.Exec(queryLikes, article.Id, authId); err != nil {
		app.error(w, err)
		return
	}

	if _, err = app.db.Exec(queryCount, article.Id); err != nil {
		app.error(w, err)
		return
	}
}

func (app *Handlers) queryArticle(filter string, args ...any) (*types.Article, error) {
	article := new(types.Article)

	query := `
		SELECT id, slug, user_id, title, COALESCE(excerpt, ''), 
			content, COALESCE(image, ''), likes, created_at, updated_at
		FROM articles `

	return article, app.db.QueryRow(query+filter, args...).Scan(
		&article.Id,
		&article.Slug,
		&article.UserId,
		&article.Title,
		&article.Excerpt,
		&article.Content,
		&article.Image,
		&article.Likes,
		&article.CreatedAt,
		&article.UpdatedAt,
	)
}

// TODO: remove http.Request param/find a better way to check liked articles.
// TODO: join article_likes so this can be used in user favorites handler.
func (app *Handlers) queryArticles(r *http.Request, filter string, args ...any) ([]types.Article, error) {
	var list []types.Article
	query := `SELECT a.id, a.user_id, a.title, a.slug, COALESCE(a.excerpt, ''), a.content, COALESCE(a.image, ''), 
		a.likes, a.created_at, a.updated_at, a.deleted_at, u.username, u.name, COALESCE(u.avatar, ''), 
		EXISTS(SELECT FROM article_likes al WHERE al.article_id = a.id and al.user_id = %v) `

	// check liked articles
	if app.auth(r) != nil {
		query = fmt.Sprintf(query, app.auth(r).Id)
	} else {
		query = fmt.Sprintf(query, 0)
	}

	query += "FROM articles a LEFT JOIN users u ON a.user_id = u.id "

	rows, err := app.db.Query(query+filter, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var article types.Article
		if err = rows.Scan(
			&article.Id, &article.UserId, &article.Title, &article.Slug,
			&article.Excerpt, &article.Content, &article.Image, &article.Likes,
			&article.CreatedAt, &article.UpdatedAt, &article.DeletedAt,
			&article.User.Username, &article.User.Name, &article.User.Avatar, &article.Liked,
		); err != nil {
			return nil, err
		}
		list = append(list, article)
	}
	return list, rows.Err()
}
