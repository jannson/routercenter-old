package rcenter

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
)

var httpdGlobal *ServerHttpd

type ServerHttpd struct {
	context *ServerContext
}

func GetHttpdGlobal() *ServerHttpd {
	return httpdGlobal
}

type appHandler struct {
	*ServerHttpd
	h func(*ServerHttpd, http.ResponseWriter, *http.Request) (int, error)
}

func (ah appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	status, err := ah.h(ah.ServerHttpd, w, r)
	if err != nil {
		ah.ServerHttpd.context.Logger.Debug("HTTP %d: %q\n", status, err)
		switch status {
		case http.StatusNotFound:
			http.NotFound(w, r)
			// And if we wanted a friendlier error page, we can
			// now leverage our context instance - e.g.
			// err := ah.renderTemplate(w, "http_404.tmpl", nil)
		case http.StatusInternalServerError:
			http.Error(w, http.StatusText(status), status)
		default:
			http.Error(w, http.StatusText(status), status)
		}
	}
}

func serveHome(s *ServerHttpd, w http.ResponseWriter, r *http.Request) (int, error) {
	//var homeTempl = template.Must(template.ParseFiles("statics/test1.html"))
	//w.Header().Set("Content-Type", "text/html; charset=utf-8")
	//homeTempl.Execute(w, r.Host)

	io.WriteString(w, "<html>contentstorage home<br/>")
	io.WriteString(w, "<a href='/static/test1.html' target='_blank'>core emulator</a><br/>")
	io.WriteString(w, "<a href='/static/uploadtest.html' target='_blank'>image upload test</a><br/>")
	io.WriteString(w, "<a href='/static/audiotest.html' target='_blank'>audio upload test</a><br/>")
	io.WriteString(w, "</html>")

	return 200, nil
}

func StartServer(c *ServerContext) {
	addr := fmt.Sprintf("%s:%d", c.Config.System.Host, c.Config.System.Port)
	router := mux.NewRouter()
	httpd := &ServerHttpd{context: c}
	httpdGlobal = httpd

	router.Handle("/", appHandler{httpd, serveHome}).Methods("GET")
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./statics/"))))

	http.Handle("/", router)

	c.Logger.Info("server start run :  %s", addr)
	http.ListenAndServe(addr, router)
}