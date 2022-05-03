package client

import (
	"log"
	"net"

	"github.com/gowsp/wsp/pkg/channel"
	"github.com/gowsp/wsp/pkg/msg"
	"github.com/segmentio/ksuid"
)

func (c *Wspc) LocalForward() {
	for _, val := range c.config.Local {
		conf, err := msg.NewWspConfig(msg.WspType_LOCAL, val)
		if err != nil {
			log.Println("forward local error,", err)
			continue
		}
		go c.ListenLocal(conf)
	}
}
func (c *Wspc) ListenLocal(conf *msg.WspConfig) {
	log.Println("listen local on channel", conf.Channel())
	local, err := net.Listen(conf.Network(), conf.Address())
	if err != nil {
		log.Println(err)
		return
	}
	for {
		conn, err := local.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		c.NewLocalConn(conn, conf)
	}
}

func (c *Wspc) NewLocalConn(conn net.Conn, conf *msg.WspConfig) {
	channel := conf.Channel()
	log.Println("open remote channel", channel)
	id := ksuid.New().String()
	l := &localLinker{channel: channel, conn: conn}
	c.channel.NewTcpSession(id, conf, l, conn).Syn()
}

type localLinker struct {
	channel string
	conn    net.Conn
}

func (l *localLinker) InActive(err error) {
	log.Println("close remote channel", l.channel, err.Error())
}
func (l *localLinker) Active(session *channel.Session) error {
	go func() {
		session.CopyFrom(l.conn)
		log.Println("close remote channel", l.channel)
	}()
	return nil
}
