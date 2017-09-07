package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/streadway/amqp"
)

//Slack json stuct for Slack
type Slack struct {
	Text string `json:"text"`
}

//ProcessMessages process the received message for post to webhooks
func ProcessMessages(msgs <-chan amqp.Delivery) {
	for d := range msgs {
		log.Printf(" [x] %s", d.Body)
		value, _, _, err := jsonparser.Get(d.Body, "message", "message", "text")
		if err != nil {
			log.Printf("Json Parse Error: %s", d.Body)
		}
		log.Printf("****Message***--> %s", value)
		log.Printf("message to post-> %s", "{\"text\":"+string(value)+"}")
		postToHook(value)

	}

}

func prepareMessage(msg []byte) *strings.Reader {
	//return strings.NewReader("{\"text\":\"" + string(msg) + "\"}")
	json, err := json.Marshal(Slack{Text: string(msg)})
	if err != nil {
		log.Printf("Json Encoding Error: %s", err)
	}
	return strings.NewReader(string(json))
}

func postToHook(value []byte) {
	resp, err := http.Post("https://hooks.slack.com/services/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", "application/json", prepareMessage(value))
	if err != nil {
		log.Printf("Error posting to hook %s", err)
	}
	defer resp.Body.Close()
}
