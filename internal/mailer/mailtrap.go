package mailer

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/Noblefel/sela/internal/types"
)

type Mailtrap struct {
	email    string
	token    string
	endpoint string
}

func NewMailtrap(c *types.Config) *Mailtrap {
	return &Mailtrap{
		email:    c.Mail,
		token:    c.MailerApiToken,
		endpoint: c.MailerApi,
	}
}

func (mt Mailtrap) Send(to, subject string, html []byte) {
	// for cleaner handler
	if len(html) == 0 {
		log.Output(2, "error reading email file")
		return
	}

	var buf bytes.Buffer

	mail := map[string]any{
		"from":    map[string]string{"email": mt.email},
		"to":      []map[string]string{{"email": to}},
		"subject": subject,
		"html":    string(html),
	}

	if err := json.NewEncoder(&buf).Encode(mail); err != nil {
		log.Output(2, err.Error())
		return
	}

	req, err := http.NewRequest("POST", mt.endpoint, &buf)
	if err != nil {
		log.Output(2, err.Error())
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+mt.token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Output(2, err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		log.Output(2, "mailtrap: "+string(b))
	}
}
