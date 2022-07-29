package main

import (
	"fmt"
	"log"
	"net/http"
)

var loginFormTmpl = []byte(`
<html>
	<head>
	<link rel="stylesheet" type="text/css" href="/static/MainPage.css">
	</head>
	<body>
		<form action="/" class="ui-form" method="post">
		<h3>Войти на сайт</h3>
		<div class="form-row">
		<input type="text" id="login" name="login" required autocomplete="off"><label for="login">Login</label>
		</div>
		<div class="form-row">
		<input type="password" id="password" required autocomplete="off"><label for="password">Password</label>
		</div>
		<p><input type="submit" value="Войти"></p>
		</form>
	</body>
</html>
`)

func mainPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		//w.Write(loginFormTmpl)
		http.ServeFile(w, r, "sources/MainPage.html")
		return
	}


	inputLogin := r.FormValue("login")
	fmt.Fprintln(w, "you enter: ", inputLogin)
}

func main() {
	http.HandleFunc("/", mainPage)

	staticHandler := http.StripPrefix(
		"/static/",
		http.FileServer(http.Dir("./sources")),
	)
	http.Handle("/static/", staticHandler)

	fmt.Println("starting server at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}