package client

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"github.com/gowsp/wsp/pkg/logger"
	"github.com/gowsp/wsp/pkg/msg"
)

var errVersion = fmt.Errorf("unsupported socks version")

// Socks5Proxy implement DynamicProxy
type Socks5Proxy struct {
	conf *msg.WspConfig
	wspc *Wspc
}

func (p *Socks5Proxy) Listen() {
	address := p.conf.Address()
	logger.Info("listen socks5 on %s", address)
	l, err := net.Listen(p.conf.Network(), address)
	if err != nil {
		logger.Error("listen socks5 error %s", err)
		return
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			logger.Error("accept socks5 error %s", err)
			continue
		}
		go func() {
			err := p.ServeConn(conn)
			if err == nil || err == io.EOF {
				return
			}
			if err != errVersion {
				logger.Error("serve socks5 error", err)
			}
		}()
	}
}

func (p *Socks5Proxy) ServeConn(conn net.Conn) error {
	if err := p.auth(conn); err != nil {
		conn.Close()
		return err
	}
	addr, err := p.readRequest(conn)
	if err != nil {
		conn.Close()
		return err
	}
	p.replies(addr, conn)
	return nil
}

func (p *Socks5Proxy) auth(conn net.Conn) error {
	// +----+----------+----------+
	// |VER | NMETHODS | METHODS  |
	// +----+----------+----------+
	// | 1  |    1     | 1 to 255 |
	// +----+----------+----------+
	info := make([]byte, 2)
	if _, err := io.ReadFull(conn, info); err != nil {
		return err
	}
	if info[0] != 0x05 {
		conn.Write([]byte{0x05, 0xFF})
		return errVersion
	}
	methods := make([]byte, info[1])
	if _, err := io.ReadFull(conn, methods); err != nil {
		return err
	}
	// +----+--------+
	// |VER | METHOD |
	// +----+--------+
	// | 1  |   1    |
	// +----+--------+
	_, err := conn.Write([]byte{0x05, 0x00})
	return err
}

func (p *Socks5Proxy) readRequest(conn net.Conn) (addr string, err error) {
	// +----+-----+-------+------+----------+----------+
	// |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
	// +----+-----+-------+------+----------+----------+
	// | 1  |  1  | X'00' |  1   | Variable |    2     |
	// +----+-----+-------+------+----------+----------+
	info := make([]byte, 4)
	if _, err := io.ReadFull(conn, info); err != nil {
		return "", err
	}
	if info[0] != 0x05 {
		return "", errVersion
	}
	var host string
	switch info[3] {
	case 1:
		host, err = p.readIP(conn, net.IPv4len)
		if err != nil {
			return "", err
		}
	case 4:
		host, err = p.readIP(conn, net.IPv6len)
		if err != nil {
			return "", err
		}
	case 3:
		if _, err := io.ReadFull(conn, info[3:]); err != nil {
			return "", err
		}
		hostName := make([]byte, info[3])
		if _, err := io.ReadFull(conn, hostName); err != nil {
			return "", err
		}
		host = string(hostName)
	default:
		return "", fmt.Errorf("unrecognized address type")
	}
	if _, err := io.ReadFull(conn, info[2:]); err != nil {
		return "", err
	}
	port := binary.BigEndian.Uint16(info[2:])
	return net.JoinHostPort(host, fmt.Sprintf("%d", port)), nil
}
func (p *Socks5Proxy) readIP(conn net.Conn, len byte) (string, error) {
	addr := make([]byte, len)
	if _, err := io.ReadFull(conn, addr); err != nil {
		return "", err
	}
	return net.IP(addr).String(), nil
}
func (p *Socks5Proxy) replies(addr string, local net.Conn) {
	config := p.conf.DynamicAddr(addr)
	remote, err := p.wspc.wan.DialTCP(local, config)
	if err != nil {
		local.Write([]byte{0x05, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		logger.Error("close socks5 %s %s", addr, err.Error())
		return
	}
	local.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	io.Copy(remote, local)
	remote.Close()
	logger.Info("close socks5 %s, addr %s", addr, local.LocalAddr())
}
