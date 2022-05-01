package channel

import (
	"fmt"

	"github.com/gowsp/wsp/pkg/msg"
	"google.golang.org/protobuf/proto"
	"nhooyr.io/websocket"
)

func (s *Socket) process(mt websocket.MessageType, data []byte) error {
	switch mt {
	case websocket.MessageBinary:
		var m msg.WspMessage
		if err := proto.Unmarshal(data, &m); err != nil {
			return fmt.Errorf("error unmarshal message: %v", err)
		}
		s.channel.Read(&msg.Data{Msg: &m, Raw: &data})
	default:
		return fmt.Errorf("unsupported message type: %v", mt)
	}
	return nil
}
