package pkg

import (
	"context"
	"log"

	"github.com/gowsp/wsp/pkg/msg"
	"google.golang.org/protobuf/proto"
	"nhooyr.io/websocket"
)

func Wrap(id string, cmd msg.WspCmd, data []byte) []byte {
	msg := &msg.WspMessage{Id: id, Cmd: cmd, Data: data}
	res, err := proto.Marshal(msg)
	if err != nil {
		log.Println("error wrap message")
	}
	return res
}

type writer struct {
	id string
	ws *websocket.Conn
}

func (w *writer) Write(p []byte) (n int, err error) {
	data := Wrap(w.id, msg.WspCmd_FORWARD, p)
	return len(p), w.ws.Write(context.Background(), websocket.MessageBinary, data)
}
func (w *writer) Close() error {
	data := Wrap(w.id, msg.WspCmd_CLOSE, []byte{})
	return w.ws.Write(context.Background(), websocket.MessageBinary, data)
}
