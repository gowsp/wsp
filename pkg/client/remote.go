package client

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/gowsp/wsp/pkg/channel"
	"github.com/gowsp/wsp/pkg/msg"
	"github.com/segmentio/ksuid"
)

type remoteRegister struct {
	c      *Wspc
	config *msg.WspConfig
}

func (l *remoteRegister) InActive(err error) {
	log.Println("error:", err.Error())
}
func (l *remoteRegister) Active(session *channel.Session) error {
	log.Println("listen remote on channel", l.config.Channel())
	l.c.configs.Store(l.config.Channel(), l.config)
	return nil
}

func (c *Wspc) Register() {
	for _, val := range c.config.Remote {
		conf, err := msg.NewWspConfig(msg.WspType_REMOTE, val)
		if err != nil {
			log.Println("forward remote error", err)
			continue
		}
		id := ksuid.New().String()
		c.channel.NewSession(id, conf, &remoteRegister{c, conf}, nil).Syn()
	}
}

type remoteLinker struct {
	conn    net.Conn
	channel string
}

func (l *remoteLinker) InActive(err error) {
	log.Println("open remote", l.channel, err.Error())
}
func (l *remoteLinker) Active(session *channel.Session) error {
	go func() {
		session.CopyFrom(l.conn)
		log.Printf("close %s connection\n", l.channel)
	}()
	return nil
}

func (c *Wspc) NewConn(id string, req *msg.WspRequest) error {
	remote, err := req.ToConfig()
	if err != nil {
		return err
	}
	channel := remote.Channel()
	log.Printf("received %s connection\n", channel)
	conf, err := c.LoadConfig(channel)
	if err != nil {
		return err
	}
	if conf.Paasowrd() != remote.Paasowrd() {
		return fmt.Errorf("%s password is incorrect", channel)
	}
	var conn net.Conn
	if conf.IsTunnel() {
		conn, err = net.DialTimeout(remote.Network(), remote.Address(), 5*time.Second)
	} else {
		conn, err = net.DialTimeout(conf.Network(), conf.Address(), 5*time.Second)
	}
	if err != nil {
		return err
	}
	l := &remoteLinker{conn: conn, channel: channel}
	return c.channel.NewTcpSession(id, remote, l, conn).Ack()
}
