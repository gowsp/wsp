package stream

import (
	"context"
	"io"
	"net"
	"sync"
	"time"

	"github.com/gowsp/wsp/pkg/msg"
	"github.com/segmentio/ksuid"
	"google.golang.org/protobuf/proto"
	"nhooyr.io/websocket"
)

func NewWan(ws *websocket.Conn) *Wan {
	ws.SetReadLimit(64 * 1024)
	return &Wan{ws: ws}
}

type Wan struct {
	ws *websocket.Conn

	waiting sync.Map
	connect sync.Map
}

func (w *Wan) read() (websocket.MessageType, []byte, error) {
	return w.ws.Read(context.Background())
}
func (w *Wan) write(data []byte, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return w.ws.Write(ctx, websocket.MessageBinary, data)
}
func (w *Wan) newLink(id string, config *msg.WspConfig) *link {
	return &link{
		id:     id,
		wan:    w,
		config: config,
		done:   make(chan error, 1),
	}
}

func (w *Wan) HeartBeat(d time.Duration) {
	t := time.NewTicker(d)
	for {
		<-t.C
		if err := w.ws.Ping(context.Background()); err != nil {
			break
		}
	}
}
func (w *Wan) DialTCP(local net.Conn, remote *msg.WspConfig) (io.WriteCloser, error) {
	id := ksuid.New().String()
	writer := w.newLink(id, remote)
	if err := writer.open(); err != nil || remote.IsRemoteType() {
		return nil, err
	}
	conn := newTCP(local, writer)
	w.connect.Store(id, conn)
	return conn, nil
}
func (w *Wan) DialHTTP(remote *msg.WspConfig) (net.Conn, error) {
	id := ksuid.New().String()
	link := w.newLink(id, remote)
	if err := link.open(); err != nil || remote.IsRemoteType() {
		return nil, err
	}
	conn := newLan(&net.TCPAddr{}, link)
	w.connect.Store(id, conn)
	return conn, nil
}
func (w *Wan) Accept(id string, local net.Conn, config *msg.WspConfig) (io.WriteCloser, error) {
	link := w.newLink(id, config)
	conn := newTCP(local, link)
	w.connect.Store(id, conn)
	if err := link.active(); err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}
func (w *Wan) Reply(id string, message error) (err error) {
	var res msg.WspResponse
	if message == nil {
		res = msg.WspResponse{Code: msg.WspCode_SUCCESS}
	} else {
		res = msg.WspResponse{Code: msg.WspCode_FAILED, Data: message.Error()}
	}
	response, _ := proto.Marshal(&res)
	data, err := encode(id, msg.WspCmd_RESPOND, response)
	if err != nil {
		return err
	}
	return w.write(data, time.Minute)
}
func (w *Wan) Bridge(req *msg.Data, config *msg.WspConfig, rwan *Wan) error {
	p := &bridge{
		input:  rwan,
		output: w,
		config: config,
		signal: make(chan struct{}, 1),
	}
	return p.connect(req)
}
