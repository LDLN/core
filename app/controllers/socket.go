package controllers

import (
	"code.google.com/p/go.net/websocket"
	"github.com/revel/revel"
)

type Socket struct {
	*revel.Controller
}

func (c Socket) RoomSocket(room string, ws *websocket.Conn) revel.Result {
	
	go func() {
		var msg string
		for {
			
			err := websocket.Message.Receive(ws, &msg)
			if err != nil {
				return
			}
			
			revel.TRACE.Printf(msg)
			
			websocket.Message.Send(ws, "response")
		}
		
	}()
	
	return nil
}