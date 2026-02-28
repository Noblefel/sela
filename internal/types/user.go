package types

import (
	"time"

	"github.com/Noblefel/sela/internal/util"
)

type User struct {
	Id        int
	Email     string
	Username  string
	Name      string
	Bio       string
	Avatar    string
	Admin     bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (u User) Authorize(auth *Auth) bool {
	// do nil check for convenience in templates
	return auth != nil && u.Id == auth.Id
}

type FormUser struct {
	Name     string
	Username string // mutated
	Bio      string
}

func (f *FormUser) Validate() string {
	// act like a regexp
	f.Username = util.Slug(f.Username)
	if f.Name == "" || f.Username == "" {
		return "name or username is empty"
	}
	if len(f.Name) > 100 {
		return "name is too long"
	}
	if len(f.Username) > 40 {
		return "username is too long"
	}
	if len(f.Bio) > 250 {
		return "bio is too long"
	}
	return ""
}
