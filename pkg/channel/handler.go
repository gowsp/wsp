package channel

import (
	"context"
	"io"
	"log"

	"nhooyr.io/websocket"
)

func (s *Channel) Serve() {
	go s.process()
	for {
		mt, data, err := s.conn.Read(context.Background())
		if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
			break
		}
		if err != nil {
			if err != io.EOF {
				log.Println("error reading webSocket message:", err)
			}
			break
		}
		s.msgs <- message{mt: mt, data: data}
	}
	close(s.msgs)
	s.session.Range(func(key, value interface{}) bool {
		value.(*Session).Interrupt()
		return true
	})
}
