package server

import (
	"fmt"
	"log"

	"github.com/gowsp/wsp/pkg/msg"
)

func (c *conn) AddRemote(id string, conf *msg.WspConfig) error {
	channel := conf.Channel()
	if channel == "" {
		return fmt.Errorf("channel %s is illegal or empty", channel)
	}
	switch conf.Mode() {
	case "path":
		if conf.Value() == c.config().Path {
			return fmt.Errorf("setting the same path as wsps is not allowed")
		}
	case "domain":
		switch {
		case c.config().Host == "":
			return fmt.Errorf("wsps does not set host, domain method is not allowed")
		case c.config().Host == conf.Value():
			return fmt.Errorf("setting the same domain as wsps is not allowed")
		}
	default:
		if conf.IsHTTP() {
			return fmt.Errorf("unsupported http mode %s", conf.Mode())
		}
	}
	if hub.Exist(channel) {
		return fmt.Errorf("channel %s already registered", channel)
	}
	if err := c.wan.Reply(id, nil); err != nil {
		return err
	}
	log.Println("register channel", channel)
	hub.Store(channel, c)
	c.listen.Store(channel, conf)
	return nil
}

func (c *conn) NewRemoteConn(req *msg.Data, config *msg.WspConfig) error {
	channel := config.Channel()
	val, ok := hub.Load(channel)
	if !ok {
		return fmt.Errorf("channel %s not registered", channel)
	}
	log.Println(channel, "bridging request received")
	remote := val.(*conn)
	if _, ok := remote.listen.Load(channel); !ok {
		hub.Remove(channel)
		return fmt.Errorf("channel %s offline", channel)
	}
	return c.wan.Bridge(req, config, remote.wan)
}
