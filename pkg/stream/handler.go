package stream

import (
	"errors"
	"fmt"
	"io"
	"runtime"
	"time"

	"github.com/gobwas/ws"
	"github.com/gowsp/wsp/pkg/logger"
	"github.com/gowsp/wsp/pkg/msg"
	"google.golang.org/protobuf/proto"
)

var ErrConnNotExist = errors.New("connection does not exist")

type Dialer interface {
	NewConn(data *msg.Data, req *msg.WspRequest) error
}

func NewHandler(dialer Dialer) *Handler {
	h := &Handler{
		dialer: dialer,
		cmds:   make(map[msg.WspCmd]handlerFunc),
		loop:   NewTaskPool(1, 256),
		event:  NewPollTaskPool(len(msg.WspCmd_value), 512),
		worker: NewPollTaskPool(runtime.NumCPU()*4, 1024),
	}
	h.init()
	return h
}

type Handler struct {
	dialer Dialer
	cmds   map[msg.WspCmd]handlerFunc
	loop   *TaskPool
	event  *PollTaskPool
	worker *PollTaskPool
}

func (w *Handler) Serve(wan *Wan) {
	for {
		data, mt, err := wan.read()
		if mt == ws.OpClose {
			break
		}
		if err == nil {
			w.process(wan, mt, data)
			continue
		}
		if err != io.EOF {
			logger.Error("error reading webSocket message: %s", err)
		}
		break
	}
	if wan.state.ClientSide() {
		w.loop.Wait()
		w.event.Wait()
		w.worker.Wait()
	} else {
		w.loop.Close()
		w.event.Close()
		w.worker.Close()
	}
	wan.connect.Range(func(key, value any) bool {
		value.(Conn).Interrupt()
		return true
	})
}

func (w *Handler) process(wan *Wan, mt ws.OpCode, data []byte) {
	w.loop.Add(func() {
		switch mt {
		case ws.OpBinary:
			m := new(msg.WspMessage)
			if err := proto.Unmarshal(data, m); err != nil {
				logger.Error("error unmarshal message: %s", err)
				return
			}
			w.event.Add(uint64(m.Cmd), func() {
				taskID := wan.getTaskID(m.Id)
				w.worker.Add(taskID, func() {
					err := w.serve(&msg.Data{Msg: m, Raw: &data}, wan)
					if errors.Is(err, ErrConnNotExist) {
						logger.Error("connect %s not exists", m.Id)
						data, _ := encode(m.Id, msg.WspCmd_INTERRUPT, []byte(err.Error()))
						wan.write(data, time.Minute)
					}
				})
			})
		default:
			logger.Error("unsupported message type %v", mt)
		}
	})
}

type handlerFunc func(data *msg.Data, wan *Wan) error

func (w *Handler) init() {
	w.cmds[msg.WspCmd_CONNECT] = func(data *msg.Data, wan *Wan) error {
		id := data.ID()
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
		return nil
	}
	w.cmds[msg.WspCmd_RESPOND] = func(data *msg.Data, wan *Wan) error {
		id := data.ID()
		logger.Debug("receive %s connect response", id)
		if val, ok := wan.waiting.LoadAndDelete(id); ok {
			return val.(ready).ready(data)
		}
		return ErrConnNotExist
	}
	w.cmds[msg.WspCmd_TRANSFER] = func(data *msg.Data, wan *Wan) error {
		id := data.ID()
		logger.Trace("receive %s transfer data", id)
		if val, ok := wan.connect.Load(id); ok {
			_, err := val.(Conn).Rewrite(data)
			if err != nil {
				val.(Conn).Close()
			}
			return nil
		}
		return ErrConnNotExist
	}
	w.cmds[msg.WspCmd_INTERRUPT] = func(data *msg.Data, wan *Wan) error {
		id := data.ID()
		logger.Debug("receive %s disconnect request", id)
		if val, ok := wan.connect.LoadAndDelete(id); ok {
			return val.(Conn).Interrupt()
		}
		return nil
	}
}
func (w *Handler) serve(data *msg.Data, wan *Wan) error {
	cmd, ok := w.cmds[data.Cmd()]
	if !ok {
		return fmt.Errorf("unknown command %s", data.Msg.Cmd)
	}
	return cmd(data, wan)
}
