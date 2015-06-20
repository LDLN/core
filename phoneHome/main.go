package main

import (
	"log"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"net/http"
	"net/url"
	"net"
	"github.com/gorilla/websocket"
	"time"
)

func main() {
	
	log.Printf("LDLN phone home...")

	// connect to mongodb
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()
	
	// connect to socket
	u, err := url.Parse("http://localhost:8080/ws")
	if err != nil {
	    log.Fatal(err)
	}
	
	// establish web socket connection to server
	rawConn, err := net.Dial("tcp", u.Host)
	if err != nil {
	    log.Fatal(err)
	}
	wsHeaders := http.Header{
	    "Origin":                   {"http://localhost:8080"},
	    // your milage may differ
	    "Sec-WebSocket-Extensions": {"permessage-deflate; client_max_window_bits, x-webkit-deflate-frame"},
	}
	wsConn, resp, err := websocket.NewClient(rawConn, u, wsHeaders, 1024, 1024)
	if err != nil {
	    log.Fatal("websocket.NewClient Error: %s\nResp:%+v", err, resp)
	}
	
	// periodically send diff request
	for {
				
		// init client_diff_request
		response_map := make(map[string]interface{})
		response_map["action"] = "client_diff_request"
		
		// find syncable object uuids
		cb := session.DB("landline").C("SyncableObjects")
		var m_result []map[string]interface{}
		err = cb.Find(bson.M{}).All(&m_result)
		if err != nil {
			log.Println(err)
		} else {
			
			// objects that the clients knows, but the server may not
			object_uuids := make(map[string]interface{})

			// loop results
			for u, result := range m_result {
				log.Println(u)
				log.Println(result)
				object_uuids[result["uuid"].(string)] = result["time_modified_since_creation"]
			}
			response_map["object_uuids"] = object_uuids
		
			// send it over websocket
			wsConn.WriteJSON(response_map)
		}
		
		// rest i need
		time.Sleep(5000 * time.Millisecond)
	}
	
}

