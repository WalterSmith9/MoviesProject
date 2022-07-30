package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

var authentication = map[string]string{
	"Walter":"asd",
	"Anthony":"zxc",
	"RandomGuy":"fgh",
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


func main() {
	http.HandleFunc("/", mainPage)
	http.HandleFunc("/login", loginPage)
	http.HandleFunc("/logout", logoutPage)

	staticHandler := http.StripPrefix(
		"/static/",
		http.FileServer(http.Dir("./sources")),
	)
	http.Handle("/static/", staticHandler)

	fmt.Println("starting server at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}