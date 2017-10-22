package main

import (
	"log"

	r "gopkg.in/gorethink/gorethink.v3"
)

func generate(databaseIP string) {
	session, err := r.Connect(r.ConnectOpts{
		Address: databaseIP,
	})
	if err != nil {
		log.Fatalln(err)
		return
	}

	if _, err := r.DBCreate(dbName).RunWrite(session); err != nil {
		log.Fatalln("Unable to create database: ", err)
		return
	}

	session.Use(dbName)

	if _, err := r.TableCreate("schema").RunWrite(session); err != nil {
		log.Fatalln("Unable to create schema table: ", err)
		return
	}

	if _, err := r.TableCreate("model").RunWrite(session); err != nil {
		log.Fatalln("Unable to create model table: ", err)
		return
	}

	if _, err := r.TableCreate("data").RunWrite(session); err != nil {
		log.Fatalln("Unable to create data table: ", err)
		return
	}
	if _, err := r.Table("data").IndexCreate("scope").RunWrite(session); err != nil {
		log.Fatalln("Unable to create `scope` secondary index on data table: ", err)
		return
	}
	if _, err := r.Table("data").IndexCreate("time").RunWrite(session); err != nil {
		log.Fatalln("Unable to create `time` secondary index on data table: ", err)
		return
	}

}
