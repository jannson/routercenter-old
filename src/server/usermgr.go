package rcenter

import (
	"errors"

	"github.com/gorilla/websocket"
)

type UserConn struct {
	ws       *websocket.Conn
	u        string
	key      string
	lanAddr  string
	lanPort  string
	proto    string
	writeMsg chan []byte
}

type DeviceConn struct {
	u        string
	deviceId string
	mainWs   *websocket.Conn
	wsMap    map[string]*UserConn
	writeMsg chan []byte
}

type User struct {
	bus    *MessageBus
	name   string
	pass   string
	devMap map[string]*DeviceConn
}

func NewUser(bus *MessageBus, name string, pass string) *User {
	return &User{bus, name, pass, make(map[string]*DeviceConn)}
}

func (user *User) RegistDevice(dev *DeviceConn) {
	if old, ok := user.devMap[dev.deviceId]; ok {
		old.mainWs.Close()
	}

	user.devMap[dev.deviceId] = dev
}

func (user *User) UnregistDevice(deviceId string) error {
	if dev, ok := user.devMap[deviceId]; ok {
		dev.mainWs.Close()
		for k, v := range dev.wsMap {
			v.ws.Close()
			delete(dev.wsMap, k)
		}

		delete(user.devMap, deviceId)
		return nil
	} else {
		return errors.New("not found")
	}
}

func (user *User) RegistConn(conn *UserConn, deviceId string) error {
	if dev, ok := user.devMap[deviceId]; ok {
		if _, ok2 := dev.wsMap[conn.key]; ok2 {
			return errors.New("exists old conn")
		}
		dev.wsMap[conn.key] = conn
		return nil
	} else {
		return errors.New("device not found")
	}
}

func (user *User) UnregistConn(key string, deviceId string) error {
	if dev, ok := user.devMap[deviceId]; ok {
		if conn, ok2 := dev.wsMap[key]; ok2 {
			delete(dev.wsMap, key)
			conn.ws.Close()
			return nil
		}
	}

	return errors.New("conn not found")
}

func (user *User) RequestControl(msg *PMessage) *Message {
	user.bus.seqMsg <- msg

	//wait for response
	resp := <-msg.resp

	if resp.MType == PROTO_TYPE_ERROR {
		return nil
	}

	return resp
}
