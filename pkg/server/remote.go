package server

import (
	"fmt"
	"log"
	"sync/atomic"

	"github.com/gowsp/wsp/pkg"
	"github.com/gowsp/wsp/pkg/msg"
	"google.golang.org/protobuf/proto"
)

func (r *Router) NewRemoteConn(id string, addr *msg.WspAddr) {
	if val, ok := r.server.network.Load(addr.Address); ok {
		log.Printf("start router bridge on %s\n", addr.Address)
		l := val.(*Router)

		registerAddr, ok := l.network.Load(addr.Address)
		if !ok {
			r.server.network.Delete(addr.Address)
			r.wan.CloseRemote(id, fmt.Sprintf("name %s not exist", addr.Address))
			return
		}
		if registerAddr.(*msg.WspAddr).Secret != addr.Secret {
			r.wan.CloseRemote(id, fmt.Sprintf("name %s secret error", addr.Address))
			return
		}

		err := l.wan.Dail(id, msg.WspType_LOCAL, addr.Address)
		if err != nil {
			r.wan.CloseRemote(id, fmt.Sprintf("name %s connect error", addr.Address))
			return
		}

		finish := func() {
			l.lan.Delete(id)
			r.lan.Delete(id)
		}

		inBrige := NewOutBridge(id, l.wan, finish)
		r.lan.Store(id, inBrige)

		outBrige := NewOutBridge(id, r.wan, finish)
		l.lan.Store(id, outBrige)

		go inBrige.Forward()
		go outBrige.Forward()
	} else {
		r.wan.CloseRemote(id, fmt.Sprintf("name %s not register", addr.Address))
	}
}

type RouterBridge struct {
	id      string
	out     *pkg.Wan
	closed  uint32
	stop    chan bool
	message chan *msg.WspMessage
	finish  func()
}

func NewOutBridge(id string, out *pkg.Wan, finish func()) pkg.Bridge {
	return &RouterBridge{
		id:      id,
		out:     out,
		stop:    make(chan bool),
		message: make(chan *msg.WspMessage, 64),
		finish:  finish,
	}
}

func (b *RouterBridge) Read(msg *msg.WspMessage) bool {
	if b.IsClosed() {
		return false
	}
	b.message <- msg
	return true
}
func (b *RouterBridge) IsClosed() bool {
	return atomic.LoadUint32(&b.closed) > 0
}
func (b *RouterBridge) Close() {
	if b.IsClosed() {
		return
	}
	b.stop <- false
}

func (b *RouterBridge) Forward() {
	go func() {
		for message := range b.message {
			data, err := proto.Marshal(message)
			if err != nil {
				log.Println("error wrap message")
				continue
			}
			_, err = b.out.Write(data)
			if err != nil {
				b.stop <- true
			}
		}
	}()
	stop := <-b.stop
	atomic.AddUint32(&b.closed, 1)
	close(b.message)
	if stop {
		b.out.CloseRemote(b.id, "stoped")
	}
	b.finish()
}
