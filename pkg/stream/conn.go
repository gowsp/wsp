package stream

import (
	"errors"
	"log"
	"time"

	"github.com/gowsp/wsp/pkg/msg"
	"google.golang.org/protobuf/proto"
)

type ready interface {
	ready(resp *msg.Data) error
}

func encode(id string, cmd msg.WspCmd, data []byte) ([]byte, error) {
	msg := &msg.WspMessage{Id: id, Cmd: cmd, Data: data}
	res, err := proto.Marshal(msg)
	if err != nil {
		log.Println("error wrap message")
		return nil, err
	}
	return res, nil
}
func parseResponse(data *msg.Data) error {
	var resp msg.WspResponse
	if err := proto.Unmarshal(data.Payload(), &resp); err != nil {
		return err
	}
	if resp.Code == msg.WspCode_FAILED {
		return errors.New(resp.Data)
	}
	return nil
}

type link struct {
	id  string
	wan *Wan

	config *msg.WspConfig
	done   chan error
}

func (w *link) open() error {
	addr, err := proto.Marshal(w.config.ToReqeust())
	if err != nil {
		return err
	}
	data, err := encode(w.id, msg.WspCmd_CONNECT, addr)
	if err != nil {
		return err
	}
	w.wan.waiting.Store(w.id, w)
	defer w.wan.waiting.Delete(w.id)
	log.Println("start connect", w.config)
	if err = w.wan.write(data, time.Second*5); err != nil {
		return err
	}
	select {
	case err := <-w.done:
		return err
	case <-time.After(time.Second * 5):
		return errors.New("timeout")
	}
}
func (w *link) ready(resp *msg.Data) error {
	w.done <- parseResponse(resp)
	return nil
}
func (w *link) active() error {
	res := msg.WspResponse{Code: msg.WspCode_SUCCESS, Data: ""}
	response, _ := proto.Marshal(&res)
	data, err := encode(w.id, msg.WspCmd_RESPOND, response)
	if err != nil {
		return err
	}
	return w.wan.write(data, time.Second*5)
}
func (w *link) Write(p []byte) (n int, err error) {
	data, err := encode(w.id, msg.WspCmd_TRANSFER, p)
	if err != nil {
		return 0, err
	}
	n = len(p)
	err = w.wan.write(data, time.Second*5)
	return
}
func (w *link) Close() error {
	w.wan.connect.Delete(w.id)
	data, err := encode(w.id, msg.WspCmd_INTERRUPT, []byte{})
	if err != nil {
		return err
	}
	return w.wan.write(data, time.Second*5)
}
