/*
 *  Copyright 2014-2015 LDLN
 *
 *  This file is part of LDLN Base Station.
 *
 *  LDLN Base Station is free software: you can redistribute it and/or modify
 *  it under the terms of the GNU General Public License as published by
 *  the Free Software Foundation, either version 3 of the License, or
 *  any later version.
 *
 *  LDLN Base Station is distributed in the hope that it will be useful,
 *  but WITHOUT ANY WARRANTY; without even the implied warranty of
 *  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *  GNU General Public License for more details.
 *
 *  You should have received a copy of the GNU General Public License
 *  along with LDLN Base Station.  If not, see <http://www.gnu.org/licenses/>.
 */
package controllers

import (
	"encoding/json"
	"encoding/hex"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"github.com/revel/revel"
	"github.com/nu7hatch/gouuid"
	"github.com/ldln/landline-basestation/app/routes"
)

type SyncableObjects struct {
	*revel.Controller
}

func (c SyncableObjects) Map() revel.Result {

	// connect to mongodb
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// find any deployments
	dbd := session.DB("landline").C("Deployments")
	var deployment map[string]string
	err = dbd.Find(bson.M{}).One(&deployment)
	
	// set dek var
	dek := c.Session["kek"]
	
	return c.Render(deployment, dek)
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

func (c SyncableObjects) CreateDataTypeForm() revel.Result {
	return c.Render()
}

func (c SyncableObjects) CreateDataTypeAction() revel.Result {
	return c.Render()
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
	
	// chrome
	var hide_chrome bool
	c.Params.Bind(&hide_chrome, "hide_chrome")

	return c.Render(result, hide_chrome)
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
	kv_string_encrypted := hex.EncodeToString(encrypt([]byte(c.Session["kek"]), key_values_map))
	revel.TRACE.Println(kv_string_encrypted)
	
	// test decrypt
	kv_hex, err := hex.DecodeString(kv_string_encrypted)
	if err != nil {
		revel.TRACE.Println(err)
	}
	kv_plain := string(decrypt([]byte(c.Session["kek"]), kv_hex))
	revel.TRACE.Println(kv_plain)
	
	// create object
	object_map := make(map[string]interface{})
	uuid, err := uuid.NewV4()
	object_map["uuid"] = uuid.String()
	object_map["object_type"] = object_key
	object_map["key_value_pairs"] = kv_string_encrypted
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
	return c.Redirect(routes.SyncableObjects.ViewObject(object_key, object_map["uuid"].(string)))
}

func (c SyncableObjects) ViewObject(object_key, uuid string) revel.Result {
	
	// connect to mongodb
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()
	
	// query types
	schemasdb := session.DB("landline").C("Schemas")
	var schema map[string]interface{}
	err = schemasdb.Find(bson.M{"object_key": object_key}).One(&schema)
	if err != nil {
		revel.TRACE.Println(err)
	}
	
	// query objects
	dbc := session.DB("landline").C("SyncableObjects")
	var object map[string]string
	err = dbc.Find(bson.M{"uuid": uuid, "object_type": object_key}).One(&object)
	if err != nil {
		revel.TRACE.Println(err)
	}
	revel.TRACE.Println(object)
	
	// decrypt key_value_pairs
	kv_hex, err := hex.DecodeString(object["key_value_pairs"])
	if err != nil {
		revel.TRACE.Println(err)
	}
	kv_plain := string(decrypt([]byte(c.Session["kek"]), kv_hex))
	revel.TRACE.Println(kv_plain)
	
	// convert string of json to json to map
	byt := []byte((kv_plain))
	var key_values map[string]interface{}
	if err := json.Unmarshal(byt, &key_values); err != nil {
		panic(err)
	}
		
	return c.Render(object, key_values, schema)
}

func (c SyncableObjects) ListObjects(object_key string) revel.Result {
	
	// connect to mongodb
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()
	
	// query
	dbs := session.DB("landline").C("Schemas")
	
	var object_type map[string]interface{}
	err = dbs.Find(bson.M{"object_key": object_key}).One(&object_type)
	if err != nil {
		revel.TRACE.Println(err)
	}
	
	
	dbc := session.DB("landline").C("SyncableObjects")
	
	var results []map[string]interface{}
	err = dbc.Find(bson.M{"object_type": object_key}).All(&results)
	if err != nil {
		revel.TRACE.Println(err)
	}
	
	revel.TRACE.Println(results)
	
	// decrypt each
	var object_list []map[string]interface{}
	for u, result := range results {
		revel.TRACE.Println(u)

		// object that the client does not know about
		syncable_object_map := make(map[string]interface{})
		syncable_object_map["uuid"] = result["uuid"]
		syncable_object_map["object_type"] = result["object_type"]
		syncable_object_map["time_modified_since_creation"] = result["time_modified_since_creation"]
			
		// decrypt
		kv_hex, err := hex.DecodeString(result["key_value_pairs"].(string))
		if err != nil {
			revel.TRACE.Println(err)
		}
		kv_plain := decrypt([]byte(c.Session["kek"]), kv_hex)
		
		// unmarshal the json
		var obj_json map[string]interface{}
		if err := json.Unmarshal(kv_plain, &obj_json); err != nil {
			panic(err)
		}

		syncable_object_map["key_value_pairs_plain"] = obj_json

		object_list = append(object_list, syncable_object_map)
	}
		
	return c.Render(object_type, object_key, results, object_list)
}

func (c SyncableObjects) MarkdownEditor() revel.Result {
	return c.Render()
}










