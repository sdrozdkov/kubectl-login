package oauthclient

import (
	"html/template"
	"net/http"
)

var clsTmpl = template.Must(template.New("close.html").Parse("<html><head><script>window.open('','_self').close();</script></head><body></body></html>"))

func renderTemplate(w http.ResponseWriter, tmpl *template.Template, data interface{}) error {
	err := tmpl.Execute(w, data)
	if err != nil {
		return err
	}
	return nil
}
