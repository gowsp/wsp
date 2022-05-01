package channel

import (
	"context"
	"errors"
	"log"
	"sync"

	"github.com/gowsp/wsp/pkg/msg"
	"github.com/segmentio/ksuid"
	"google.golang.org/protobuf/proto"
	"nhooyr.io/websocket"
)

var ErrConnNotExist = errors.New("connection does not exist")

type Channel struct {
	conn    *websocket.Conn
	session sync.Map
	adapter func(data *msg.WspConfig) error
}

func (c *Channel) NewSession(config *msg.WspConfig, h *Handler) *Session {
	return &Session{id: ksuid.New().String(), config: config, channel: c, handler: h}
}
func (c *Channel) Write(p []byte) (n int, err error) {
	return len(p), c.conn.Write(context.Background(), websocket.MessageBinary, p)
}
func (c *Channel) Read(data *msg.Data) error {
	id := data.ID()
	switch data.Msg.Cmd {
	case msg.WspCmd_CONNECT:
		var req msg.WspRequest
		err := proto.Unmarshal(data.Payload(), &req)
		if err != nil {
			return err
		}
		conf, err := req.ToConfig()
		if err != nil {
			return err
		}
		return c.adapter(conf)
	case msg.WspCmd_RESPOND:
		if val, ok := c.session.Load(id); ok {
			return val.(*Session).SynAck(data)
		}
		return ErrConnNotExist
	case msg.WspCmd_TRANSFER:
		if val, ok := c.session.Load(id); ok {
			return val.(*Session).Transport(data)
		}
		return ErrConnNotExist
	case msg.WspCmd_INTERRUPT:
		if val, ok := c.session.Load(id); ok {
			return val.(*Session).Interrupt()
		}
	default:
		log.Println("unknown command")
	}
	return nil
}
