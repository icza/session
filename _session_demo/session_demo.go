/*
This is a session demo application.

It starts a web server locally on port 8080, and registers a handler to "/demo".

Code demonstrates session access, creation and removal.
*/
package main

import (
	"github.com/icza/session"
	"html/template"
	"log"
	"net/http"
)

var templ = template.Must(template.New("").Parse(page))

// myHandler handles everything: page/form rendering, processing login form submits, logout submits.
// If login is successful, a new session is created. If logout is successful, session is removed.
func myHandler(w http.ResponseWriter, r *http.Request) {
	m := map[string]interface{}{}

	sess := session.Get(r)
	if sess != nil {
		// Already logged in
		if r.FormValue("Logout") != "" {
			session.Remove(sess, w) // Logout user
			sess = nil
		} else {
			sess.SetAttr("Count", sess.Attr("Count").(int)+1)
		}
	} else {
		// Not logged in
		if r.FormValue("Login") != "" {
			if userName := r.FormValue("UserName"); userName != "" && r.FormValue("Password") == "a" {
				// Successful login. New session with initial constant and variable attributes:
				sess = session.NewSessionOptions(&session.SessOptions{
					CAttrs: map[string]interface{}{"UserName": userName},
					Attrs:  map[string]interface{}{"Count": 1},
				})
				session.Add(sess, w)
			} else {
				m["InvalidLogin"] = true
			}
		}
	}

	if sess != nil {
		m["UserName"] = sess.CAttr("UserName")
		m["Count"] = sess.Attr("Count")
	}

	if err := templ.Execute(w, m); err != nil {
		log.Println("Error:", err)
	}
}

func main() {
	// For testing purposes, we want cookies to be sent over HTTP too (not just HTTPS):
	session.Global.Close()
	session.Global = session.NewCookieManagerOptions(session.NewInMemStore(), &session.CookieMngrOptions{AllowHTTP: true})

	log.Println("Session demo is about to start. Visit: localhost:8080/demo")
	http.HandleFunc("/demo", myHandler)
	http.ListenAndServe(":8080", nil)
}

const page = `<html><body>
{{if .InvalidLogin}}<p style="color:red">Invalid user name or password!</p>{{end}}

{{if .UserName}}
	<p>Hello <b>{{.UserName}}</b>! Since login you visited <b>{{.Count}}</b> times! <a href="/demo">Refresh!</a></p>
{{end}}

<form method="post" action="/demo">
	{{if .UserName}}
		<input type="submit" name="Logout" value="Logout">
	{{else}}
		<label for="UserNameId" style="width:100px; display: inline-block">User name:</label>
		<input type="text" name="UserName" id="UserNameId"><br>
		<label for="PasswordId" style="width:100px; display: inline-block">Password:</label>
		<input type="password" name="Password" id="PasswordId">
		<span style="font-style:italic; font-size: 90%">Tip: use 'a' to login ;)</span><br>
		<input type="submit" name="Login" value="Login">
	{{end}}
</form>
</body></html>`
