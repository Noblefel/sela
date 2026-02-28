package types

import (
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
	Mail               string `json:"mail"`
	MailerApi          string `json:"mailer_api"`
	MailerApiToken     string `json:"mailer_api_token"`
}

type DB interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

type Renderer interface {
	View(out io.Writer, page string, data any) error
}

type Mailer interface {
	Send(to, subject string, html []byte)
}
