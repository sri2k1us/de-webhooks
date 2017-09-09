package main

import (
	"github.com/cyverse-de/dbutil"
	_ "github.com/lib/pq"
)

//Init init database connection
func Init() {
	dburi := cfg.GetString("db.uri")
	connector, err := dbutil.NewDefaultConnector("1m")
	if err != nil {
		Log.Fatal(err)
	}

	Log.Printf("uri is %s", dburi)

	db, err := connector.Connect("postgres", dburi)
	if err != nil {
		Log.Fatal(err)
	}
	defer db.Close()

	Log.Println("Connected to the database.")

	if err = db.Ping(); err != nil {
		Log.Fatal(err)
	}

	Log.Println("Successfully pinged the database.")
}
