package channel

import (
	"errors"
	"fmt"
	"log"

	"github.com/gowsp/wsp/pkg/msg"
	"google.golang.org/protobuf/proto"
	"nhooyr.io/websocket"
)

type message struct {
	mt   websocket.MessageType
	data []byte
}

func (s *Channel) process() {
	for message := range s.msgs {
		switch message.mt {
		case websocket.MessageBinary:
			s.processBinary(message.data)
		default:
			log.Println("unsupported message type", message.mt)
		}
	}
}
func (s *Channel) processBinary(data []byte) error {
	var m msg.WspMessage
	if err := proto.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("error unmarshal message: %v", err)
	}
	err := s.serve(&msg.Data{Msg: &m, Raw: &data})
	if errors.Is(err, ErrConnNotExist) {
		data := encode(m.Id, msg.WspCmd_INTERRUPT, []byte(err.Error()))
		s.Write(data)
	}
	return err
}
