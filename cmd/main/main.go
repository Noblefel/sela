package main

import (
	"database/sql"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Noblefel/lensa"
	"github.com/Noblefel/sela/internal/handler"
	"github.com/Noblefel/sela/internal/types"
	"github.com/Noblefel/sela/internal/util"
	"github.com/alexedwards/scs/v2"
	_ "github.com/lib/pq"
)

var (
	conn    *sql.DB
	config  *types.Config
	session *scs.SessionManager
	app     *handler.Handlers
)

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)
	setConfigAndDB()

	conn.SetMaxIdleConns(25)
	conn.SetMaxOpenConns(25)

	// for scs session
	gob.Register(types.Auth{})
	gob.Register(types.FormUser{})
	gob.Register(types.FormArticle{})

	render := lensa.New("web/pages", "web/parts", ".tem")
	render.UseFuncs(util.TemplateFuncs)
	// render.UseCache()

	session = scs.New()
	app = handler.New(conn, session, render, config)

	log.Println("serving...")
	http.ListenAndServe("localhost:8080", route())
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
