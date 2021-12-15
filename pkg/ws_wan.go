package pkg

import (
	"context"
	"io"
	"net"
	"time"

	"github.com/gowsp/wsp/pkg/msg"
	"google.golang.org/protobuf/proto"
	"nhooyr.io/websocket"
)

type Wan struct {
	ws     *websocket.Conn
	binary net.Conn
}

func NewWan(ws *websocket.Conn) *Wan {
	ws.SetReadLimit(1 << 23)
	binary := websocket.NetConn(context.Background(), ws, websocket.MessageBinary)
	return &Wan{ws: ws, binary: binary}
}
func (w *Wan) NewWriter(id string) io.WriteCloser {
	return &writer{id: id, ws: w.ws}
}
func (w *Wan) HeartBeat(d time.Duration) {
	t := time.NewTicker(d)
	for {
		<-t.C
		w.ws.Ping(context.Background())
	}
}
func (w *Wan) Read() (websocket.MessageType, []byte, error) {
	return w.ws.Read(context.Background())
}
func (w *Wan) Write(p []byte) (n int, err error) {
	return len(p), w.ws.Write(context.Background(), websocket.MessageBinary, p)
}
func (w *Wan) Dail(id string, wspType msg.WspType, address string) (err error) {
	addr, err := proto.Marshal(&msg.WspAddr{Type: wspType, Address: address})
	if err != nil {
		return err
	}
	data := Wrap(id, msg.WspCmd_CONN_REQ, addr)
	_, err = w.Write(data)
	return err
}
func (w *Wan) SecretDail(id string, wspType msg.WspType, address, secret string) (err error) {
	addr, err := proto.Marshal(&msg.WspAddr{Type: wspType, Address: address, Secret: secret})
	if err != nil {
		return err
	}
	data := Wrap(id, msg.WspCmd_CONN_REQ, addr)
	_, err = w.Write(data)
	return err
}
func (w *Wan) Reply(id string, succeed bool) (err error) {
	var val byte
	if succeed {
		val = 1
	}
	data := Wrap(id, msg.WspCmd_CONN_REP, []byte{val})
	_, err = w.Write(data)
	return err
}
func (w *Wan) Close() {
	w.ws.Close(websocket.StatusNormalClosure, "normal close")
}
func (w *Wan) CloseRemote(id, message string) {
	data := Wrap(id, msg.WspCmd_CLOSE, []byte(message))
	w.Write(data)
}
