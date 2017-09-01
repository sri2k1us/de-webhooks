package main

import (
	"fmt"
	"log"
	"os"

	"github.com/streadway/amqp"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
		panic(fmt.Sprintf("%s: %s", msg, err))
	}
}

func main() {
	if len(os.Args) < 3 {
		log.Printf("Usage: %s [amqp_url] [binding_key]...", os.Args[0])
		os.Exit(0)
	}

	conn, err := amqp.Dial(os.Args[1])
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	err = ch.ExchangeDeclare(
		"de",    // name
		"topic", // type
		true,    // durable
		false,   // auto-deleted
		false,   // internal
		false,   // no-wait
		nil,     // arguments
	)
	failOnError(err, "Failed to declare an exchange")

	q, err := ch.QueueDeclare(
		"",    // name
		false, // durable
		false, // delete when usused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	failOnError(err, "Failed to declare a queue")

	for _, s := range os.Args[2:] {
		log.Printf("Binding queue %s to exchange %s with routing key %s",
			q.Name, "Notifications topic", s)
		err = ch.QueueBind(
			q.Name, // queue name
			s,      // routing key
			"de",   // exchange
			false,
			nil)
		failOnError(err, "Failed to bind a queue")
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
	failOnError(err, "Failed to register a consumer")

	forever := make(chan bool)

	go func() {
		ProcessMessages(msgs)
	}()

	log.Printf(" [*] Waiting for notifications. To exit press CTRL+C")
	<-forever
}
