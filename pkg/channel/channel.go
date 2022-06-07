package channel

import (
	"context"
	"errors"
	"log"
	"net"
	"sync"
	"time"

	"github.com/gowsp/wsp/pkg/msg"
	"google.golang.org/protobuf/proto"
	"nhooyr.io/websocket"
)

var ErrConnNotExist = errors.New("connection does not exist")

type Adapter func(id string, data *msg.WspRequest) error

type Channel struct {
	conn    *websocket.Conn
	session sync.Map
	adapter Adapter
	msgs    chan message
}
type Linker interface {
	InActive(err error)

	Active(session *Session) error
}

func New(conn *websocket.Conn, adapter Adapter) *Channel {
	conn.SetReadLimit(64 * 1024)
	return &Channel{conn: conn, adapter: adapter, msgs: make(chan message, 64)}
}
func (c *Channel) NewSession(id string, config *msg.WspConfig, l Linker, w Writer) *Session {
	return &Session{id: id, config: config, channel: c, linker: l, writer: w, msgs: make(chan *msg.Data, 64)}
}
func (c *Channel) NewTcpSession(id string, config *msg.WspConfig, l Linker, conn net.Conn) *Session {
	w := NewTcpWriter(conn, c.NewWriter(id))
	return c.NewSession(id, config, l, w)
}
func (c *Channel) Bridge(id string, config *msg.WspConfig, output *Channel) {
	writer := output.NewWsWriter(id)
	c.NewSession(id, config, nil, writer).active()
}
func (c *Channel) Reply(id string, code msg.WspCode, message string) (err error) {
	res := msg.WspResponse{Code: code, Data: message}
	response, _ := proto.Marshal(&res)
	data := encode(id, msg.WspCmd_RESPOND, response)
	_, err = c.Write(data)
	return err
}
func (c *Channel) Write(p []byte) (n int, err error) {
	return len(p), c.conn.Write(context.Background(), websocket.MessageBinary, p)
}
func (c *Channel) HeartBeat(d time.Duration) {
	t := time.NewTicker(d)
	for {
		<-t.C
		if err := c.conn.Ping(context.Background()); err != nil {
			break
		}
	}
}
func (c *Channel) serve(data *msg.Data) error {
	id := data.ID()
	switch data.Msg.Cmd {
	case msg.WspCmd_CONNECT:
		var req msg.WspRequest
		err := proto.Unmarshal(data.Payload(), &req)
		if err != nil {
			return err
		}
		go func() {
			err := c.adapter(id, &req)
			if err == nil {
				return
			}
			c.Reply(id, msg.WspCode_FAILED, err.Error())
		}()
		return nil
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
