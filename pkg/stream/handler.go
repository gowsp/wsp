package stream

import (
	"errors"
	"io"
	"time"

	"github.com/gowsp/wsp/pkg/logger"
	"github.com/gowsp/wsp/pkg/msg"
	"google.golang.org/protobuf/proto"
	"nhooyr.io/websocket"
)

var ErrConnNotExist = errors.New("connection does not exist")

type Dialer interface {
	NewConn(data *msg.Data, req *msg.WspRequest) error
}

type message struct {
	mt   websocket.MessageType
	data []byte
}

func NewHandler(dialer Dialer) *Handler {
	return &Handler{
		dialer: dialer,
	}
}

type Handler struct {
	msgs   chan message
	dialer Dialer
}

func (w *Handler) Serve(wan *Wan) {
	go w.process(wan)
	for {
		mt, data, err := wan.read()
		if err == nil {
			w.msgs <- message{mt: mt, data: data}
			continue
		}
		if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
			break
		}
		if err != io.EOF {
			logger.Error("error reading webSocket message: %s", err)
		}
		break
	}
	close(w.msgs)
	wan.connect.Range(func(key, value any) bool {
		value.(Lan).Interrupt()
		return true
	})
}

func (w *Handler) process(wan *Wan) {
	w.msgs = make(chan message, 64)
	for message := range w.msgs {
		switch message.mt {
		case websocket.MessageBinary:
			data := message.data
			m := new(msg.WspMessage)
			if err := proto.Unmarshal(data, m); err != nil {
				logger.Error("error unmarshal message: %s", err)
			}
			err := w.serve(&msg.Data{Msg: m, Raw: &data}, wan)
			if errors.Is(err, ErrConnNotExist) {
				logger.Error("connect %s not exists", m.Id)
				data, _ := encode(m.Id, msg.WspCmd_INTERRUPT, []byte(err.Error()))
				wan.write(data, time.Minute)
			}
		default:
			logger.Error("unsupported message type %v", message.mt)
		}
	}
}
func (w *Handler) serve(data *msg.Data, wan *Wan) error {
	id := data.ID()
	switch data.Msg.Cmd {
	case msg.WspCmd_CONNECT:
		req := new(msg.WspRequest)
		if err := proto.Unmarshal(data.Payload(), req); err != nil {
			logger.Error("invalid request data %s", err)
			wan.Reply(id, err)
			return err
		}
		go func() {
			logger.Debug("receive %s connect reqeust: %s", id, req)
			if err := w.dialer.NewConn(data, req); err != nil {
				logger.Error("error to open connect %s", req)
				wan.Reply(id, err)
			}
		}()
	case msg.WspCmd_RESPOND:
		logger.Debug("receive %s connect response", id)
		if val, ok := wan.waiting.Load(id); ok {
			return val.(ready).ready(data)
		}
		return ErrConnNotExist
	case msg.WspCmd_TRANSFER:
		logger.Trace("receive %s transfer data", id)
		if val, ok := wan.connect.Load(id); ok {
			val.(Lan).Rewrite(data)
			return nil
		}
		return ErrConnNotExist
	case msg.WspCmd_INTERRUPT:
		logger.Debug("receive %s disconnect request", id)
		if val, ok := wan.connect.LoadAndDelete(id); ok {
			return val.(Lan).Interrupt()
		}
	default:
		logger.Error("unknown command %s", data.Msg.Cmd)
	}
	return nil
}
