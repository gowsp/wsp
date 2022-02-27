package proxy

import (
	"context"
	"errors"
	"log"

	"github.com/gowsp/wsp/pkg/msg"
	"google.golang.org/protobuf/proto"
	"nhooyr.io/websocket"
)

func (w *Wan) Serve() {
	for {
		_, data, err := w.conn.Read(context.Background())
		if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
			break
		}
		if err != nil {
			log.Println("error reading webSocket message:", err)
			break
		}
		var m msg.WspMessage
		if proto.Unmarshal(data, &m) != nil {
			log.Println("error unmarshal message:", err)
			continue
		}
		w.process(&msg.Data{Msg: &m, Raw: &data})
	}
	w.proxy.Close()
}

func (w *Wan) process(data *msg.Data) {
	switch data.Cmd() {
	case msg.WspCmd_CONNECT:
		go func() {
			err := w.proxy.NewConn(data.Msg)
			if err != nil {
				w.reply(data.ID(), false, err.Error())
			}
		}()
	default:
		err := w.routing.Routing(data)
		if errors.Is(err, ErrConnNotExist) {
			w.interrupt(data.ID(), err.Error())
		}
	}
}

func (w *Wan) interrupt(id, message string) {
	data := Wrap(id, msg.WspCmd_INTERRUPT, []byte(message))
	w.Write(data)
}
