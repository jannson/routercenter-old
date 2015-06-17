package rcenter

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"
	"unsafe"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
)

func checkSameOrigin(r *http.Request) bool {
	origin := r.Header["Origin"]
	if len(origin) == 0 {
		return true
	}
	u, err := url.Parse(origin[0])
	if err != nil {
		return false
	}
	log.Println("origin is ", origin, u.Host, r.Host)
	return u.Host == r.Host
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     checkSameOrigin,
	Subprotocols:    []string{"tunnel-protocol"},
}

func (devConn *DeviceConn) write(mt int, payload []byte) error {
	devConn.mainWs.SetWriteDeadline(time.Now().Add(writeWait))
	return devConn.mainWs.WriteMessage(mt, payload)
}

func (devConn *DeviceConn) writePump(s *ServerHttpd) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		devConn.mainWs.Close()
	}()

	for {
		select {
		case <-ticker.C:
			//s.context.Logger.Info("websocket closed because of ping timeout")
			if err := devConn.write(websocket.PingMessage, []byte{}); err != nil {
				//if error, return
				s.context.Logger.Debug("websocket closed because of ping error")
				return
			}
		}
	}
}

//TODO http://stackoverflow.com/questions/23174362/packing-struct-in-golang-in-bytes-to-talk-with-c-application
func (devConn *DeviceConn) readPump(s *ServerHttpd) {
	defer func() {
		devConn.mainWs.Close()
	}()

	devConn.mainWs.SetReadLimit(maxMessageSize)
	devConn.mainWs.SetReadDeadline(time.Now().Add(pongWait))
	devConn.mainWs.SetPongHandler(func(string) error {
		devConn.mainWs.SetReadDeadline(time.Now().Add(pongWait))
		s.context.Logger.Debug("websocket got pong")
		return nil
	})

	b := make([]byte, maxMessageSize)
	for {
		log.Println("begin nextRead")
		_, reader, err := devConn.mainWs.NextReader()
		if err != nil {
			s.context.Logger.Info("newReader error %v\n", err)
			break
		} else {
			length, err := reader.Read(b)
			if err != nil {
				s.context.Logger.Info("read message error %v\n", err)
				break
			}

			var mHeader MessageHeader
			if length < int(unsafe.Sizeof(mHeader)) {
				s.context.Logger.Error("ignore error package %v\n", err)
				continue
			}

			//parse message header
			buf := bytes.NewBuffer(b)
			//dec := gob.NewDecoder(buffer)
			if nil != binary.Read(buf, binary.BigEndian, &mHeader) {
				s.context.Logger.Error("error read header %v\n", err)
				continue
			}

			s.context.Logger.Info("0x%x %d\n", mHeader.Magic, mHeader.Length)
		}

	}
}

func serveMainChannel(s *ServerHttpd, w http.ResponseWriter, r *http.Request) (int, error) {
	log.Println("in serveMainChannel")
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return 404, fmt.Errorf("Upgrade error")
	}
	s.context.Logger.Info("Got new connection\n")

	devConn := &DeviceConn{mainWs: ws, wsMap: make(map[string]*UserConn), writeMsg: make(chan *Message, 100)}
	devConn.readPump(s)

	return 200, nil
}
