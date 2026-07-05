package types

import "time"

type Comment struct {
	Id        int
	UserId    int
	ArticleId int
	Comment   string
	CreatedAt time.Time
	UpdatedAt time.Time

	User    User
	Article Article
}

func (c Comment) Authorize(auth *Auth) bool {
	return auth != nil && auth.Id == c.UserId
}

type FormComment struct {
	Comment string `json:"comment"`
}

func (f FormComment) Validate() string {
	if f.Comment == "" {
		return "Comment is empty"
	}
	if len(f.Comment) > 1999 {
		return "Comment is too long"
	}
	return ""
}
