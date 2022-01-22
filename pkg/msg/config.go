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
func (c *WspConfig) IsHTTP() bool {
	switch c.url.Scheme {
	case "http", "https":
		return true
	}
	return false
}
func (c *WspConfig) IsTunnel() bool {
	return c.url.Scheme == "tunnel"
}
func (c *WspConfig) Channel() string {
	if c.IsHTTP() {
		return "http:" + c.Mode() + ":" + c.Value()
	}
	mode := c.Mode()
	if mode == "" {
		return c.url.User.Username()
	}
	return "http:" + c.Mode() + ":" + c.Value()
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
func (c *WspConfig) DynamicAddr(addr string) *WspConfig {
	wspType := c.wspType
	if c.url.User != nil {
		wspType = WspType_LOCAL
	}
	return &WspConfig{
		wspType: wspType,
		url: &url.URL{
			Scheme:   c.url.Scheme,
			Host:     addr,
			User:     c.url.User,
			RawQuery: c.url.RawQuery,
		},
	}
}

func (c *WspConfig) ReverseURL() *url.URL {
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
