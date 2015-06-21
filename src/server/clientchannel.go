package rcenter

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/gorilla/websocket"
)

func checkSameOrigin2(r *http.Request) bool {
	origin := r.Header["Origin"]
	if len(origin) == 0 {
		return true
	}
	u, err := url.Parse(origin[0])
	if err != nil {
		return false
	}

	s := GetHttpdGlobal()
	session, _ := s.session.Get(r, "auth-info")
	name := session.Values["login-user"]

	return (u.Host == r.Host) && len(name.(string)) > 0
}

var upgraderTunnel = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     checkSameOrigin2,
	Subprotocols:    []string{"tunnel-protocol"},
}

var upgraderClient = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     checkSameOrigin,
	Subprotocols:    []string{"client-protocol"},
}

func newTunnelConn(bus *MessageBus, u string, d string, addr uint32, port uint32, ws *websocket.Conn) *UserConn {
	rv, err := bus.Call(NewUserConn, bus, u, d, addr, port, ws)
	if err != nil {
		return nil
	}

	if rv[0].IsNil() {
		return nil
	}

	return rv[0].Interface().(*UserConn)
}

func (userConn *UserConn) CloseInner(s *ServerHttpd) {
	// if isClosed == 1, already closed
	if !atomic.CompareAndSwapInt32(&devConn.isClosed, 0, 1) {
		return
	}
	if len(userConn.u) > 0 {
		if userMgr, ok := s.bus.users[userConn.u]; ok {
			userMgr.UnregistConn(userConn.key, userConn.deviceId)
		}
	}

	if userConn.clientWs != nil {
		userConn.clientWs.Close()
	}
	if userConn.tunnelWs != nil {
		userConn.tunnelWs.Close()
	}
	close(userConn.clientMsg)
	close(userConn.tunnelMsg)
}

func (userConn *UserConn) Close(s *ServerHttpd) {
	s.bus.CallNoWait(userConn.CloseInner, s)
}

func (userConn *UserConn) tunnelWrite(s *ServerHttpd) {
	defer userConn.Close(s)

	for {
		select {
		case msg, ok := <-userConn.tunnelMsg:
			if ok {
				log.Println("begin write to tunnel msg", len(msg))
				userConn.tunnelWs.SetWriteDeadline(time.Now().Add(writeWait))
				if err := userConn.tunnelWs.WriteMessage(websocket.BinaryMessage, msg); err != nil {
					log.Println("write tunnel message error ", err.Error())
					return
				}
			} else {
				log.Println("got tunnel write channel error ")
			}
		}
	}
}

//The connection from browser
func (userConn *UserConn) tunnelRead(s *ServerHttpd) {
	defer userConn.Close(s)

	devConn.mainWs.SetReadLimit(maxMessageSize)

	for {
		_, b, err := userConn.tunnelWs.ReadMessage()
		if err != nil {
			s.context.Logger.Info("read tunnel message error %v\n", err)
			break
		}

		userConn.clientMsg <- b
	}
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

	ws, err := upgraderTunnel.Upgrade(w, r, nil)
	if err != nil {
		return 400, errors.New("Upgrade failed")
	}

	userConn := newTunnelConn(s.bus, name.(string), device, msg.lanAddr, msg.lanPort, ws)
	if nil == userConn {
		return 400, errors.New("create user conn failed")
	}
	defer userConn.Close(s)
	msg.sessionId = uint32(userConn.sessionId)

	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)
	binary.Write(writer, binary.BigEndian, &msg)
	writer.Flush()

	pmsg := NewPMessage(device, PROTO_TYPE_TUNNEL_REQ)
	pmsg.msg = buf.Bytes()
	pmsg.Length = uint32(len(pmsg.msg))
	respMsg := s.bus.RequestControl(name.(string), device, pmsg)
	if nil == respMsg {
		return 500, errors.New("request message error")
	}

	go userConn.tunnelWrite()
	userConn.tunnelRead()
	return 200, nil
}

func UpdateUserConnInner(bus *MessageBus, u string, d string, sessionId int, clientWs *websocket.Conn) *UserConn {
	if userMgr, ok := bus.users[u]; ok {
		if dev, ok2 := userMgr.devMap[d]; ok2 {
			if uc, ok3 := dev.wsMap[sessionId]; ok3 {
				uc.clientWs = clientWs
				return uc
			}
		}
	}

	return nil
}

func UpdateUserConn(bus *MessageBus, u string, d string, sessionId int, clientWs *websocket.Conn) *UserConn {
	rv, err := bus.Call(UpdateUserConnInner, u, d, sessionId, clientWs)
	if err != nil {
		return false
	}

	return rv[0].Interface().(*UserConn)
}

func (userConn *UserConn) clientWrite(s *ServerHttpd) {
	defer userConn.Close(s)

	for {
		select {
		case msg, ok := <-userConn.clientMsg:
			if ok {
				log.Println("begin write to client msg", len(msg))
				userConn.tunnelWs.SetWriteDeadline(time.Now().Add(writeWait))
				if err := userConn.tunnelWs.WriteMessage(websocket.BinaryMessage, msg); err != nil {
					log.Println("write client message error ", err.Error())
					return
				}
			} else {
				log.Println("got client write channel error ")
			}
		}
	}
}

func (userConn *UserConn) clientRead(s *ServerHttpd) {
	defer userConn.Close(s)

	userConn.clientWs.SetReadLimit(maxMessageSize)

	for {
		_, b, err := userConn.clientWs.ReadMessage()
		if err != nil {
			s.context.Logger.Info("read tunnel message error %v\n", err)
			break
		}

		userConn.tunnelMsg <- b
	}
}

func serveClientChannel(s *ServerHttpd, w http.ResponseWriter, r *http.Request) (int, error) {
	params := mux.Vars(r)
	device := params["device"]
	sessionid := params["sessionid"]
	u := params["user"]

	isession, err := strconv.Atoi(port)
	if err != nil {
		return 400, errors.New("convert session error")
	}

	ws, err := upgraderClient.Upgrade(w, r, nil)
	if err != nil {
		return 400, errors.New("Upgrade failed")
	}
	userConn := UpdateUserConn(s.bus, u.(string), d.(string), isession, ws)
	if userConn == nil {
		ws.Close()
		return 400, errors.New("session id not found")
	}

	defer userConn.Close(s)

	go userConn.clientWrite(s)

	userConn.clientRead(s)

	return 200, nil
}
