package types

import (
	"context"
	"database/sql"
	"io"
)

type Config struct {
	GoogleClientId     string `json:"google_client_id"`
	GoogleClientSecret string `json:"google_client_secret"`
	GoogleCallbackURL  string `json:"google_callback_url"`
	DB_Name            string `json:"db_name"`
	DB_User            string `json:"db_user"`
	DB_Password        string `json:"db_password"`
	UploadRoot         string `json:"upload_root"`
}

type DB interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

type Session interface {
	Get(ctx context.Context, key string) any
	Pop(ctx context.Context, key string) any
	Put(ctx context.Context, key string, val any)
	Destroy(context.Context) error
	RenewToken(context.Context) error
}

type Renderer interface {
	View(out io.Writer, page string, data any) error
}
