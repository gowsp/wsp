package channel

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/gowsp/wsp/pkg/msg"
	"google.golang.org/protobuf/proto"
	"nhooyr.io/websocket"
)

func (s *Channel) Serve() {
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
		if err := s.process(mt, data); err != nil {
			log.Println(err)
		}
	}
	s.session.Range(func(key, value interface{}) bool {
		value.(*Session).Interrupt()
		return true
	})
}

func (s *Channel) process(mt websocket.MessageType, data []byte) error {
	switch mt {
	case websocket.MessageBinary:
		var m msg.WspMessage
		if err := proto.Unmarshal(data, &m); err != nil {
			return fmt.Errorf("error unmarshal message: %v", err)
		}
		err := s.serve(&msg.Data{Msg: &m, Raw: &data})
		if errors.Is(err, ErrConnNotExist) {
			data := encode(m.Id, msg.WspCmd_INTERRUPT, []byte(err.Error()))
			s.Write(data)
		}
	default:
		return fmt.Errorf("unsupported message type: %v", mt)
	}
	return nil
}
