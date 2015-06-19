package rcenter

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"text/template"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
)

type ServerHttpd struct {
	sync.RWMutex
	context *ServerContext
	session *sessions.CookieStore
	bus     *MessageBus
	users   map[string]*User
}

var httpdGlobal *ServerHttpd

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

	io.WriteString(w, "<html>Hello<br/>")
	io.WriteString(w, "</html>")

	return 200, nil
}

func getUserMgr(s *ServerHttpd, u string) *User {
	defer func() {
		s.RUnlock()
	}()
	s.RLock()

	if userMgr, ok := s.users[u]; ok {
		return userMgr
	}

	return nil
}

func (s *ServerHttpd) checkUserLogin(u string, p string) bool {
	defer func() {
		s.RUnlock()
	}()
	s.RLock()

	if userMgr, ok := s.users[u]; ok {
		if userMgr.pass == p {
			return true
		}
	}

	return false
}

func serveLogin(s *ServerHttpd, w http.ResponseWriter, r *http.Request) (int, error) {
	if r.Method == "GET" {
		var homeTempl = template.Must(template.ParseFiles("statics/login.html"))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		homeTempl.Execute(w, r.Host)
		return 200, nil

	} else {
		name := r.FormValue("username")
		pass := r.FormValue("password")
		redirectTarget := "/"
		if name != "" && pass != "" && s.checkUserLogin(name, pass) {
			session, _ := s.session.Get(r, "auth-info")
			session.Values["login-user"] = name
			session.Save(r, w)
			http.Redirect(w, r, redirectTarget, http.StatusSeeOther)
			return 200, nil
		}
	}

	io.WriteString(w, "<html>login error!</html>")
	return 200, nil
}

func serveTunnel(s *ServerHttpd, w http.ResponseWriter, r *http.Request) (int, error) {
	params := mux.Vars(r)
	device := params["device"]
	ip := params["ip"]
	port := params["port"]
	device = strings.Replace(device, "-", ":", -1)

	session, _ := s.session.Get(r, "auth-info")
	name := session.Values["login-user"]

	var msg MsgTunnel
	msg.lanAddr = inet_aton(ip)
	iport, err := strconv.Atoi(port)
	if err != nil {
		return 400, errors.New("parse port error")
	}
	msg.lanPort = uint32(iport)
	usrMgr := getUserMgr(s, name.(string))
	if nil == usrMgr {
		return 400, errors.New("user not found")
	}

	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)
	binary.Write(writer, binary.BigEndian, &msg)
	writer.Flush()

	pmsg := NewPMessage(device)
	pmsg.msg = buf.Bytes()
	pmsg.Length = uint32(len(pmsg.msg))
	respMsg := usrMgr.RequestControl(device, pmsg)
	if nil == respMsg {
		io.WriteString(w, "error")
	} else {
		io.WriteString(w, string(respMsg.msg))
	}

	return 200, nil
}

func StartServer(c *ServerContext) {
	addr := fmt.Sprintf("%s:%d", c.Config.System.Host, c.Config.System.Port)
	router := mux.NewRouter()
	httpd := &ServerHttpd{context: c, session: sessions.NewCookieStore([]byte("something-very-secret-heihei")), bus: NewMessageBus(), users: make(map[string]*User)}
	httpdGlobal = httpd

	//start bus
	go httpd.bus.Run()

	router.Handle("/", appHandler{httpd, serveHome}).Methods("GET")
	router.Handle("/__login", appHandler{httpd, serveLogin})
	router.Handle("/__tunnel/{device}/{ip}/{port}", appHandler{httpd, serveTunnel})
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./statics/"))))
	router.Handle("/__main_channel", appHandler{httpd, serveMainChannel})

	http.Handle("/", router)

	c.Logger.Info("server start run :  %s", addr)
	http.ListenAndServe(addr, router)
}
