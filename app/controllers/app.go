package controllers

import (
	"code.google.com/p/go.net/websocket"
	"github.com/revel/revel"
	"basestation/app/chatroom"
	"strings"
	"encoding/json"
/* 	"io" */
)

type App struct {
	*revel.Controller
}

type Message struct {
	Action, Objects string
}

func (c App) Index() revel.Result {
	return c.Render()
}

func (c App) RoomSocket(user string, ws *websocket.Conn) revel.Result {
	// Join the room.
	subscription := chatroom.Subscribe()
	defer subscription.Cancel()

	chatroom.Join(user)
	defer chatroom.Leave(user)

	// Send down the archive.
	for _, event := range subscription.Archive {
		if websocket.JSON.Send(ws, &event) != nil {
			// They disconnected
			return nil
		}
	}

	// In order to select between websocket messages and subscription events, we
	// need to stuff websocket events into a channel.
	newMessages := make(chan string)
	go func() {
		var msg string
		for {
			err := websocket.Message.Receive(ws, &msg)
			if err != nil {
				close(newMessages)
				revel.TRACE.Printf("err != nil - newMessages: %s", newMessages)
				return
			}
			
			
			
			revel.TRACE.Printf(msg)
/*
			dec := json.NewDecoder(strings.NewReader(msg))
			
			var m Message
			if err := dec.Decode(&m); err == io.EOF {
				break
			} else if err != nil {
				revel.ERROR.Printf("%s", err)
			}
			revel.TRACE.Printf("action: %s", m.Action)
			revel.TRACE.Printf("object_uuids: %s", m.Objects)
*/
			

			byt := []byte((msg))
			var dat map[string]interface{}
			if err := json.Unmarshal(byt, &dat); err != nil {
        		panic(err)
    		}
    		revel.TRACE.Println(dat)
			
			
			action := dat["action"].(string)
    		revel.TRACE.Println(action)
    		
    		switch(action) {
    		
    			case "client_get_users":
    				s := []string{"client_get_users", action}
    				revel.TRACE.Println(strings.Join(s, " : "))
    		
    			case "client_get_schema":
    				s := []string{"client_get_schema", action}
    				revel.TRACE.Println(strings.Join(s, " : "))
    				
    			case "client_diff_request":
    				s := []string{"client_diff_request", action}
    				revel.TRACE.Println(strings.Join(s, " : "))
    				
    			case "client_update_request":
    				s := []string{"client_update_request", action}
    				revel.TRACE.Println(strings.Join(s, " : "))
    				
    		}
			
			newMessages <- msg
		}
	}()

	// Now listen for new events from either the websocket or the chatroom.
	for {
		select {
		case event := <-subscription.New:
			if websocket.JSON.Send(ws, &event) != nil {
				// They disconnected.
				return nil
			}
		case msg, ok := <-newMessages:
			// If the channel is closed, they disconnected.
			if !ok {
				return nil
			}

			// Otherwise, say something.
			s := []string{"this", msg};
			chatroom.Say(user, strings.Join(s, " - "))
		}
	}
	return nil
}

