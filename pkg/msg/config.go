package msg

import (
	"net/url"
)

func (r *WspRequest) ToConfig() (*WspConfig, error) {
	u, err := url.Parse(r.Data)
	if err != nil {
		return nil, err
	}
	return &WspConfig{wspType: r.Type, url: u}, nil
}

func NewWspConfig(wspType WspType, addr string) (*WspConfig, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	return &WspConfig{wspType: wspType, url: u}, nil
}

type WspConfig struct {
	wspType WspType
	url     *url.URL
}

func (c *WspConfig) ToReqeust() *WspRequest {
	return &WspRequest{Type: c.wspType, Data: c.url.String()}
}
func (c *WspConfig) IsHttp() bool {
	switch c.url.Scheme {
	case "http", "https":
		return true
	}
	return false
}
func (c *WspConfig) Channel() string {
	if c.IsHttp() {
		return "http:" + c.Mode() + ":" + c.Value()
	}
	return c.url.User.Username()
}
func (c *WspConfig) Scheme() string {
	return c.url.Scheme
}
func (c *WspConfig) Network() string {
	network := c.url.Scheme
	switch network {
	case "http", "https", "socks5":
		return "tcp"
	default:
		return network
	}
}

func (c *WspConfig) Address() string {
	return c.url.Host
}
func (c *WspConfig) Paasowrd() string {
	pwd, _ := c.url.User.Password()
	return pwd
}
func (c *WspConfig) ReverseUrl() *url.URL {
	return &url.URL{
		Scheme: c.url.Scheme,
		Host:   c.url.Host,
		Path:   c.url.Path,
	}
}
func (c *WspConfig) Mode() string {
	return c.url.Query().Get("mode")
}
func (c *WspConfig) Value() string {
	return c.url.Query().Get("value")
}
