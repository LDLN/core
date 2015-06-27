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
	u, err := url.Parse("http://192.168.0.1:8080/ws")
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
	
	// start the upstream reader
	go reader(wsConn)
	
	// check if local deployment is empty
	
		// sync deployment from upstream
		
	// else check if upstream has same deployment
	
	// else

		// fail
	
	// sync users
	clientGetRequest(wsConn, "client_get_users")
	
	// sync schemas
	clientGetRequest(wsConn, "client_get_schemas")
	
	// periodically send diff request upstream
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
			
			log.Println("Wrote message:")
			log.Println(response_map)
		}
		
		// rest i need
		time.Sleep(5000 * time.Millisecond)
	}
}

func reader(wsConn *websocket.Conn) {
	
	// connect to mongodb
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()
	
	for {
		m := make(map[string]interface{})
 
		err := wsConn.ReadJSON(&m)
		if err != nil {
			log.Println("Error reading json.", err)
		}
 
		log.Println("Got message:")
		log.Println(m)
		
		// if its server_diff_response
		if m["action"] != nil {
			
			action := m["action"].(string)
			switch action {
				case "server_diff_response":
				
					processServerDiffResponse(m, session)
					
				case "server_send_users"	:
				
				
				case "server_send_schemas"	:
				
			}
		}
	}
	
}

func clientGetRequest(wsConn *websocket.Conn, action string) {
	
	messageMap := make(map[string]interface{})
	messageMap["action"] = action
	
	wsConn.WriteJSON(messageMap)
}
func processServerDiffResponse(m map[string]interface{}, session *mgo.Session) {
				
	// parse client_unknown_objects and save them
	log.Println("PARSING client_unknown_objects")
	if m["client_unknown_objects"] != nil {
		for k, v := range m["client_unknown_objects"].([]interface{}) {
			
			log.Println(k)
			object := v.(map[string]interface{})
			
			// create object
			object_map := make(map[string]interface{})
			object_map["uuid"] = object["uuid"].(string)
			object_map["object_type"] = object["object_type"].(string)
			object_map["key_value_pairs"] = object["key_value_pairs"].(string)
			object_map["time_modified_since_creation"] = object["time_modified_since_creation"].(float64)
			
			// insert into database
			c := session.DB("landline").C("SyncableObjects")
			err := c.Insert(object_map)
			if err != nil {
				panic(err)
			}
		}
	} else {
		log.Println("client_unknown_objects is empty")
	}
	
	// for each server_unknown_object_uuids
	log.Println("PARSING server_unknown_object_uuids")
	if m["server_unknown_object_uuids"] != nil {
		
		// send back client_update_request with objects array
		for k, v := range m["server_unknown_object_uuids"].([]interface {}) {
			
			log.Println(k)
			log.Println(v)
			
			// create response back
		}
	}
}