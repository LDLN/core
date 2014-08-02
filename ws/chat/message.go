package chat

import (
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
)

type Message struct {
	body   map[string]interface{}
}

func (self *Message) String() map[string]interface{} {
	return self.body
}

func (msg_obj *Message) parse(c *Client) {

	// unmarshal the json
	//byt := []byte((msg_obj.Body))
	//var dat map[string]interface{}
	//if err := json.Unmarshal(byt, &dat); err != nil {
	//	panic(err)
	//}
	dat := msg_obj.body

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
	log.Println(action)
	switch action {

	case "client_get_users":

		// prep response
		response_map["action"] = "server_send_users"
		var users_array []interface{}

		// query
		c := session.DB("landline").C("Users")
		var results []map[string]interface{}
		err = c.Find(bson.M{}).All(&results)
		if err != nil {

			log.Println(err)

		} else {
			log.Println(results)

			for u, result := range results {
				log.Println(u)
				log.Println(result)
				log.Println(result["username"].(string))

				user_object_map := make(map[string]string)
				user_object_map["username"] = result["username"].(string)
				user_object_map["hashed_password"] = result["hashed_password"].(string)
				user_object_map["encrypted_kek"] = result["encrypted_kek"].(string)
				user_object_map["encrypted_rsa_private"] = result["encrypted_rsa_private"].(string)
				user_object_map["rsa_public"] = result["rsa_public"].(string)
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

			log.Println(err)

		} else {
			log.Println(results)

			for u, result := range results {
				log.Println(u)

				schema_object_map := make(map[string]interface{})
				schema_object_map["object_key"] = result["object_key"].(string)
				schema_object_map["object_label"] = result["object_label"].(string)
				schema_object_map["weight"] = result["weight"].(float64)

				var fields_array []map[string]interface{}
				for f, field := range result["schema"].([]interface{}) {
					log.Println(f)
					log.Println(field)

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

		log.Println(dat["object_uuids"])

		// prep response
		response_map["action"] = "server_diff_response"
		var server_unknown_object_uuids []interface{}
		var modified_objects []interface{}
		var client_unknown_objects []interface{}

		// loop through object_uuids
		var object_uuid_client_knowns []interface{}
		for k, v := range dat["object_uuids"].(map[string]interface{}) {

			log.Println(k)
			log.Println(v)

			// build list that which client does not have - for later use
			object_uuid_client_knowns = append(object_uuid_client_knowns, k)

			// find from mongodb
			c := session.DB("landline").C("SyncableObjects")
			var result map[string]interface{}
			err = c.Find(bson.M{"uuid": k}).One(&result)
			if err != nil {

				// add unknowns to unknowns array
				log.Println(err)
				server_unknown_object_uuids = append(server_unknown_object_uuids, k)

			} else {

				// debug results
				log.Println(result)
				log.Println(result["uuid"])
				log.Println(result["time_modified_since_creation"])

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
					syncable_object_map["object_type"] = result["object_type"]
					syncable_object_map["key_value_pairs"] = result["key_value_pairs"]
					syncable_object_map["time_modified_since_creation"] = result["time_modified_since_creation"]

					modified_objects = append(modified_objects, syncable_object_map)
				}
			}
		}
		log.Println(object_uuid_client_knowns)

		// find things that client does not have
		cb := session.DB("landline").C("SyncableObjects")
		var m_result []map[string]interface{}
		err = cb.Find(bson.M{"uuid": bson.M{"$not": bson.M{"$in": object_uuid_client_knowns}}}).All(&m_result)
		if err != nil {

			log.Println(err)

		} else {

			for u, result := range m_result {
				log.Println(u)
				log.Println(result)

				// object that the client does not know about
				syncable_object_map := make(map[string]interface{})
				syncable_object_map["uuid"] = result["uuid"]
				syncable_object_map["object_type"] = result["object_type"]
				syncable_object_map["key_value_pairs"] = result["key_value_pairs"]
				syncable_object_map["time_modified_since_creation"] = result["time_modified_since_creation"]

				client_unknown_objects = append(client_unknown_objects, syncable_object_map)
			}

		}

		response_map["server_unknown_object_uuids"] = server_unknown_object_uuids
		response_map["modified_objects"] = modified_objects
		response_map["client_unknown_objects"] = client_unknown_objects

	case "client_update_request":

		// prep response
		response_map["action"] = "server_update_response"
		var created_object_uuids []interface{}
		var updated_objects []interface{}
		
		// parse objects
		for k, v := range dat["objects"].([]interface{}) {
			
			log.Println(k)
			log.Println(v.(map[string]interface{})["uuid"].(string))
			
			object := v.(map[string]interface{})
			
			// find from mongodb
			c := session.DB("landline").C("SyncableObjects")
			var result map[string]interface{}
			err = c.Find(bson.M{"uuid": object["uuid"].(string)}).One(&result)
			if err != nil {

				// create object
				object_map := make(map[string]interface{})
				object_map["uuid"] = object["uuid"].(string)
				object_map["object_type"] = object["object_type"].(string)
				object_map["key_value_pairs"] = object["key_value_pairs"].(string)
				object_map["time_modified_since_creation"] = object["time_modified_since_creation"].(float64)
			
				err = c.Insert(object_map)
				if err != nil {
					panic(err)
				}
				
				// add to response
				created_object_uuids = append(created_object_uuids, object["uuid"])

			} else {
			
				// who has more recent object
				if(result["time_modified_since_creation"].(float64) >= object["time_modified_since_creation"].(float64)) {
					
					// output more recent object if more recent
					syncable_object_map := make(map[string]interface{})
					syncable_object_map["uuid"] = result["uuid"]
					syncable_object_map["object_type"] = result["object_type"]
					syncable_object_map["key_value_pairs"] = result["key_value_pairs"]
					syncable_object_map["time_modified_since_creation"] = result["time_modified_since_creation"]
	
					updated_objects = append(updated_objects, syncable_object_map)
					
				} else {
					
					// update object
					object_map := make(map[string]interface{})
					object_map["uuid"] = object["uuid"].(string)
					object_map["object_type"] = object["object_type"].(string)
					object_map["key_value_pairs"] = object["key_value_pairs"].(string)
					object_map["time_modified_since_creation"] = object["time_modified_since_creation"].(float64)
				
					err = c.Update(bson.M{"uuid": result["uuid"]}, object_map)
					if err != nil {
						panic(err)
					}
				
					// add to response
					created_object_uuids = append(created_object_uuids, result["uuid"])
				}
			
			}
		}

		response_map["created_object_uuids"] = created_object_uuids
		response_map["updated_objects"] = updated_objects
	}

	c.Write(&Message{response_map})

	// send server response message
	//return string(response_json_map[:])
}
