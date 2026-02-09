package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"strings"
	"time"

	"github.com/Noblefel/sela/internal/types"
	"github.com/Noblefel/sela/internal/util"
	_ "github.com/lib/pq"
)

var (
	conn   *sql.DB
	config *types.Config
	spacer = strings.NewReplacer(
		"-", " ",
		"_", " ",
		"X", " ",
		"x", " ",
		"0", " ",
		"Z", " ",
		"z", " ",
	)
)

func main() {
	setConfigAndDB()
	seedUsers(5)
	seedArticles(50)
}

func setConfigAndDB() {
	f, err := os.Open("env.json")
	if err != nil {
		panic(err)
	}
	if err = json.NewDecoder(f).Decode(&config); err != nil {
		panic(err)
	}
	f.Close()
	// dsn := fmt.Sprintf(`user=%s password=%s dbname=%s sslmode=disable`,
	// 	config.DB_User,
	// 	config.DB_Password,
	// 	config.DB_Name,
	// )
	dsn := fmt.Sprintf(`user=postgres dbname=sela sslmode=disable`)
	if conn, err = sql.Open("postgres", dsn); err != nil {
		panic(err)
	}
	if err = conn.Ping(); err != nil {
		panic(err)
	}
}

func seedUsers(n int) {
	var sb strings.Builder
	t := time.Now().UnixMilli()

	sb.WriteString("INSERT INTO users (email, name, username, bio, avatar, created_at) VALUES")

	for i := range n {
		uniq := fmt.Sprintf("u_%d_%d", t, i)

		sb.WriteString(fmt.Sprintf(
			` ('%s@test.com', '%s', '%s', '%s', '', '%s'), `,
			uniq,
			spacer.Replace(util.RandomString(20)),
			uniq,
			spacer.Replace(util.RandomString(80)),
			randomCreated(-2, 0, 0),
		))
	}

	// to close the batch insert
	sb.WriteString(fmt.Sprintf(
		` ('%s@test.com', '%s', '%s', '%s', '', '%s'); `,
		fmt.Sprintf("u_%d", t),
		spacer.Replace(util.RandomString(20)),
		fmt.Sprintf("u_%d", t),
		spacer.Replace(util.RandomString(80)),
		randomCreated(-2, 0, 0),
	))

	res, err := conn.Exec(sb.String())
	if err != nil {
		panic(err)
	}

	rows, _ := res.RowsAffected()
	fmt.Printf("--- %d users inserted\n", rows)
}

func seedArticles(n int) {
	var sb strings.Builder
	t := time.Now().UnixMilli()

	var ids []int

	rows, err := conn.Query("SELECT id FROM users;")
	if err != nil {
		panic(err)
	}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			panic(err)
		}
		ids = append(ids, id)
	}
	rows.Close()

	sb.WriteString("INSERT INTO articles (user_id, title, slug, excerpt, content, image, created_at) VALUES")

	for i := range n {
		uniq := fmt.Sprintf("a_%d_%d", t, i)

		sb.WriteString(fmt.Sprintf(
			` ('%d', '%s', '%s', '%s', '%s', '%s', '%s'), `,
			ids[rand.IntN(len(ids)-1)],
			// set key "the" to test the search handler
			"The "+spacer.Replace(util.RandomString(50)),
			uniq,
			spacer.Replace(util.RandomString(200)),
			"[]",
			"",
			randomCreated(-2, 0, 0),
		))
	}

	// to close the batch insert
	sb.WriteString(fmt.Sprintf(
		` ('%d', '%s', '%s', '%s', '%s', '%s', '%s'); `,
		ids[rand.IntN(len(ids)-1)],
		"The "+spacer.Replace(util.RandomString(50)),
		fmt.Sprintf("a_%d", t),
		spacer.Replace(util.RandomString(200)),
		"[]",
		"",
		randomCreated(-2, 0, 0),
	))

	res, err := conn.Exec(sb.String())
	if err != nil {
		panic(err)
	}

	count, _ := res.RowsAffected()
	fmt.Printf("--- %d articles inserted\n", count)
}

func randomCreated(years, months, day int) string {
	from := time.Now().AddDate(years, months, day)
	ranges := int64(time.Since(from))
	from = from.Add(time.Duration(rand.Int64N(ranges)))
	return from.Format(time.RFC3339)
}
