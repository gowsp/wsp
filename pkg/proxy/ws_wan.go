package proxy

import (
	"context"
	"io"
	"time"

	"github.com/gowsp/wsp/pkg/msg"
	"google.golang.org/protobuf/proto"
	"nhooyr.io/websocket"
)

type Wan struct {
	conn *websocket.Conn
}

func NewWan(ws *websocket.Conn) *Wan {
	ws.SetReadLimit(1 << 23)
	return &Wan{conn: ws}
}
func (w *Wan) NewWriter(id string) io.WriteCloser {
	return &writer{id: id, ws: w.conn}
}
func (w *Wan) HeartBeat(d time.Duration) {
	t := time.NewTicker(d)
	for {
		<-t.C
		w.conn.Ping(context.Background())
	}
}
func (w *Wan) Read() (websocket.MessageType, []byte, error) {
	return w.conn.Read(context.Background())
}
func (w *Wan) Write(p []byte) (n int, err error) {
	return len(p), w.conn.Write(context.Background(), websocket.MessageBinary, p)
}
func (w *Wan) Dail(id string, conf *msg.WspConfig) (err error) {
	addr, err := proto.Marshal(conf.ToReqeust())
	if err != nil {
		return err
	}
	data := Wrap(id, msg.WspCmd_CONNECT, addr)
	_, err = w.Write(data)
	return err
}
func (w *Wan) Reply(id string, succeed bool, message string) (err error) {
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
	return w.Reply(id, true, "")
}
func (w *Wan) Close() {
	w.conn.Close(websocket.StatusNormalClosure, "normal close")
}
func (w *Wan) CloseRemote(id, message string) {
	data := Wrap(id, msg.WspCmd_INTERRUPT, []byte(message))
	w.Write(data)
}
