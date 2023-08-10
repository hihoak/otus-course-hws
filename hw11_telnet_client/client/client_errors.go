package client

type ErrorEstablishConnection struct{}

func (ErrorEstablishConnection) Error() string {
	return "failed to establish connection"
}
