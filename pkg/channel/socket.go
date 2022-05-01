package channel

import (
	"context"
	"log"

	"nhooyr.io/websocket"
)

type Socket struct {
	channel *Channel
}

func NewSocket(conn *websocket.Conn) *Socket {
	return &Socket{channel: &Channel{conn: conn}}
}

func (s *Socket) Serve() {
	for {
		mt, data, err := s.channel.conn.Read(context.Background())
		if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
			break
		}
		if err != nil {
			log.Println("error reading webSocket message:", err)
			break
		}
		if err := s.process(mt, data); err != nil {
			log.Println(err)
		}
	}
}
