package websocketadp

import (
	"github.com/gorilla/websocket"
	"net"
	"time"
)

func NewWsAdp(ws *websocket.Conn) *WsAdp {
	return &WsAdp{ws}
}

type WsAdp struct {
	*websocket.Conn
}

func (ws *WsAdp) Read(b []byte) (n int, err error) {
	_, msg, errw := ws.ReadMessage()
	if errw != nil {
		return 0, err
	}
	copy(b, msg)
	return len(msg), nil
}

func (ws *WsAdp) Write(b []byte) (n int, err error) {
	return len(b), ws.WriteMessage(websocket.BinaryMessage, b)
}

func (ws *WsAdp) SetDeadline(t time.Time) error {
	return nil
}

func (ws *WsAdp) AsConn() net.Conn {
	return ws
}
