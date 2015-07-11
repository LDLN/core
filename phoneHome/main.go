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
	"encoding/json"
	"github.com/hashicorp/mdns"
	"fmt"
	"strings"
	"strconv"
)

func main() {
    
	// Make a channel for results and start listening
	entriesCh := make(chan *mdns.ServiceEntry, 4)
	go func() {
	    for entry := range entriesCh {
	        fmt.Printf("Gots new entry: %v\n", entry.Name)
			
			if strings.Contains(entry.Name, "LDLN\\ Basestation") {
				
				clientConnect(entry.Host, entry.Port)
			}
	    }
	}()
	
	// Start the lookup
	for {
		mdns.Lookup("_http._tcp", entriesCh)
		time.Sleep(60000 * time.Millisecond)
	}
	defer close(entriesCh)
}

func clientConnect(host string, port int) {
	
	log.Printf("LDLN phone home...")

	// connect to mongodb
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()
	
	// connect to socket
	s := []string{"http://", host, ":", strconv.Itoa(port), "/ws"};
    wsurl := strings.Join(s, "")
	
	u, err := url.Parse(wsurl)
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
	wsConn, resp, err := websocket.NewClient(rawConn, u, wsHeaders, 2048, 2048)
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
				
		clientDiffRequest(wsConn, session)
		
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
					log.Println("RECEIVED server_diff_response")
					processServerDiffResponse(wsConn, m, session)
					
				case "server_send_users":
					log.Println("RECEIVED server_send_users")
					processServerSendUsers(m, session)
					
				case "server_send_schemas":
					log.Println("RECEIVED server_send_schemas")
			}
		}
	}
	
}

func clientGetRequest(wsConn *websocket.Conn, action string) {
	
	messageMap := make(map[string]interface{})
	messageMap["action"] = action
	
	wsConn.WriteJSON(messageMap)
}

func clientDiffRequest(wsConn *websocket.Conn, session *mgo.Session) {
		
	// init client_diff_request
	response_map := make(map[string]interface{})
	response_map["action"] = "client_diff_request"
	
	// find syncable object uuids
	cb := session.DB("landline").C("SyncableObjects")
	var m_result []map[string]interface{}
	err := cb.Find(bson.M{}).All(&m_result)
	if err != nil {
		log.Println(err)
	} else {
		
		// objects that the clients knows, but the server may not
		object_uuids := make(map[string]interface{})

		// loop results
		for u, result := range m_result {
			_ = u
			object_uuids[result["uuid"].(string)] = result["time_modified_since_creation"]
		}
		response_map["object_uuids"] = object_uuids
	
		// send it over websocket
		wsConn.WriteJSON(response_map)
		
		log.Println("Wrote message:")
		jsonString, _ := json.Marshal(response_map)
		log.Println(string(jsonString))
	}
}

func processServerSendUsers(m map[string]interface{}, session *mgo.Session) {

	log.Println("PARSING server_send_users")

	// wholesale replace users collection
	log.Println("wholesale replace users collection")
	c := session.DB("landline").C("Users")
	c.RemoveAll(bson.M{})
		
	for k, v := range m["users"].([]interface{}) {
		
		log.Println(k)
		object := v.(map[string]interface{})
		
		// create object
		object_map := make(map[string]interface{})
		object_map["encrypted_kek"] = object["encrypted_kek"].(string)
		object_map["encrypted_rsa_private"] = object["encrypted_rsa_private"].(string)
		object_map["hashed_password"] = object["hashed_password"].(string)
		object_map["rsa_public"] = object["rsa_public"].(string)
		object_map["username"] = object["username"].(string)
		
		err := c.Insert(object_map)
		if err != nil {
			panic(err)
		}
	}
}

func processServerDiffResponse(wsConn *websocket.Conn, m map[string]interface{}, session *mgo.Session) {
				
	// parse client_unknown_objects and save them
	log.Println("PARSING client_unknown_objects")
	if m["client_unknown_objects"] != nil {
		for k, v := range m["client_unknown_objects"].([]interface{}) {
			
			log.Println("A client_unknown_object:")
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
		
		log.Println(m["server_unknown_object_uuids"])
		
		// create response back
		messageMap := make(map[string]interface{})
		messageMap["action"] = "client_update_request"
		
		// find object the server does not have
		var serverNeeds []map[string]interface{}
		c := session.DB("landline").C("SyncableObjects")
		
		log.Println("all server needs:")
		log.Println(m["server_unknown_object_uuids"])
			
		err := c.Find(bson.M{"uuid": bson.M{"$in":  m["server_unknown_object_uuids"]}}).All(&serverNeeds)
		if err != nil {
			panic(err)
		}
			
		// send back client_update_request with objects array
		var objects []interface{}
		for _, v := range serverNeeds {
			object := make(map[string]interface{})
			object["uuid"] = v["uuid"]
			object["key_value_pairs"] = v["key_value_pairs"]
			object["object_type"] = v["object_type"]
			object["time_modified_since_creation"] = v["time_modified_since_creation"]
			objects = append(objects, object)
		}
		messageMap["objects"] = objects
		
		// write back
		wsConn.WriteJSON(messageMap)
		
		log.Println("Wrote message:")
		jsonString, _ := json.Marshal(messageMap)
		log.Println(string(jsonString))
	}
}