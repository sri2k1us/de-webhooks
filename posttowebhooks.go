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

//slack template
//const templatetext = `
//{
//	"text": "{{.Msg}}. {{if .Completed}} <{{.Link}}|{{.LinkText}}> {{- end}}"
//}
//`

//compltedstatus Analysis completed status
const compltedstatus = "Completed"

//Payload payload to post to the webhooks
type Payload struct {
	Msg, Link, LinkText string
	Completed           bool
}

//template cache
var templatesmap map[string]string

//ProcessMessages process the received message for post to webhooks
func (d *DBConnection) ProcessMessages(msgs <-chan amqp.Delivery) {
	if templatesmap == nil { // call only when template cache is not ready
		temmap, err := d.getTemplates()
		if err != nil {
			Log.Error(err)
			return
		}
		templatesmap = temmap
	}
	for delivery := range msgs {
		Log.Printf("[X] Notification %s", delivery.Body)
		isAnalysis := isAnalysisNotification(delivery.Body)
		uid := d.getUserID(delivery.Body)
		if isAnalysis && uid != "" {
			d.postToHook(uid, delivery.Body)
		}
	}
}

//getUserID Get user id for this Notification
func (d *DBConnection) getUserID(msg []byte) string {
	value, _, _, err := jsonparser.Get(msg, "message", "user")
	if err != nil {
		Log.Error(err)
		return ""
	}
	log.Printf("user is %s", string(value))
	uid, err := d.getUserInfo(string(value) + "@" + config.GetString("user.suffix"))
	if err != nil {
		Log.Error(err)
		return ""
	}
	return uid
}

//post to webhooks
func (d *DBConnection) postToHook(uid string, msg []byte) {
	subs, err := d.getUserSubscriptions(uid)
	if err != nil {
		Log.Error(err)
		return
	}
	Log.Printf("No. of subscriptions found: %d", len(subs))
	if len(subs) > 0 {
		for _, v := range subs {
			resp, err := http.Post(v.url, "application/json", preparePayloadFromTemplate(templatesmap[v.templatetype], msg))
			if err != nil {
				Log.Printf("Error posting to hook %s", err)
			}
			defer resp.Body.Close()
		}
	}
}

//Check if the Notification is for Analysis
func isAnalysisNotification(msg []byte) bool {
	value, _, _, err := jsonparser.Get(msg, "message", "type")
	if err != nil {
		Log.Error(err)
		return false
	}
	log.Printf("Notification type is %s", string(value))
	if string(value) == "analysis" {
		return true
	}
	return false
}

//Prepare payload from template
func preparePayloadFromTemplate(templatetext string, msg []byte) *strings.Reader {
	var buf1 bytes.Buffer
	var postbody Payload
	t := template.Must(template.New("slack").Parse(templatetext))
	w := io.MultiWriter(&buf1)
	isCompleted := isAnalysisCompleted(msg)
	postbody = Payload{getMessage(msg), config.GetString("de.base") + getResultFolder(msg), "Go to results folder in DE", isCompleted}
	t.Execute(w, postbody)
	log.Printf("message to post-> %s", buf1.String())
	return strings.NewReader(buf1.String())
}

//check if the analysis is completed
func isAnalysisCompleted(msg []byte) bool {
	Log.Printf("Getting analysis status")
	value, _, _, err := jsonparser.Get(msg, "message", "payload", "analysisstatus")
	if err != nil {
		Log.Error(err)
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
		Log.Error(err)
	}
	Log.Printf("Analysis result folder is %s", value)
	return string(value)
}

//get message from notfication
func getMessage(msg []byte) string {
	value, _, _, err := jsonparser.Get(msg, "message", "message", "text")
	if err != nil {
		Log.Error(err)
		return ""
	}
	Log.Printf("Message is %s", value)
	return string(value)
}
