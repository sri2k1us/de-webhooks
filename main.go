package main

import (
	"flag"
	"log"
	"os"

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

var cfg *viper.Viper

func main() {

	logrus.SetFormatter(&logrus.JSONFormatter{})

	var (
		cfgPath = flag.String("config", "/etc/iplant/de/webhooks.yml", "The path to the config file")
		err     error
	)

	flag.Parse()

	if *cfgPath == "" {
		Log.Fatal("--config must be set")
	}

	if cfg, err = configurate.InitDefaults(*cfgPath, configurate.JobServicesDefaults); err != nil {
		Log.Fatal(err)
	}

	if len(os.Args) < 2 {
		Log.Printf("Usage: %s [binding_key]...", os.Args[0])
		os.Exit(0)
	}

	Log.Printf("Connecting to amqp %s", cfg.GetString("amqp.uri"))
	conn, err := amqp.Dial(cfg.GetString("amqp.uri"))
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
		"",    // name
		false, // durable
		false, // delete when usused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		Log.Fatal(err)
	}

	for _, s := range os.Args[1:] {
		log.Printf("Binding queue %s to exchange %s with routing key %s",
			q.Name, "Notifications topic", s)
		err = ch.QueueBind(
			q.Name, // queue name
			s,      // routing key
			"de",   // exchange
			false,
			nil)
		if err != nil {
			Log.Fatal(err)
		}
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

	forever := make(chan bool)

	go func() {
		ProcessMessages(msgs)
	}()

	Log.Print("****Waiting for notfications. Press Ctrl + c to quit!****")
	<-forever
}
