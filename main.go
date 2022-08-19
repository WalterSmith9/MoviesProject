package main

import (
	"strconv"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"net/http"
	"sort"
	"time"
)

var database *sql.DB

var HTMLOpen = []byte(`
	<html>
	<body>
`)

var HTMLClose = []byte(`
	</body>
	</html>
`)

var SortForm = []byte(`
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
    <button type="submit">Sort</button>
  </div>
	</form>
`)

var FilterForm = []byte(`
  <form>
   <p>Choose between years:</p>
   <div>
	<input name="yearFrom" pattern="[0-9]{4}">
	<b> - </b>
	<input name="yearTo" pattern="[0-9]{4}">
	<br><input type="submit" value="Fiter">
   </div>
  </form>
`)

type User struct{
	id int
	login string
	password string
	moviesID []int
}

type film struct {
	ID int
	Name string
	Year int
	Director string
}

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

func isFound (slice []int, a int) bool {
	for _, n := range slice {
		if a == n {
			return true
		}
	}
	return false
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
		fmt.Fprintln(w, `<br> <a href="/wishlist">Список желаемого</a><br>`)
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

func allFilmsPage(w http.ResponseWriter, r *http.Request) {
	//checking if authorized
	session, err := r.Cookie("session_id")
	if err == http.ErrNoCookie {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	//collecting user's data
	user :=User{}
	user.login = session.Value
	row := database.QueryRow("SELECT id FROM users WHERE login = ?", user.login)
	row.Scan(&user.id)
	//adding movie to wishlist
	addFilm, err := strconv.Atoi(r.FormValue("addFilm"))
	if err == nil {
		_, err = database.Exec("INSERT INTO user_movie (userID, movieID) VALUES (?, ?);", user.id, addFilm)
		if err != nil{
			fmt.Println(err)
		}
	}
	//getting movies from user's wishlist
	rows, err := database.Query("Select movieID FROM user_movie WHERE userID=?", user.id)
	if err != nil {
		fmt.Println(err)
	}
	for rows.Next(){
		var mvID int
		err := rows.Scan(&mvID)
		if err != nil{
			fmt.Println(err)
			continue
		}
		user.moviesID = append(user.moviesID, mvID)
	}
	//getting movies from db and putting them into structure
	rows, err = database.Query("Select * from usersdb.movies")
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
	//setting filter parameters
	yearFrom, yearTo := 0, 10000
	if r.FormValue("yearFrom") != ""{
		yearFrom,_ = strconv.Atoi(r.FormValue("yearFrom"))
	}
	if r.FormValue("yearTo") != ""{
		yearTo,_ = strconv.Atoi(r.FormValue("yearTo"))
	}
	//setting sorting parameters and sorting right away
	method := r.FormValue("sortMethod")
	if r.FormValue("sortMethod") == "" {
		method = "name"
	}
	sortedMv := movies.sorted(method)
	//printing the list of movies according to filter parameters
	w.Write(HTMLOpen)
	w.Write([]byte(`<form id="addFilm"></form>`))
	for i := 0; i<len(movies.movies); i++{
		if sortedMv[i].Year >= yearFrom && sortedMv[i].Year <= yearTo {
			if  isFound(user.moviesID, sortedMv[i].ID){
				fmt.Fprintln(w,"+")
			}else {
				//pressing on that button will add movie to wishlist
				w.Write([]byte(`<input type="submit" form="addFilm" name="addFilm" value=` + strconv.Itoa(sortedMv[i].ID) + `>`))
			}
			fmt.Fprintln(w, sortedMv[i].Name + " | " + sortedMv[i].Director + " ", sortedMv[i].Year)
			w.Write([]byte(`<br>`))
		}
	}
	w.Write(SortForm)
	w.Write(FilterForm)
	w.Write(HTMLClose)
}

func wishlistPage(w http.ResponseWriter, r *http.Request) {
	//checking if authorized
	session, err := r.Cookie("session_id")
	if err == http.ErrNoCookie {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	//collecting user's data
	user :=User{}
	user.login = session.Value
	row := database.QueryRow("SELECT id FROM usersdb.users WHERE login = ?", user.login)
	row.Scan(&user.id)
	//deleting movie from wishlist
	deleteFilm, err := strconv.Atoi(r.FormValue("deleteFilm"))
	if err == nil {
		_, err = database.Exec("delete from usersdb.user_movie where userID = ? and movieID = ?;", user.id, deleteFilm)
		if err != nil{
			fmt.Println(err)
		}
	}
	//getting movies from db and putting them into structure
	rows, err := database.Query("SELECT movies.id, movies.name, movies.director, movies.year " +
		"FROM usersdb.movies, usersdb.user_movie where user_movie.userID = ? and movies.id = user_movie.movieID",
		user.id)
	if err != nil {
		fmt.Println(err)
	}
	defer rows.Close()
	wishlist := movieList{}
	for rows.Next(){
		mv := film{}
		err := rows.Scan(&mv.ID, &mv.Name, &mv.Director, &mv.Year)
		if err != nil{
			fmt.Println(err)
			continue
		}
		wishlist.movies = append(wishlist.movies, mv)
	}
	//setting filter parameters
	yearFrom, yearTo := 0, 10000
	if r.FormValue("yearFrom") != ""{
		yearFrom,_ = strconv.Atoi(r.FormValue("yearFrom"))
	}
	if r.FormValue("yearTo") != ""{
		yearTo,_ = strconv.Atoi(r.FormValue("yearTo"))
	}
	//setting sorting parameters and sorting right away
	method := r.FormValue("sortMethod")
	if r.FormValue("sortMethod") == "" {
		method = "name"
	}
	sortedMv := wishlist.sorted(method)
	//printing the wishlist according to filter parameters
	w.Write(HTMLOpen)
	w.Write([]byte(`<form id="deleteFilm"></form>`))
	for i := 0; i<len(wishlist.movies); i++{
		if sortedMv[i].Year >= yearFrom && sortedMv[i].Year <= yearTo {
			//pressing on that button will withdraw movie from wishlist
			w.Write([]byte(`<input type="submit" form="deleteFilm" name="deleteFilm" value=` + strconv.Itoa(sortedMv[i].ID) + `>`))
			fmt.Fprintln(w, sortedMv[i].Name + " | " + sortedMv[i].Director + " ", sortedMv[i].Year)
			w.Write([]byte(`<br>`))
		}
	}
	w.Write(SortForm)
	w.Write(FilterForm)
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
	http.HandleFunc("/films", allFilmsPage)
	http.HandleFunc("/wishlist", wishlistPage)
	staticHandler := http.StripPrefix(
		"/static/",
		http.FileServer(http.Dir("./sources")),
	)
	http.Handle("/static/", staticHandler)

	fmt.Println("starting server at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}