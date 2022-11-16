package client

import (
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/gowsp/wsp/pkg/msg"
)

func (c *Wspc) NewConn(data *msg.Data, req *msg.WspRequest) error {
	remote, err := req.ToConfig()
	if err != nil {
		return err
	}
	channel := remote.Channel()
	log.Println("received conn", channel)
	defer log.Println("close conn", channel)
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
	local, err := c.wan.Accept(data.ID(), conn, conf)
	if err != nil {
		conn.Close()
		return err
	}
	io.Copy(local, conn)
	local.Close()
	return nil
}
