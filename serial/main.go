package main

import (
	"log"
	"github.com/tarm/serial"
	"labix.org/v2/mgo"
	"encoding/json"
)

func main() {

	// connect to mongodb
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()
	
	// connect to port
	c := &serial.Config{Name: "/dev/ttyS0", Baud: 38400}
	s, err := serial.OpenPort(c)
	if err != nil {
		log.Fatal(err)
	}
	
	// write message	
	n, err := s.Write([]byte("LDLN Serial is listening"))
	if err != nil {
		log.Fatal(err)
	}

	// read messages
	for {
        buf := make([]byte, 2048)
        n, err = s.Read(buf)
        if err != nil {
                log.Fatal(err)
        }
        log.Printf("Incoming on serial: %q", buf[:n])
		
		// convert string to JSON to map[string]interface{}
		v := make(map[string]interface{})
		err := json.Unmarshal(buf[:n], &v)
		if err != nil {
			log.Printf("Not a JSON object")
        } else {
			
			// create object
			object_map := make(map[string]interface{})
			object_map["uuid"] = v["uuid"].(string)
			object_map["object_type"] = v["object_type"].(string)
			object_map["key_value_pairs"] = v["key_value_pairs"].(string)
			object_map["time_modified_since_creation"] = v["time_modified_since_creation"].(float64)
			
			// db insert
			mc := session.DB("landline").C("SyncableObjects")
			err = mc.Insert(object_map)
			if err != nil {
				panic(err)
			}
			log.Printf("Inserted object %q into database.", v["uuid"].(string))
		}
	}
}