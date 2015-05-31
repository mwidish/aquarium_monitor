package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/codegangsta/negroni"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/pjebs/restgate"
)

const indexPage = `
<h1>Login</h1>
 <form method="post" action="/login">
     <label for="name">User name</label>
     <input type="text" id="name" name="name">
     <label for="password">Password</label>
     <input type="password" id="password" name="password">
     <button type="submit">Login</button>
</form>
`

func IndexHandler() func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		user := getUserName(req)
		if user == "" {
			fmt.Fprint(w, indexPage)
		} else {
			http.Redirect(w, req, "/aquarium", 302)
		}
	}
}

func PhHandler() func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprint(w, "test")
	}
}

func LoginHandler() func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		user := req.FormValue("name")
		pw := req.FormValue("password")

		redirectTarget := "/"
		if user == "fishies" && pw == "lefty" {
			setSession(user, w)
			redirectTarget = "/aquarium"
		}
		http.Redirect(w, req, redirectTarget, 302)

	}
}

func setSession(user string, w http.ResponseWriter) {
	value := map[string]string{
		"username": user,
	}

	if encoded, err := cookieHandler.Encode("session", value); err == nil {
		cookie := &http.Cookie{
			Name:  "session",
			Value: encoded,
			Path:  "/"}
		http.SetCookie(w, cookie)
	}
}

func AquaHandler() func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		user := getUserName(req)
		if user == "" {
			http.Redirect(w, req, "/", 302)
		}
		fmt.Fprint(w, "Aqua")
	}
}

func getUserName(req *http.Request) (username string) {
	if cookie, err := req.Cookie("session"); err == nil {
		cookieValue := make(map[string]string)
		if err = cookieHandler.Decode("session", cookie.Value, &cookieValue); err == nil {
			username = cookieValue["username"]
		}

	}
	return username
}

func OrpHandler() func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		sql := SqlDB()
		defer sql.Close()

		rows, err := sql.Query("select * from ph")
		if err != nil {
			log.Fatal("Unable to retrieve ph", err)
		}

		defer rows.Close()
		for rows.Next() {
			var phValue int32
			if err := rows.Scan(&phValue); err != nil {
				log.Fatal(err)
			}
			fmt.Fprint(w, phValue)
		}
		if err := rows.Err(); err != nil {
			log.Fatal(err)
		}
		//fmt.Fprint(w, "test")
	}
}

func SqlDB() *sql.DB {

	DB_TYPE := "mysql"
	DB_HOST := "localhost"
	DB_PORT := "3306"
	DB_USER := "root"
	DB_NAME := "mydatabase"
	DB_PASSWORD := ""

	openString := DB_USER + ":" + DB_PASSWORD + "@tcp(" + DB_HOST + ":" + DB_PORT + ")/" + DB_NAME

	db, err := sql.Open(DB_TYPE, openString)
	if err != nil {
		return nil
	}

	return db
}

var cookieHandler = securecookie.New(
	securecookie.GenerateRandomKey(64),
	securecookie.GenerateRandomKey(32))

func main() {

	n := negroni.Classic()
	nRest := negroni.Classic()

	// set up REST to go through restgate
	restMux := mux.NewRouter()
	restMux.HandleFunc("/ph", PhHandler())
	restMux.HandleFunc("/orp", OrpHandler())
	sqlDb := SqlDB()
	defer sqlDb.Close()
	nRest.Use(restgate.New("X-Auth-Key", "X-Auth-Secret", restgate.Database,
		restgate.Config{DB: sqlDb, TableName: "users", Key: []string{"keys"}, Secret: []string{"secrets"}}))
	nRest.UseHandler(restMux)

	// No additional middleware, just Classic
	mainMux := mux.NewRouter()
	mainMux.HandleFunc("/", IndexHandler())
	mainMux.Handle("/ph", nRest)
	mainMux.Handle("/orp", nRest)
	mainMux.HandleFunc("/login", LoginHandler())
	mainMux.HandleFunc("/aquarium", AquaHandler())
	n.UseHandler(mainMux)
	n.Run(":80")
}
