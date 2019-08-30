package managers

import "github.com/byuoitav/shipwright/socket"

//WebsocketForwarder .
type WebsocketForwarder struct {
}

//GetDefaultWebsocketForwarder .
func GetDefaultWebsocketForwarder() *WebsocketForwarder {
	return &WebsocketForwarder{}
}

//Send .
func (e *WebsocketForwarder) Send(toSend interface{}) error {
	socket.GetManager().WriteToSockets(toSend)
	return nil
}
