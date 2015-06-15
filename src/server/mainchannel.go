package rcenter

import (
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
}

type MainChannel struct {
	ws *websocket.Conn
}

func serveMainChannel(s *ServerHttpd, w http.ResponseWriter, r *http.Request) (int, error) {
	ws, err := upgrader.Upgrade(w, r, nil)
	return 0, nil
}
