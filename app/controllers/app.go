package controllers

import (
	"code.google.com/p/go.net/websocket"
	"github.com/revel/revel"
	"github.com/ldln/landline-basestation/app/chatroom"
	"strings"
	"encoding/json"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
/* 	"io" */
)

type App struct {
	*revel.Controller
}

type Message struct {
	Action, Objects string
}

type SyncableObject struct {
	uuid string
	key_value_pairs string
	time_modified_since_creation float64
}

type UserObject struct {
	username string
	hashed_password string
	encrypted_kek string
}

func (c App) Chatroom() revel.Result {
	return c.Render()
}

func parseObjectUUIDs(m map[string]interface{}) {
				
	for k, v := range m {
		revel.TRACE.Println(k)
		revel.TRACE.Println(v)
	}
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
			
			// unmarshal the json
			byt := []byte((msg))
			var dat map[string]interface{}
			if err := json.Unmarshal(byt, &dat); err != nil {
        		panic(err)
    		}
    		revel.TRACE.Println(dat)
			
			// init response
			response_map := make(map[string]interface{})
						
			// connect to mongodb
			session, err := mgo.Dial("localhost")
	        if err != nil {
				panic(err)
	        }
	        defer session.Close()
			
			// get action string
			action := dat["action"].(string)
    		revel.TRACE.Println(action)
    		switch(action) {
    		
    			case "client_get_users":
					
					// prep response
					response_map["action"] = "server_send_users"
					var users_array []interface{}
				
					// query
					c := session.DB("landline").C("Users")
					var results []map[string]interface{}
					err = c.Find(bson.M{}).All(&results)
					if err != nil {
							
						revel.TRACE.Println(err)
						
			        } else {
						revel.TRACE.Println(results)
						
						for u, result := range results {
							revel.TRACE.Println(u)
							revel.TRACE.Println(result)
							revel.TRACE.Println(result["username"].(string))
							
							user_object_map := make(map[string]string)
							user_object_map["username"] = result["username"].(string)
							user_object_map["hashed_password"] = result["hashed_password"].(string)
							user_object_map["encrypted_kek"] = result["encrypted_kek"].(string)
							users_array = append(users_array, user_object_map)
						}
					}
					
					response_map["users"] = users_array
    		
    			case "client_get_schemas":
					
					// prep response
					response_map["action"] = "server_send_schemas"
					var schemas_array []interface{}
				
					// query
					c := session.DB("landline").C("Schemas")
					var results []map[string]interface{}
					err = c.Find(bson.M{}).All(&results)
					if err != nil {
							
						revel.TRACE.Println(err)
						
			        } else {
						revel.TRACE.Println(results)
						
						for u, result := range results {
							revel.TRACE.Println(u)
							
							schema_object_map := make(map[string]interface{})
							schema_object_map["object_key"] = result["object_key"].(string)
							schema_object_map["object_label"] = result["object_label"].(string)
							schema_object_map["weight"] = result["weight"].(float64)
							
							var fields_array []map[string]interface{}
							for f, field := range result["schema"].([]interface{}) {
								revel.TRACE.Println(f)
								revel.TRACE.Println(field)
								
								field_object_map := make(map[string]interface{})
								field_object_map["label"] = field.(map[string]interface{})["label"].(string)
								field_object_map["type"] = field.(map[string]interface{})["type"].(string)
								field_object_map["weight"] = field.(map[string]interface{})["weight"].(float64)
								fields_array = append(fields_array, field_object_map)
							}
							schema_object_map["schema"] = fields_array
							
							schemas_array = append(schemas_array, schema_object_map)
						}
					}
					
					response_map["schemas"] = schemas_array
    				
    			case "client_diff_request":
					
					revel.TRACE.Println(dat["object_uuids"])
					
					// prep response
					response_map["action"] = "server_diff_response"
					var server_unknown_object_uuids []interface{}
					var modified_objects []interface{}
					var client_unknown_objects []interface{}
					
					// loop through object_uuids
 					var object_uuid_client_knowns []interface{}
					for k, v := range dat["object_uuids"].(map[string]interface{}) {
						
						revel.TRACE.Println(k)
						revel.TRACE.Println(v)
						
						// build list that which client does not have - for later use
 						object_uuid_client_knowns = append(object_uuid_client_knowns, k)
						
						// find from mongodb
						c := session.DB("landline").C("SyncableObjects")
						var result map[string]interface{}
						err = c.Find(bson.M{"uuid": k}).One(&result)
				        if err != nil {
							
							// add unknowns to unknowns array
							revel.TRACE.Println(err)
							server_unknown_object_uuids = append(server_unknown_object_uuids, k)
							
				        } else {
							
							// debug results
							revel.TRACE.Println(result)
							revel.TRACE.Println(result["uuid"])
							revel.TRACE.Println(result["time_modified_since_creation"])
							
							// if there's a difference in time_modified_since_creation
							var smsc float64
							smsc = v.(float64)
							msc := result["time_modified_since_creation"].(float64)
							if smsc > msc {
								
								// client has updated more recently than server - put in unknown_object_uuids array
								server_unknown_object_uuids = append(server_unknown_object_uuids, k)
							
							} else if smsc < msc {
								
								// server has updated more recently than client - put in modified_objects array
								syncable_object_map := make(map[string]interface{})
								syncable_object_map["uuid"] = result["uuid"]
								syncable_object_map["key_value_pairs"] = result["key_value_pairs"]
								syncable_object_map["time_modified_since_creation"] = result["time_modified_since_creation"]
								
								modified_objects = append(modified_objects, syncable_object_map)
							}
						}
					}
					revel.TRACE.Println(object_uuid_client_knowns)
						
					// find things that client does not have
					cb := session.DB("landline").C("SyncableObjects")
					var m_result []map[string]interface{}
					err = cb.Find(bson.M{"uuid" : bson.M{ "$not" : bson.M{ "$in" : object_uuid_client_knowns} } } ).All(&m_result)
					if err != nil {
						
						revel.TRACE.Println(err)
						
			        } else {
						
						for u, result := range m_result {
							revel.TRACE.Println(u)
							revel.TRACE.Println(result)
							
							// object that the client does not know about
							syncable_object_map := make(map[string]interface{})
							syncable_object_map["uuid"] = result["uuid"]
							syncable_object_map["key_value_pairs"] = result["key_value_pairs"]
							syncable_object_map["time_modified_since_creation"] = result["time_modified_since_creation"]
							
							client_unknown_objects = append(client_unknown_objects, syncable_object_map)
						}
						
					}
						
						
					response_map["server_unknown_object_uuids"] = server_unknown_object_uuids
					response_map["modified_objects"] = modified_objects
					response_map["client_unknown_objects"] = client_unknown_objects
    				
    			case "client_update_request":
    				s := []string{"client_update_request", action}
    				revel.TRACE.Println(strings.Join(s, " : "))
    				
    		}
			
			// send initial message
			newMessages <- msg
			
			// form json server response message 
			response_json_map, err := json.Marshal(response_map)
			if err != nil {
				revel.TRACE.Println(err)
			}
			revel.TRACE.Println(string(response_json_map[:]))
			revel.TRACE.Println(response_json_map)
			
			// send server response message
			newMessages <- string(response_json_map[:])
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
			chatroom.Say(user, msg)
		}
	}
	return nil
}

