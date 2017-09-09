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

const templatetext = `
{
	"text": "{{.Msg}}. {{if .Completed}} <{{.Link}}|{{.LinkText}}> {{- end}}"
}
`

//compltedstatus Analysis completed status
const compltedstatus = "Completed"

//Payload payload to post to the webhooks
type Payload struct {
	Msg, Link, LinkText string
	Completed           bool
}

//ProcessMessages process the received message for post to webhooks
func ProcessMessages(msgs <-chan amqp.Delivery) {
	for d := range msgs {
		Log.Printf("[X] Notification %s", d.Body)
		isAnalysis := isAnalysisNotification(d.Body)
		if isAnalysis {
			postToHook(d.Body)
		}
	}
}

//Check if the Notification is for Analysis
func isAnalysisNotification(msg []byte) bool {
	value, _, _, err := jsonparser.Get(msg, "message", "type")
	if err != nil {
		Log.Fatal(err)
	}
	log.Printf("Notification type is %s", string(value))
	if string(value) == "analysis" {
		return true
	}
	return false
}

//Prepare payload from template
func preparePayloadFromTemplate(msg []byte) *strings.Reader {
	var buf1 bytes.Buffer
	var postbody Payload
	t := template.Must(template.New("slack").Parse(templatetext))
	w := io.MultiWriter(&buf1)
	isCompleted := isAnalysisCompleted(msg)
	postbody = Payload{getMessage(msg), cfg.GetString("de.base") + getResultFolder(msg), "Go to results folder in DE", isCompleted}
	t.Execute(w, postbody)
	log.Printf("message to post-> %s", buf1.String())
	return strings.NewReader(buf1.String())
}

//check if the analysis is completed
func isAnalysisCompleted(msg []byte) bool {
	Log.Printf("Getting analysis status")
	value, _, _, err := jsonparser.Get(msg, "message", "payload", "analysisstatus")
	if err != nil {
		Log.Fatal(err)
	}
	Log.Printf("Analysis status is %s", value)
	if string(value) == compltedstatus {
		return true
	}
	return false
}

//get analysis result folder
func getResultFolder(msg []byte) string {
	Log.Printf("Getting result folder")
	value, _, _, err := jsonparser.Get(msg, "message", "payload", "analysisresultsfolder")
	if err != nil {
		Log.Fatal(err)
	}
	Log.Printf("Analysis result folder is %s", value)
	return string(value)
}

//get message from notfication
func getMessage(msg []byte) string {
	value, _, _, err := jsonparser.Get(msg, "message", "message", "text")
	if err != nil {
		Log.Fatal(err)
	}
	Log.Printf("Message is %s", value)
	return string(value)
}

//post to webhooks
func postToHook(msg []byte) {
	resp, err := http.Post("https://hooks.slack.com/services/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", "application/json", preparePayloadFromTemplate(msg))
	if err != nil {
		Log.Printf("Error posting to hook %s", err)
	}
	defer resp.Body.Close()
}
