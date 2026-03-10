package util

import (
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"strings"
	"time"
	"unicode"
)

func RandomString(n int) string {
	s := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz1234567890-_"
	b := make([]byte, n)

	for i := range n {
		b[i] = s[rand.Intn(len(s))]
	}

	return string(b)
}

func Slug(s string) string {
	var sb strings.Builder
	var stripNext bool

	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			if stripNext {
				sb.WriteRune('-')
				stripNext = false
			}
			continue
		}
		sb.WriteRune(unicode.ToLower(r))
		stripNext = true
	}

	return strings.TrimSuffix(sb.String(), "-")
}

var TemplateFuncs = map[string]any{
	"html": func(s string) template.HTML {
		return template.HTML(s)
	},
	"forwardslash": func(url string) string {
		return strings.ReplaceAll(url, "\\", "/")
	},
	"avatar": func(src string) string {
		if strings.Contains(src, "googleusercontent") {
			return src
		} else if src != "" {
			return "/images/" + src
		} else {
			return "/static/pfp.png"
		}
	},
	"since": func(t time.Time) string {
		m := time.Since(t).Minutes()
		hr := m / 60
		d := hr / 24

		if d >= 1 {
			return fmt.Sprintf("%.0fd ago", d)
		} else if hr >= 1 {
			return fmt.Sprintf("%.0fh ago", hr)
		} else {
			return fmt.Sprintf("%.0fm ago", m)
		}
	},
	"has": strings.Contains,
}

// i don't know much about message brokers and workers
func Background(do func()) {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Println("RECOVERED (BACKGROUND)", err)
			}
		}()
		do()
	}()
}
