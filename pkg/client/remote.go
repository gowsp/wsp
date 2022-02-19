package client

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/gowsp/wsp/pkg/msg"
	"github.com/gowsp/wsp/pkg/proxy"
	"github.com/segmentio/ksuid"
)

func (c *Wspc) RemoteForward() {
	for _, val := range c.Config.Remote {
		conf, err := msg.NewWspConfig(msg.WspType_REMOTE, val)
		if err != nil {
			log.Println("forward remote error,", err)
			continue
		}
		go c.ListenRemote(conf)
	}
}
func (c *Wspc) ListenRemote(conf *msg.WspConfig) {
	log.Println("listen remote on channel", conf.Channel())
	id := ksuid.New().String()
	c.routing.AddPending(id, func(data *msg.Data, message *msg.WspResponse) {
		if message.Code == msg.WspCode_FAILED {
			log.Println("err", message.Data)
			return
		}
		c.channel.Store(conf.Channel(), conf)
	})
	if err := c.wan.Dail(id, conf); err != nil {
		log.Println(err)
		return
	}
}

func (c *Wspc) NewRemoteConn(id string, remote *msg.WspConfig) error {
	channel := remote.Channel()
	log.Printf("received %s connection\n", channel)
	defer log.Printf("close %s connection\n", channel)
	val, ok := c.channel.Load(channel)
	if !ok {
		return fmt.Errorf(channel + " not found")
	}

	conf := val.(*msg.WspConfig)
	if conf.Paasowrd() != remote.Paasowrd() {
		return fmt.Errorf("%s password is incorrect", channel)
	}
	var err error
	var conn net.Conn
	if conf.IsTunnel() {
		conn, err = net.DialTimeout(remote.Network(), remote.Address(), 5*time.Second)
	} else {
		conn, err = net.DialTimeout(conf.Network(), conf.Address(), 5*time.Second)
	}
	if err != nil {
		return err
	}

	in := c.wan.NewWriter(id)
	repeater := proxy.NewNetRepeater(in, conn)
	c.routing.AddRepeater(id, repeater)
	defer c.routing.Delete(id)

	c.wan.Reply(id, true)

	repeater.Copy()
	return nil
}
