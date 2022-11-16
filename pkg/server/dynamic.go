package server

import (
	"io"
	"log"
	"net"
	"time"

	"github.com/gowsp/wsp/pkg/msg"
)

func (c *conn) NewDynamic(id string, conf *msg.WspConfig) error {
	address := conf.Address()
	log.Println("open proxy", address)

	remote, err := net.DialTimeout(conf.Network(), address, 5*time.Second)
	if err != nil {
		return err
	}
	local, err := c.wan.Accept(id, remote, conf)
	if err != nil {
		remote.Close()
		return err
	}
	io.Copy(local, remote)
	local.Close()
	log.Println("close proxy", address)
	return nil
}
