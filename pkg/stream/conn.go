package stream

import (
	"errors"
	"time"

	"github.com/gowsp/wsp/pkg/logger"
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
		logger.Error("encode msg %s error", cmd.String())
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
	signal *Signal
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
	logger.Debug("send connect request %s, id %s", w.config, w.id)
	if err = w.wan.write(data, time.Second*5); err != nil {
		return err
	}
	return w.signal.Wait(time.Second * 5)
}
func (w *link) ready(resp *msg.Data) error {
	err := parseResponse(resp)
	w.signal.Notify(err)
	return nil
}
func (w *link) active() error {
	logger.Debug("send connect response %s", w.config)
	res := msg.WspResponse{Code: msg.WspCode_SUCCESS, Data: ""}
	response, _ := proto.Marshal(&res)
	data, err := encode(w.id, msg.WspCmd_RESPOND, response)
	if err != nil {
		return err
	}
	return w.wan.write(data, time.Second*5)
}
func (w *link) Write(p []byte) (n int, err error) {
	logger.Trace("send data %s", w.config)
	data, err := encode(w.id, msg.WspCmd_TRANSFER, p)
	if err != nil {
		return 0, err
	}
	n = len(p)
	err = w.wan.write(data, time.Second*5)
	return
}
func (w *link) Close() error {
	logger.Debug("send interrupt request %s", w.config)
	w.wan.connect.Delete(w.id)
	data, err := encode(w.id, msg.WspCmd_INTERRUPT, []byte{})
	if err != nil {
		return err
	}
	return w.wan.write(data, time.Second*5)
}
