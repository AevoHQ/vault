package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/urfave/cli"
	r "gopkg.in/gorethink/gorethink.v3"
)

func main() {

	app := cli.NewApp()

	app.Name = "Aevo Vault"
	app.Usage = "Aevo data storage API."

	app.Version = "1.0-dev"
	app.Copyright = "(c) 2017 Harrison Grodin"
	app.Authors = []cli.Author{
		cli.Author{
			Name:  "Harrison Grodin",
			Email: "grodinh@winchesterthurston.org",
		},
	}

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "ip",
			Value:  ":3000",
			Usage:  "server IP address",
			EnvVar: "AEVO_IP",
		},
		cli.StringFlag{
			Name:   "database-ip",
			Value:  "localhost:28015",
			Usage:  "database IP address",
			EnvVar: "AEVO_DATABASE_IP",
		},
		cli.StringFlag{
			Name:        "db",
			Value:       "aevo",
			Usage:       "database name",
			Destination: &dbName,
			EnvVar:      "AEVO_DB",
		},
	}

	app.Action = func(c *cli.Context) error {
		route(
			c.String("ip"),
			c.String("database-ip"),
		)
		return nil
	}

	app.Commands = []cli.Command{
		{
			Name:    "generate",
			Aliases: []string{"gen", "g"},
			Usage:   "generate databases",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "database-ip",
					Value: "localhost:28015",
					Usage: "database IP address",
				},
			},
			Action: func(c *cli.Context) error {
				generate(c.String("database-ip"))
				return nil
			},
		},
	}

	app.Run(os.Args)

}

var dbName = "aevo"

func route(IP string, databaseIP string) {

	router := gin.Default()
	scope := router.Group("/scope")

	session, err := r.Connect(r.ConnectOpts{
		Address:  databaseIP,
		Database: dbName,
	})
	if err != nil {
		log.Fatalln(err)
	}
	routeState(scope, session)

	routeSchema(scope, session)
	routeModel(scope, session)

	routeData(scope, session)

	router.Run(IP)

}
