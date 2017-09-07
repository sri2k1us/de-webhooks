package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"strings"
	"text/template"

	"github.com/buger/jsonparser"
	"github.com/streadway/amqp"
)

//Slack json stuct for Slack
type Slack struct {
	Text string `json:"text"`
}

const templatetext = `
{
	"text": {{ printf "%q" .}}
}
`

//ProcessMessages process the received message for post to webhooks
func ProcessMessages(msgs <-chan amqp.Delivery) {
	for d := range msgs {
		log.Printf(" [x] %s", d.Body)
		value, _, _, err := jsonparser.Get(d.Body, "message", "message", "text")
		if err != nil {
			log.Printf("Json Parse Error: %s", d.Body)
		}
		log.Printf("****Message***--> %s", value)
		postToHook(value)

	}

}

func prepareMessageFromTemplate(msg []byte) *strings.Reader {
	var buf1 bytes.Buffer
	t := template.Must(template.New("slack").Parse(templatetext))
	w := io.MultiWriter(&buf1)
	t.Execute(w, string(msg))
	log.Printf("message to post-> %s", buf1.String())
	return strings.NewReader(buf1.String())
}

func postToHook(value []byte) {
	resp, err := http.Post("https://hooks.slack.com/services/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", "application/json", prepareMessageFromTemplate(value))
	if err != nil {
		log.Printf("Error posting to hook %s", err)
	}
	defer resp.Body.Close()
}
