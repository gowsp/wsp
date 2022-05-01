package channel

type Linker interface {
	InActive()

	Active(session *Session) error
}

type Handler struct {
	linker Linker
	writer Writer
}
