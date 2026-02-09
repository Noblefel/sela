package handler

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/Noblefel/sela/internal/types"
	"github.com/Noblefel/sela/internal/util"
)

type Handlers struct {
	db      types.DB
	session types.Session
	render  types.Renderer
	config  *types.Config
}

func New(db types.DB, s types.Session, r types.Renderer, c *types.Config) *Handlers {
	return &Handlers{db, s, r, c}
}

func (app *Handlers) Image(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, filepath.Join(app.config.UploadRoot, r.URL.Path))
}

func (app *Handlers) Home(w http.ResponseWriter, r *http.Request) {
	articlesNew, err := app.queryArticles("ORDER BY a.created_at DESC LIMIT 10")
	if err != nil {
		app.error(w, err)
		return
	}

	month := time.Now().AddDate(0, -1, 0)

	articlesMonthly, err := app.queryArticles("WHERE a.created_at > $1 LIMIT 2", month)
	if err != nil {
		app.error(w, err)
		return
	}

	app.view(w, r, "index", map[string]any{
		"articles_new":     articlesNew,
		"articles_monthly": articlesMonthly,
	})
}

func (app *Handlers) Search(w http.ResponseWriter, r *http.Request) {
	var (
		filter     = "WHERE to_tsvector('simple', a.title) @@ plainto_tsquery('simple', $1) "
		path       = r.URL.Query()
		pagination = types.NewPagination(path)
		total      int
	)

	query := "SELECT COUNT(a.id) FROM articles a " + filter
	// get the total early before the ORDER BY filter
	if err := app.db.QueryRow(query, path.Get("key")).Scan(&total); err != nil {
		app.error(w, err)
		return
	}

	if path.Get("sort") == "oldest" {
		filter += "ORDER BY a.created_at ASC "
	} else {
		filter += "ORDER BY a.created_at DESC "
	}

	articles, err := app.queryArticles(filter+pagination.Query(), path.Get("key"))
	if err != nil {
		app.error(w, err)
		return
	}

	pagination.ParsePages(total)

	app.view(w, r, "search", map[string]any{
		"articles":   articles,
		"pagination": pagination,
		"query":      r.URL.Query(),
	})
}

//
// WRAPPERS
//

// parse form file and store the file if exist. Returns path (after root) or err
func (app *Handlers) upload(r *http.Request, file string, dir string) (string, error) {
	f, fh, err := r.FormFile(file)
	if err == http.ErrMissingFile {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	defer f.Close()

	if fh.Size > (2 << 20) {
		return "", errors.New("image too big")
	}

	fbyte, _ := io.ReadAll(f)
	ftype := http.DetectContentType(fbyte)

	if ftype != "image/jpeg" &&
		ftype != "image/jpg" &&
		ftype != "image/png" &&
		ftype != "image/svg" {
		return "", errors.New("file not supported")
	}

	name := fmt.Sprintf("%s-%d.%s", util.RandomString(30), time.Now().UnixNano(), ftype[6:])
	path := filepath.Join(dir, name[0:3], name[3:6], name)
	full := filepath.Join(app.config.UploadRoot, path)

	if err := os.MkdirAll(filepath.Dir(full), os.ModePerm); err != nil {
		return "", err
	}

	return path, os.WriteFile(full, fbyte, os.ModePerm)
}

// remove file inside upload root
func (app *Handlers) remove(path string) {
	os.Remove(filepath.Join(app.config.UploadRoot, path))
}

func (app *Handlers) view(w http.ResponseWriter, r *http.Request, page string, data map[string]any) {
	// flashes
	data["success"] = app.session.Pop(r.Context(), "success")
	data["error"] = app.session.Pop(r.Context(), "error")
	data["form"] = app.session.Pop(r.Context(), "form")
	// common
	data["auth"] = app.auth(r)
	data["page"] = r.URL.Path

	if err := app.render.View(w, page, data); err != nil {
		fmt.Fprint(w, err.Error())
		log.Output(2, err.Error())
	}
}

func (app *Handlers) back(w http.ResponseWriter, r *http.Request) {
	back := r.Referer()
	if back == "" {
		back = "/"
	}
	http.Redirect(w, r, back, http.StatusSeeOther)
}

func (app *Handlers) auth(r *http.Request) *types.Auth {
	if auth, ok := app.session.Get(r.Context(), "auth").(types.Auth); ok {
		return &auth
	}
	return nil // assertion
}

func (app *Handlers) error(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
	log.Output(2, err.Error())
}
