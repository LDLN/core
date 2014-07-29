package controllers

import (
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"github.com/revel/revel"
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
		
	return c.Render()
}