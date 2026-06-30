package types

import (
	"time"

	"github.com/Noblefel/sela/internal/util"
	"github.com/kennygrant/sanitize"
)

// from github.com/kennygrant/sanitize with just added "style" attribute to allow text alignment
var allowedTags = []string{"h1", "h2", "h3", "h4", "h5", "h6", "div", "span", "hr", "p", "br", "b", "i", "strong", "em", "ol", "ul", "li", "a", "img", "pre", "code", "blockquote", "article", "section"}
var allowedAttributes = []string{"id", "class", "src", "href", "title", "alt", "name", "rel", "style"}

type Article struct {
	Id        int
	UserId    int
	Title     string
	Slug      string
	Excerpt   string
	Content   string
	Image     string
	Likes     int
	Liked     bool
	Views     int
	Comments  int
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt time.Time
	User      User
}

func (a Article) Authorize(auth *Auth) bool {
	// do nil check for convenience in templates
	return auth != nil && a.UserId == auth.Id
}

type FormArticle struct {
	Title   string
	Excerpt string
	Content string // mutated, sanitized
	Slug    string // mutated from title
}

func (f *FormArticle) Validate() string {
	if f.Slug = util.Slug(f.Title); f.Slug == "" {
		return "title is empty"
	}
	if len(f.Title) > 150 || len(f.Excerpt) > 500 {
		return "title or except is too long"
	}
	content, err := sanitize.HTMLAllowing(f.Content, allowedTags, allowedAttributes)
	if err != nil {
		return "content is malformed"
	}
	if len(content) < 100 {
		return "content is too short"
	}
	f.Content = content
	return ""
}

type ArticleDraft struct {
	Id        int    `json:"id"`
	UserId    int    `json:"user_id"`
	Title     string `json:"title"`
	Excerpt   string `json:"excerpt"`
	Content   string `json:"content"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (a ArticleDraft) Authorize(auth *Auth) bool {
	return auth != nil && auth.Id == a.UserId
}

type FormArticleDraft struct {
	Title   string `json:"title"`
	Excerpt string `json:"excerpt"`
	Content string `json:"content"` // sanitized
}

func (f *FormArticleDraft) Validate() string {
	if f.Title == "" {
		return "title is empty"
	}
	if len(f.Title) > 150 || len(f.Excerpt) > 500 {
		return "title or except is too long"
	}
	return ""
}
