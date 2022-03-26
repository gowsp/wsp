package msg

import "strconv"

const PROTOCOL_VERSION Protocol = 0.10

type Protocol float64

func ParseVersion(version string) (Protocol, error) {
	proto, err := strconv.ParseFloat(version, 64)
	return Protocol(proto), err
}
func (p Protocol) String() string {
	return strconv.FormatFloat(float64(p), 'f', 2, 64)
}
func (p Protocol) Major() int {
	return int(p)
}
