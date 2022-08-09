package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"net/http"
	"sort"
	"strconv"
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

var films =[]film{
	{1, "The Wolf Of Wall Street", 2013, "Martin Scorsese"},
	{2, "The Hateful Eight", 2015, "Quentin Tarantino"},
	{3, "Inception", 2010, "Christopher Nolan"},
	{4, "The Departed", 2006, "Martin Scorsese"},
	{5, "Enemy", 2013, "Denis Villeneuve"},
	{6, "Nomadland", 2020, "Chloe Zhao"},
}

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

func (mv movieList) sorted (metho string) []film{
	if (metho == "name"){
		sort.Slice(mv.movies, func(i, j int) bool {
			return mv.movies[i].Name<mv.movies[j].Name
		})
	}
	if (metho == "year"){
		sort.Slice(mv.movies, func(i, j int) bool {
			return mv.movies[i].Year<mv.movies[j].Year
		})
	}
	if (metho == "director"){
		sort.Slice(mv.movies, func(i, j int) bool {
			return mv.movies[i].Director<mv.movies[j].Director
		})
	}
	return mv.movies
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {

	rows, err := database.Query("select * from usersdb.users")
	if err != nil {
		log.Println(err)
	}
	defer rows.Close()
	users := []User{}

	for rows.Next(){
		u := User{}
		err := rows.Scan(&u.id, &u.login, &u.password)
		if err != nil{
			fmt.Println(err)
			continue
		}
		users = append(users, u)
	}
	fmt.Fprintln(w, users)
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

func signupPage(w http.ResponseWriter, r *http.Request){
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

		login := r.FormValue("login")
		row := database.QueryRow("select login from usersdb.users where login = ?", login)
		err := row.Scan()
		if err != nil {
			fmt.Println(err)
		}
		if err != sql.ErrNoRows {
			fmt.Println("User already exists")
			http.Redirect(w, r, "/signup", http.StatusFound)
			return
		}

		var id int
		row = database.QueryRow("SELECT MAX(id) FROM users")
		err = row.Scan(&id)
		if err != nil {
			fmt.Println(err)
		}
		id++
		_, err = database.Exec("INSERT INTO users (id, login, uPassword) VALUES (?, ?, ?)", id, login, r.FormValue("password"))
		if err != nil {
			fmt.Println(err)
		}

		expiration := time.Now().Add(10 * time.Hour)
		cookie := http.Cookie{
			Name: "session_id",
			Value: login,
			Expires: expiration,
		}
		http.SetCookie(w, &cookie)
		http.Redirect(w,r,"/", http.StatusFound)
		return
	}
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
	w.Write([]byte(`
	<html>	
	<body>
`))
	movies := movieList{films}
	method := r.FormValue("sortMethod")
	if (r.FormValue("sortMethod") == ""){
		method = "name"
	}
	sortedMv := movies.sorted(method)
	for i := 0; i<len(films); i++{
		fmt.Fprintln(w, sortedMv[i].Name + " " + sortedMv[i].Director + " " + strconv.Itoa(sortedMv[i].Year))
		w.Write([]byte(`<br>`))
	}
	w.Write(RBForm)
	w.Write([]byte(`
	</body>
	</html>
`))
}

func main() {
	db, err := sql.Open("mysql", "root:Riptide_Embassy73@/usersdb")
	if err != nil {
		log.Println(err)
	}
	database = db

	http.HandleFunc("/", mainPage)
	http.HandleFunc("/index", IndexHandler)
	http.HandleFunc("/login", loginPage)
	http.HandleFunc("/signup", signupPage)
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