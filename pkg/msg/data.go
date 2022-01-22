//Package msg is Data transferred before client and server
package msg

type Data struct {
	Msg *WspMessage
	Raw *[]byte
}

func (data *Data) ID() string {
	return data.Msg.Id
}
func (data *Data) Cmd() WspCmd {
	return data.Msg.Cmd
}
func (data *Data) Payload() []byte {
	return data.Msg.Data
}
