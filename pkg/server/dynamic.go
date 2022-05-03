package server

import (
	"log"
	"net"
	"time"

	"github.com/gowsp/wsp/pkg/channel"
	"github.com/gowsp/wsp/pkg/msg"
)

type localLinker struct {
	addr string
	conn net.Conn
}

func (l *localLinker) InActive(err error) {
	log.Println("close proxy", l.addr)
}

func (l *localLinker) Active(session *channel.Session) error {
	go func() {
		session.CopyFrom(l.conn)
		log.Println("close proxy", l.addr)
	}()
	return nil
}

func (c *conn) NewDynamic(id string, conf *msg.WspConfig) error {
	addr := conf.Address()
	log.Println("open proxy", addr)

	conn, err := net.DialTimeout(conf.Network(), addr, 5*time.Second)
	if err != nil {
		return err
	}
	l := &localLinker{conn: conn, addr: addr}
	return c.channel.NewSession(id, conf, l, nil).Ack()
}
