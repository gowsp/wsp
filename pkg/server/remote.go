package server

import (
	"fmt"
	"log"

	"github.com/gowsp/wsp/pkg/channel"
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
	if c.wsps.Exist(channel) {
		return fmt.Errorf("channel %s already registered", channel)
	}
	if err := c.channel.Reply(id, msg.WspCode_SUCCESS, ""); err != nil {
		return err
	}
	log.Println("register channel", channel)
	c.wsps.Store(channel, c)
	c.configs.Store(channel, conf)
	return nil
}

type bridge struct {
	id     string
	config *msg.WspConfig
	local  *channel.Channel
	output *channel.Channel
}

func (l *bridge) InActive(err error) {
	l.local.Reply(l.id, msg.WspCode_FAILED, err.Error())
}

func (l *bridge) Active(session *channel.Session) error {
	l.local.Bridge(l.id, l.config, l.output)
	return l.local.Reply(l.id, msg.WspCode_SUCCESS, "")
}

func (c *conn) NewRemoteConn(id string, conf *msg.WspConfig) error {
	key := conf.Channel()
	if val, ok := c.wsps.LoadRouter(key); ok {
		log.Printf("start bridge channel %s\n", key)
		remote := val.(*conn)

		config, ok := remote.configs.Load(key)
		if !ok {
			return fmt.Errorf("channel not registered")
		}
		w := c.channel.NewWsWriter(id)
		l := &bridge{id, config.(*msg.WspConfig), c.channel, remote.channel}
		remote.channel.NewSession(id, conf, l, w).Syn()
		return nil
	}
	return fmt.Errorf("channel not registered")
}
