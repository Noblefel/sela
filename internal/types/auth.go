package types

import (
	"time"
)

// user data in session after login
type Auth struct {
	Id          int
	Name        string
	Username    string
	Avatar      string
	LastRefresh time.Time
	// TODO: HasNotification bool
}

func (a Auth) ShouldRefresh() bool {
	return time.Since(a.LastRefresh) > 5*time.Minute
}

type GoogleUser struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Picture string `json:"picture"`
}

type ResetEmail struct {
	Token     string
	UserId    int
	Code      string
	Email     string
	CreatedAt time.Time
}

func (r ResetEmail) Authorize(auth *Auth) bool {
	// do nil check for convenience in templates
	return auth != nil && r.UserId == auth.Id
}
