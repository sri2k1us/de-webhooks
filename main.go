package main

import (
	"database/sql"
	"flag"
	"log"

	"github.com/cyverse-de/configurate"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
)

//Log define a logrus logger
var Log = logrus.WithFields(logrus.Fields{
	"service": "de-webhooks",
	"art-id":  "de-webhooks",
	"group":   "org.cyverse",
})

//DBConnection db connection to DE database
type DBConnection struct {
	db *sql.DB
}

//Queue name
const queuename = "notification-queue"

var config *viper.Viper

func main() {

	logrus.SetFormatter(&logrus.JSONFormatter{})

	var (
		cfgPath = flag.String("config", "/etc/iplant/de/webhooks.yml", "The path to the config file")
	)

	flag.Parse()

	if *cfgPath == "" {
		Log.Fatal("--config must be set")
	}

	cfg, err := configurate.InitDefaults(*cfgPath, configurate.JobServicesDefaults)
	if err != nil {
		Log.Fatal(err)
	}
	config = cfg

	Log.Print("Connecting to amqp...")
	conn, err := amqp.Dial(config.GetString("amqp.uri"))
	if err != nil {
		Log.Fatal(err)
	}
	defer conn.Close()

	Log.Printf("Connected to amqp.")
	ch, err := conn.Channel()
	if err != nil {
		Log.Fatal(err)
	}
	defer ch.Close()

	err = ch.ExchangeDeclare(
		cfg.GetString("amqp.exchange.name"), // name
		"topic", // type
		true,    // durable
		false,   // auto-deleted
		false,   // internal
		false,   // no-wait
		nil,     // arguments
	)
	if err != nil {
		Log.Fatal(err)
	}

	q, err := ch.QueueDeclare(
		queuename, // name
		false,     // durable
		false,     // delete when usused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		Log.Fatal(err)
	}

	log.Printf("Binding queue %s to exchange %s with routing key %s",
		q.Name, cfg.GetString("amqp.exchange.name"), config.GetString("amqp.routing"))
	err = ch.QueueBind(
		q.Name, // queue name
		config.GetString("amqp.routing"),    // routing key
		cfg.GetString("amqp.exchange.name"), // exchange
		false,
		nil)
	if err != nil {
		Log.Fatal(err)
	}

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto ack
		false,  // exclusive
		false,  // no local
		false,  // no wait
		nil,    // args
	)
	if err != nil {
		Log.Fatal(err)
	}

	DBConnection := NewDBConnection(Init())
	defer DBConnection.db.Close()

	forever := make(chan bool)
	go func() {
		ProcessMessages(DBConnection, msgs)
	}()
	Log.Print("****Waiting for notifications. Press Ctrl + c to quit!****")
	<-forever
}

//NewDBConnection makes a new DBConnection
func NewDBConnection(db *sql.DB) *DBConnection {
	return &DBConnection{
		db: db,
	}
}
