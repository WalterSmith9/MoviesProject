package main

import (
	"database/sql"
	"fmt"

	"html/template"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var database *sql.DB

type User struct {
	id       int
	login    string
	password string
	moviesID []int
}

type film struct {
	ID       int
	Name     string
	Year     int
	Director string
}

type movieList struct {
	movies []film
}

func (mv movieList) sorted(method string) []film {
	switch method {
	case "":
		method = "name"
		fallthrough
	case "name":
		sort.Slice(mv.movies, func(i, j int) bool {
			return mv.movies[i].Name < mv.movies[j].Name
		})

	case "year":
		sort.Slice(mv.movies, func(i, j int) bool {
			return mv.movies[i].Year < mv.movies[j].Year
		})
	case "director":
		sort.Slice(mv.movies, func(i, j int) bool {
			return mv.movies[i].Director < mv.movies[j].Director
		})
	default:
		fmt.Println("sorting failed")
	}
	return mv.movies
}

func (mv movieList) filtered(yearFrom, yearTo interface{}) []film {
	var yrFromConv, yrToConv int
	var err error

	switch yearFrom.(type) {
	case string:
		yrFromConv, err = strconv.Atoi(yearFrom.(string))
		if err != nil {
			yrFromConv = 0
		}
	case float32:
		yrFromConv = int(yearFrom.(float32))
	case float64:
		yrFromConv = int(yearFrom.(float64))
	case int:
		yrFromConv = yearFrom.(int)
	default:
		yrFromConv = 0
	}

	switch yearTo.(type) {
	case string:
		yrToConv, err = strconv.Atoi(yearTo.(string))
		if err != nil {
			yrToConv = 10000
		}
	case float32:
		yrToConv = int(yearTo.(float32))
	case float64:
		yrToConv = int(yearTo.(float64))
	case int:
		yrToConv = yearTo.(int)
	default:
		yrToConv = 0
	}

	var filteredList []film
	for _, currMv := range mv.movies {
		if currMv.Year >= yrFromConv && currMv.Year <= yrToConv {
			filteredList = append(filteredList, currMv)
		}
	}

	return filteredList
}

func isFound(slice []int, a int) bool {
	for _, n := range slice {
		if a == n {
			return true
		}
	}
	return false
}

func mainPage(w http.ResponseWriter, r *http.Request) {
	//authentication is done only with post method
	if r.Method == http.MethodPost {
		//getting password from DB by login
		login := r.FormValue("login")
		row := database.QueryRow("select uPassword from usersdb.users where login = ?", login)
		var password string
		err := row.Scan(&password)
		if err != nil {
			fmt.Println(err)
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		//checking if passwords from DB and client match
		if password != r.FormValue("password") {
			fmt.Println("Wrong password")
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		//setting cookies
		expiration := time.Now().Add(10 * time.Hour)
		cookie := http.Cookie{
			Name:    "session_id",
			Value:   login,
			Expires: expiration,
		}
		http.SetCookie(w, &cookie)
		fmt.Println("Right password by " + login)
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	session, err := r.Cookie("session_id")
	loggedIn := err != http.ErrNoCookie
	if loggedIn {
		//writing html for authorized users
		tmpl, err := template.New("").ParseFiles("sources/mainPage.html")
		if err != nil {
			panic(err)
		}
		tmplData := struct {
			Session *http.Cookie
		}{
			session,
		}
		err = tmpl.ExecuteTemplate(w, "mainPage.html", tmplData)
		if err != nil {
			panic(err)
		}
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

func signUpPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "sources/signupPage.html")
	return
}

func registerPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		//checking fo empty params, which are not acceptable
		if r.FormValue("login") == "" || r.FormValue("password") == "" {
			fmt.Println("Empty values")
			http.Redirect(w, r, "/signup", http.StatusFound)
			return
		}
		//creating structure with new user
		u := User{}
		u.login = r.FormValue("login")
		u.password = r.FormValue("password")
		//checking if user already exists
		row := database.QueryRow("select login from usersdb.users where login = ?", u.login)
		err := row.Scan()
		switch err {
		case nil:
			fmt.Println("User already exists")
			http.Redirect(w, r, "/signup", http.StatusFound)
			return
		case sql.ErrNoRows:
			fmt.Println("User doesn't exist")
		default:
			fmt.Println(err)
			http.Redirect(w, r, "/signup", http.StatusFound)
			return
		}
		//adding new user to DB
		_, err = database.Exec("INSERT INTO users (login, uPassword) VALUES (?, ?)", u.login, u.password)
		if err != nil {
			fmt.Println(err)
		}
		//setting cookies
		expiration := time.Now().Add(10 * time.Hour)
		cookie := http.Cookie{
			Name:    "session_id",
			Value:   u.login,
			Expires: expiration,
		}
		http.SetCookie(w, &cookie)
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
	return
}

func logoutPage(w http.ResponseWriter, r *http.Request) {
	//checking if authorized
	session, err := r.Cookie("session_id")
	if err == http.ErrNoCookie {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	//withdrawing cookies
	session.Expires = time.Now().AddDate(0, 0, -1)
	http.SetCookie(w, session)
	http.Redirect(w, r, "/", http.StatusFound)
}

func deleteAccountPage(w http.ResponseWriter, r *http.Request) {
	//checking if authorized
	session, err := r.Cookie("session_id")
	if err == http.ErrNoCookie {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	user := User{}
	user.login = session.Value
	//withdrawing cookies
	session.Expires = time.Now().AddDate(0, 0, -1)
	http.SetCookie(w, session)
	//deleting from DB
	_, err = database.Exec("delete from usersdb.users where login = ?", user.login)
	if err != nil {
		fmt.Println(err)
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func wishlistPage(w http.ResponseWriter, r *http.Request) {
	//checking if authorized
	session, err := r.Cookie("session_id")
	if err == http.ErrNoCookie {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	//collecting user's data
	user := User{}
	user.login = session.Value
	row := database.QueryRow("SELECT id FROM usersdb.users WHERE login = ?", user.login)
	row.Scan(&user.id)
	//deleting movie from wishlist
	deleteFilm, err := strconv.Atoi(r.FormValue("deleteFilm"))
	if err == nil {
		_, err = database.Exec("delete from usersdb.user_movie where userID = ? and movieID = ?;", user.id, deleteFilm)
		if err != nil {
			fmt.Println(err)
		}
	}
	//getting movies from db and putting them into structure
	rows, err := database.Query("SELECT movies.id, movies.name, movies.director, movies.year "+
		"FROM usersdb.movies, usersdb.user_movie WHERE user_movie.userID = ? AND movies.id = user_movie.movieID",
		user.id)
	if err != nil {
		fmt.Println(err)
	}
	defer rows.Close()
	wishlist := movieList{}
	for rows.Next() {
		mv := film{}
		err := rows.Scan(&mv.ID, &mv.Name, &mv.Director, &mv.Year)
		if err != nil {
			fmt.Println(err)
			continue
		}
		wishlist.movies = append(wishlist.movies, mv)
	}
	//sorting and filtering movieList
	wishlist.sorted(r.FormValue("sortMethod"))
	filteredMv := wishlist.filtered(r.FormValue("yearFrom"), r.FormValue("yearTo"))
	//Writing html file
	tmpl, err := template.New("").ParseFiles("sources/wishlistPage.html")
	if err != nil {
		panic(err)
	}
	tmplData := struct {
		FilmList []film
	}{
		filteredMv,
	}
	err = tmpl.ExecuteTemplate(w, "wishlistPage.html", tmplData)
	if err != nil {
		panic(err)
	}
}

func allFilmsPage(w http.ResponseWriter, r *http.Request) {
	//checking if authorized
	session, err := r.Cookie("session_id")
	if err == http.ErrNoCookie {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	//collecting user's data
	user := User{}
	user.login = session.Value
	row := database.QueryRow("SELECT id FROM users WHERE login = ?", user.login)
	row.Scan(&user.id)
	//adding movie to wishlist
	addFilm, err := strconv.Atoi(r.FormValue("addFilm"))
	if err == nil {
		_, err = database.Exec("INSERT INTO user_movie (userID, movieID) VALUES (?, ?);", user.id, addFilm)
		if err != nil {
			fmt.Println(err)
		}
	}
	//getting movies from user's wishlist
	rows, err := database.Query("Select movieID FROM user_movie WHERE userID=?", user.id)
	if err != nil {
		fmt.Println(err)
	}
	for rows.Next() {
		var mvID int
		err := rows.Scan(&mvID)
		if err != nil {
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
	for rows.Next() {
		mv := film{}
		err := rows.Scan(&mv.ID, &mv.Name, &mv.Director, &mv.Year)
		if err != nil {
			fmt.Println(err)
			continue
		}
		movies.movies = append(movies.movies, mv)
	}
	//sorting and filtering movieList
	movies.sorted(r.FormValue("sortMethod"))
	filteredMv := movies.filtered(r.FormValue("yearFrom"), r.FormValue("yearTo"))
	//Writing html file
	tmplFunc := template.FuncMap{
		"Found": isFound,
	}
	tmpl, err := template.New("").Funcs(tmplFunc).ParseFiles("sources/allFilmsPage.html")
	if err != nil {
		panic(err)
	}
	tmplData := struct {
		WishlistID []int
		FilmList   []film
	}{
		user.moviesID,
		filteredMv,
	}
	err = tmpl.ExecuteTemplate(w, "allFilmsPage.html", tmplData)
	if err != nil {
		panic(err)
	}
}

func testPage(w http.ResponseWriter, r *http.Request) {
	//checking if authorized
	session, err := r.Cookie("session_id")
	if err == http.ErrNoCookie {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	//collecting user's data
	user := User{}
	user.login = session.Value
	row := database.QueryRow("SELECT id FROM usersdb.users WHERE login = ?", user.login)
	row.Scan(&user.id)
	//deleting movie from wishlist
	deleteFilm, err := strconv.Atoi(r.FormValue("deleteFilm"))
	if err == nil {
		_, err = database.Exec("delete from usersdb.user_movie where userID = ? and movieID = ?;", user.id, deleteFilm)
		if err != nil {
			fmt.Println(err)
		}
	}
	//getting movies from db and putting them into structure
	rows, err := database.Query("SELECT movies.id, movies.name, movies.director, movies.year "+
		"FROM usersdb.movies, usersdb.user_movie WHERE user_movie.userID = ? AND movies.id = user_movie.movieID",
		user.id)
	if err != nil {
		fmt.Println(err)
	}
	defer rows.Close()
	wishlist := movieList{}
	for rows.Next() {
		mv := film{}
		err := rows.Scan(&mv.ID, &mv.Name, &mv.Director, &mv.Year)
		if err != nil {
			fmt.Println(err)
			continue
		}
		wishlist.movies = append(wishlist.movies, mv)
	}
	//sorting and filtering movieList
	wishlist.sorted(r.FormValue("sortMethod"))
	filteredMv := wishlist.filtered(r.FormValue("yearFrom"), r.FormValue("yearTo"))
	//Writing html file
	tmpl, err := template.New("").ParseFiles("sources/wishlistPage.html")
	if err != nil {
		panic(err)
	}
	tmplData := struct {
		FilmList []film
	}{
		filteredMv,
	}
	err = tmpl.ExecuteTemplate(w, "wishlistPage.html", tmplData)
	if err != nil {
		panic(err)
	}
}

func main() {
	db, err := sql.Open("mysql", "root:Riptide_Embassy73@/usersdb")
	if err != nil {
		log.Println(err)
	}
	database = db

	mux := http.NewServeMux()
	mux.HandleFunc("/", mainPage)
	mux.HandleFunc("/login", loginPage)
	mux.HandleFunc("/signup", signUpPage)
	mux.HandleFunc("/register", registerPage)
	mux.HandleFunc("/logout", logoutPage)
	mux.HandleFunc("/deleteAccount", deleteAccountPage)
	mux.HandleFunc("/films", allFilmsPage)
	mux.HandleFunc("/wishlist", wishlistPage)
	mux.HandleFunc("/test", testPage)
	staticHandler := http.StripPrefix(
		"/static/",
		http.FileServer(http.Dir("./sources")),
	)
	mux.Handle("/static/", staticHandler)
	server := http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	fmt.Println("starting server at :8080")
	log.Fatal(server.ListenAndServe())
}
