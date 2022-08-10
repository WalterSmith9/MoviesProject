package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"net/http"
	"sort"
	"time"
)

type User struct{
	id int
	login string
	password string
}

type film struct {
	ID int
	Name string
	Year int
	Director string
}

var database *sql.DB

var HTMLOpen = []byte(`
	<html>
	<body>
`)

var HTMLClose = []byte(`
	</body>
	</html>
`)

var RBForm = []byte(`
<form>
  <p>Sort by:</p>
  <div>
    <input type="radio" id="sortChoice1"
     name="sortMethod" value="name">
    <label for="contactChoice1">Name</label>

    <input type="radio" id="sortChoice2"
     name="sortMethod" value="director">
    <label for="contactChoice2">Director</label>

    <input type="radio" id="sortChoice3"
     name="sortMethod" value="year">
    <label for="contactChoice3">Year</label>
  </div>
  <div>
    <button type="submit">Submit</button>
  </div>
	</form>
`)

type movieList struct{
	movies []film
}

func (mv movieList) sorted (method string) []film{
	if method == "name"{
		sort.Slice(mv.movies, func(i, j int) bool {
			return mv.movies[i].Name<mv.movies[j].Name
		})
	}
	if method == "year"{
		sort.Slice(mv.movies, func(i, j int) bool {
			return mv.movies[i].Year<mv.movies[j].Year
		})
	}
	if method == "director"{
		sort.Slice(mv.movies, func(i, j int) bool {
			return mv.movies[i].Director<mv.movies[j].Director
		})
	}
	return mv.movies
}

func mainPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		login := r.FormValue("login")
		row:= database.QueryRow("select uPassword from usersdb.users where login = ?", login)
		var password string
		err := row.Scan(&password)
		if err != nil{
			fmt.Println(err)
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		if password != r.FormValue("password") {
			fmt.Println("Wrong password")
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		expiration := time.Now().Add(10 * time.Hour)
		cookie := http.Cookie{
			Name: "session_id",
			Value: login,
			Expires: expiration,
		}
		http.SetCookie(w, &cookie)
		fmt.Println("Right password")
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	session, err := r.Cookie("session_id")
	loggedIn := err != http.ErrNoCookie
	if loggedIn {
		fmt.Fprintln(w, `<a href="/logout">logout</a>`)
		fmt.Fprintln(w, "Welcome, " + session.Value)
		fmt.Fprintln(w, `<br> <a href="/films">Фильмы</a><br>`)
	} else {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

}

func loginPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.ServeFile(w, r, "sources/loginPage.html")
		return
	}
}

func signUpPage(w http.ResponseWriter, r *http.Request){
	http.ServeFile(w, r, "sources/signupPage.html")
	return
}

func registerPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if r.FormValue("login") == "" || r.FormValue("password") == ""{
			fmt.Println("Empty values")
			http.Redirect(w, r, "/signup", http.StatusFound)
			return
		}

		u := User{}
		u.login = r.FormValue("login")
		u.password = r.FormValue("password")
		row := database.QueryRow("select login from usersdb.users where login = ?", u.login)
		err := row.Scan()
		if err != nil {
			fmt.Println(err)
		}
		if err != sql.ErrNoRows {
			fmt.Println("User already exists")
			http.Redirect(w, r, "/signup", http.StatusFound)
			return
		}

		row = database.QueryRow("SELECT MAX(id) FROM users")
		err = row.Scan(&u.id)
		if err != nil {
			fmt.Println(err)
		}
		u.id++
		_, err = database.Exec("INSERT INTO users (id, login, uPassword) VALUES (?, ?, ?)", u.id, u.login, u.password)
		if err != nil {
			fmt.Println(err)
		}

		expiration := time.Now().Add(10 * time.Hour)
		cookie := http.Cookie{
			Name: "session_id",
			Value: u.login,
			Expires: expiration,
		}
		http.SetCookie(w, &cookie)
		http.Redirect(w,r,"/", http.StatusFound)
		return
	}
	http.Redirect(w,r,"/", http.StatusFound)
	return
}

func logoutPage(w http.ResponseWriter, r *http.Request) {
	session, err := r.Cookie("session_id")
	if err == http.ErrNoCookie {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	session.Expires = time.Now().AddDate(0, 0, -1)
	http.SetCookie(w, session)
	http.Redirect(w, r, "/", http.StatusFound)
}

func filmsPage(w http.ResponseWriter, r *http.Request) {
	_, err := r.Cookie("session_id")
	if err == http.ErrNoCookie {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	w.Write(HTMLOpen)
	rows, err := database.Query("Select * from usersdb.movies")
	if err != nil {
		fmt.Println(err)
	}
	defer rows.Close()
	movies := movieList{}
	for rows.Next(){
		mv := film{}
		err := rows.Scan(&mv.ID, &mv.Name, &mv.Director, &mv.Year)
		if err != nil{
			fmt.Println(err)
			continue
		}
		movies.movies = append(movies.movies, mv)
	}

	method := r.FormValue("sortMethod")
	if r.FormValue("sortMethod") == "" {
		method = "name"
	}
	sortedMv := movies.sorted(method)
	for i := 0; i<len(movies.movies); i++{
		fmt.Fprintln(w, sortedMv[i].Name + " " + sortedMv[i].Director + " ", sortedMv[i].Year)
		w.Write([]byte(`<br>`))
	}
	w.Write(RBForm)
	w.Write(HTMLClose)
}

func main() {
	db, err := sql.Open("mysql", "root:Riptide_Embassy73@/usersdb")
	if err != nil {
		log.Println(err)
	}
	database = db

	http.HandleFunc("/", mainPage)
	http.HandleFunc("/login", loginPage)
	http.HandleFunc("/signup", signUpPage)
	http.HandleFunc("/register", registerPage)
	http.HandleFunc("/logout", logoutPage)
	http.HandleFunc("/films", filmsPage)

	staticHandler := http.StripPrefix(
		"/static/",
		http.FileServer(http.Dir("./sources")),
	)
	http.Handle("/static/", staticHandler)

	fmt.Println("starting server at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}