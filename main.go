package main

import (
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"
)

var authentication = map[string]string{
	"Walter":"asd",
	"Anthony":"zxc",
	"RandomGuy":"fgh",
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

type film struct {
	ID int
	Name string
	Year int
	Director string
}

var films =[]film{
	{1, "The Wolf Of Wall Street", 2013, "Martin Scorsese"},
	{2, "The Hateful Eight", 2015, "Quentin Tarantino"},
	{3, "Inception", 2010, "Christopher Nolan"},
	{4, "The Departed", 2006, "Martin Scorsese"},
	{5, "Enemy", 2013, "Denis Villeneuve"},
	{6, "Nomadland", 2020, "Chloe Zhao"},
}

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

func mainPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if (authentication[r.FormValue("login")] != r.FormValue("password")){
			http.Redirect(w, r, "/login", http.StatusFound)
		}

		expiration := time.Now().Add(10 * time.Hour)
		cookie := http.Cookie{
			Name: "session_id",
			Value: r.FormValue("login"),
			Expires: expiration,
		}
		http.SetCookie(w, &cookie)
		http.Redirect(w, r, "/", http.StatusFound)
	}

	session, err := r.Cookie("session_id")
	loggedIn := (err != http.ErrNoCookie)
	if loggedIn {
		fmt.Fprintln(w, `<a href="/logout">logout</a>`)
		fmt.Fprintln(w, "Welcome, "+session.Value)
		fmt.Fprintln(w, `<br> <a href="/films">Фильмы</a>`)
	} else {
		//fmt.Fprintln(w, `<a href="/login">login</a>`)
		http.Redirect(w, r, "/login", http.StatusFound)
	}

}

func loginPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.ServeFile(w, r, "sources/loginPage.html")
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
	http.HandleFunc("/", mainPage)
	http.HandleFunc("/login", loginPage)
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