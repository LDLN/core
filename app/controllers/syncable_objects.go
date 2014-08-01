package controllers

import (
	"encoding/json"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"github.com/revel/revel"
	"github.com/nu7hatch/gouuid"
	"github.com/ldln/landline-basestation/app/routes"
)

type SyncableObjects struct {
	*revel.Controller
}

func (c SyncableObjects) ListDataTypes() revel.Result {
	
	// connect to mongodb
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()
	
	// query
	dbc := session.DB("landline").C("Schemas")
	var results []map[string]interface{}
	err = dbc.Find(bson.M{}).All(&results)
	if err != nil {
		revel.TRACE.Println(err)
	}
	
	return c.Render(results)
}

func (c SyncableObjects) CreateObjectForm(object_key string) revel.Result {
	
	// connect to mongodb
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()
	
	// query
	dbc := session.DB("landline").C("Schemas")
	var result map[string]interface{}
	err = dbc.Find(bson.M{"object_key" : object_key}).One(&result)
	if err != nil {
		revel.TRACE.Println(err)
	}
		
	return c.Render(result)
}

func (c SyncableObjects) CreateObjectAction(object_key string) revel.Result {
	
	revel.TRACE.Println(c.Params.Values)
	
	// build kv map
	key_values := make(map[string]interface{})
	
	// parse params and put into map
	for k, v := range c.Params.Values {
		if k != "object_key" {
			key_values[k] = v[0]
		}
	}
	revel.TRACE.Println(key_values)
	
	// convert map to json to string of json
	key_values_map, err := json.Marshal(key_values)
	if err != nil {
		revel.TRACE.Println(err)
	}
	key_values_string := string(key_values_map[:])
	revel.TRACE.Println(key_values_string)
	
	// encrypt json string
	key_values_string_encrypted := encrypt([]byte(c.Session["kek"]), []byte(key_values_string))
	revel.TRACE.Println(key_values_string_encrypted)
	revel.TRACE.Println(string(decrypt([]byte(c.Session["kek"]), []byte(key_values_string_encrypted))))
	
	// create object
	object_map := make(map[string]interface{})
	uuid, err := uuid.NewV4()
	object_map["uuid"] = uuid.String()
	object_map["object_type"] = object_key
	object_map["key_value_pairs"] = key_values_string_encrypted
	object_map["time_modified_since_creation"] = float64(0)

	// connect to mongodb
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()
	
	// insert into db
	dbc := session.DB("landline").C("SyncableObjects")
	err = dbc.Insert(object_map)
	if err != nil {
		panic(err)
	}
	
	// redirect
	c.Flash.Success("Object created")
	return c.Redirect(routes.SyncableObjects.CreateObjectForm(object_key))
}

func (c SyncableObjects) ViewObject(uuid string) revel.Result {
	
	// connect to mongodb
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()
	
	// query
	dbc := session.DB("landline").C("SyncableObjects")
	var result map[string]string
	err = dbc.Find(bson.M{"uuid" : uuid}).One(&result)
	if err != nil {
		revel.TRACE.Println(err)
	}
	revel.TRACE.Println(result["key_value_pairs"])
	
	// decrypt key_value_pairs
	key_values_string := string(decrypt([]byte(c.Session["kek"]), []byte(result["key_value_pairs"])))
	
	// convert string of json to json to map
	byt := []byte((key_values_string))
	var key_values map[string]interface{}
	if err := json.Unmarshal(byt, &key_values); err != nil {
		panic(err)
	}
		
	return c.Render(result, key_values)
}










