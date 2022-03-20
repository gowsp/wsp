package proxy

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
	conn    *websocket.Conn
	routing *Routing
	proxy   Proxy
}

func NewWan(ws *websocket.Conn, proxy Proxy) *Wan {
	ws.SetReadLimit(1 << 23)
	return &Wan{conn: ws, routing: proxy.Routing(), proxy: proxy}
}
func (w *Wan) NewWriter(id string) io.WriteCloser {
	return &writer{id: id, ws: w.conn}
}
func (w *Wan) NewTCPChannel(id string, conn net.Conn) (io.WriteCloser, *TCPChannel) {
	writer := w.NewWriter(id)
	channel := NewTCPChannel(writer, conn, func() {
		w.routing.Delete(id)
	})
	w.routing.AddRepeater(id, channel)
	return writer, channel
}
func (w *Wan) HeartBeat(d time.Duration) {
	t := time.NewTicker(d)
	for {
		<-t.C
		if err := w.conn.Ping(context.Background()); err != nil {
			break
		}
	}
}
func (w *Wan) Write(p []byte) (n int, err error) {
	return len(p), w.conn.Write(context.Background(), websocket.MessageBinary, p)
}
func (w *Wan) Dail(id string, conf *msg.WspConfig, pending Pending) (err error) {
	addr, err := proto.Marshal(conf.ToReqeust())
	if err != nil {
		return err
	}
	data := Wrap(id, msg.WspCmd_CONNECT, addr)
	w.routing.connect.Store(id, pending)
	_, err = w.Write(data)
	if err != nil {
		w.routing.connect.Delete(id)
	}
	return err
}
func (w *Wan) reply(id string, succeed bool, message string) (err error) {
	res := msg.WspResponse{Data: message}
	if succeed {
		res.Code = msg.WspCode_SUCCESS
	} else {
		res.Code = msg.WspCode_FAILED
	}
	response, _ := proto.Marshal(&res)
	data := Wrap(id, msg.WspCmd_RESPOND, response)
	_, err = w.Write(data)
	return err
}
func (w *Wan) Succeed(id string) (err error) {
	return w.reply(id, true, "")
}
func (w *Wan) Close() {
	if w == nil || w.conn == nil {
		return
	}
	w.conn.Close(websocket.StatusNormalClosure, "normal close")
}
