package stream

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/gowsp/wsp/pkg/logger"
	"github.com/gowsp/wsp/pkg/msg"
	"github.com/segmentio/ksuid"
	"google.golang.org/protobuf/proto"
)

func NewWan(conn net.Conn, state ws.State) *Wan {
	return &Wan{
		conn:  conn,
		state: state,
	}
}

type Wan struct {
	conn  net.Conn
	state ws.State
	lock  sync.Mutex

	waiting sync.Map
	connect sync.Map
}

func (w *Wan) read() ([]byte, ws.OpCode, error) {
	return wsutil.ReadData(w.conn, w.state)
}
func (w *Wan) write(data []byte, timeout time.Duration) error {
	w.lock.Lock()
	defer w.lock.Unlock()
	return wsutil.WriteMessage(w.conn, w.state, ws.OpBinary, data)
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
		w.lock.Lock()
		if err := wsutil.WriteMessage(w.conn, w.state, ws.OpPing, []byte{}); err != nil {
			w.lock.Unlock()
			w.conn.Close()
			break
		}
		w.lock.Unlock()
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
	if err := link.open(); err != nil {
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
	logger.Debug("send connect response %s, error: %s", id, err)
	response, _ := proto.Marshal(&res)
	data, _ := encode(id, msg.WspCmd_RESPOND, response)
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
