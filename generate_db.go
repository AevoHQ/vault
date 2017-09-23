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

	if _, err := r.DBCreate(dbModel).RunWrite(session); err != nil {
		log.Fatalln("Unable to create model database: ", err)
		return
	}

	if _, err := r.DBCreate(dbData).RunWrite(session); err != nil {
		log.Fatalln("Unable to create data database: ", err)
		return
	}

	session.Use(dbModel)

	if _, err := r.TableCreate("schema").RunWrite(session); err != nil {
		log.Fatalln("Unable to create schema table: ", err)
		return
	}

	if _, err := r.TableCreate("model").RunWrite(session); err != nil {
		log.Fatalln("Unable to create model table: ", err)
		return
	}
}
