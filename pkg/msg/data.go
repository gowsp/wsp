package msg

type Data struct {
	Msg *WspMessage
	Raw *[]byte
}

func (data *Data) Id() string {
	return data.Msg.Id
}
func (data *Data) Cmd() WspCmd {
	return data.Msg.Cmd
}
func (data *Data) Payload() []byte {
	return data.Msg.Data
}
