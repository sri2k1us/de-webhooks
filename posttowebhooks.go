package main

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"text/template"

	"github.com/buger/jsonparser"
	"github.com/streadway/amqp"
)

//compltedstatus Analysis completed status
const compltedstatus = "Completed"
const failedstatus = "Failed"
const dequerystring = "?type=data&folder="

//Payload payload to post to the webhooks
type Payload struct {
	ID, Name, Msg, Link, LinkText, Type string
	Completed                           bool
}

//Subscription defines user subscriptions to webhooks
type Subscription struct {
	id, templatetype, url string
	topics                []string
}

//template cache
var templatesmap map[string]string

//ProcessMessages process the received message for post to webhooks
func ProcessMessages(d *DBConnection, msgs <-chan amqp.Delivery) {
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
		uid := getUserID(d, delivery.Body)
		if uid != "" {
			postToHook(d, uid, delivery.Body)
		} else {
			Log.Error("User not found!")
		}
	}
}

//getUserID Get user id for this Notification
func getUserID(d *DBConnection, msg []byte) string {
	value, _, _, err := jsonparser.Get(msg, "message", "user")
	if err != nil {
		Log.Error(err)
		return ""
	}
	Log.Printf("user is %s", string(value))
	uid, err := d.getUserInfo(string(value) + "@" + config.GetString("user.suffix"))
	if err != nil {
		Log.Error(err)
		return ""
	}
	return uid
}

//post to webhooks
func postToHook(d *DBConnection, uid string, msg []byte) {
	subs, err := d.getUserSubscriptions(uid)
	if err != nil {
		Log.Error(err)
		return
	}
	Log.Printf("No. of subscriptions found: %d", len(subs))
	if len(subs) > 0 {
		for _, v := range subs {
			if isNotificationInTopic(msg, v.topics) {
				resp, err := http.Post(v.url, "application/json", preparePayloadFromTemplate(templatesmap[v.templatetype], msg))
				if err != nil {
					Log.Printf("Error posting to hook %s", err)
				}
				defer resp.Body.Close()
			}
		}
	}
}

//isNotificationInTopic check if user is subscribed to this notification topic
func isNotificationInTopic(msg []byte, topics []string) bool {
	value, _, _, err := jsonparser.Get(msg, "message", "type")
	if err != nil {
		Log.Error(err)
		return false
	}

	if len(topics) < 1 {
		return false
	}

	for _, to := range topics {
		if string(value) == to {
			Log.Printf("Subscription topic found: %s", to)
			return true
		}
	}
	return false

}

//Prepare payload from template
func preparePayloadFromTemplate(templatetext string, msg []byte) *strings.Reader {
	var buf1 bytes.Buffer
	var postbody Payload
	if len(templatetext) == 0 {
		payload := string(msg)
		Log.Printf("Empty Template. message to post: %s", payload)
		return strings.NewReader(payload)
	}
	t := template.Must(template.New("newtemplate").Parse(templatetext))
	w := io.MultiWriter(&buf1)
	isCompleted := (getType(msg) == "analysis") && isAnalysisCompleted(msg)
	postbody = Payload{ID: getID(msg),
		Msg:      getMessage(msg),
		Name:     getName(msg),
		Type:     getType(msg),
		Link:     config.GetString("de.base") + dequerystring + getResultFolder(msg),
		LinkText: "Go to results folder in DE", Completed: isCompleted}
	t.Execute(w, postbody)
	Log.Printf("message to post: %s", buf1.String())
	return strings.NewReader(buf1.String())
}

//check if it is an analysis notification
func getType(msg []byte) string {
	value, _, _, err := jsonparser.Get(msg, "message", "type")
	if err != nil {
		Log.Error(err)
		return ""
	}
	return string(value)
}

//check if the analysis is completed
func isAnalysisCompleted(msg []byte) bool {
	Log.Printf("Getting analysis status")
	value, _, _, err := jsonparser.Get(msg, "message", "payload", "analysisstatus")
	if err != nil {
		Log.Error(err)
	}
	Log.Printf("Analysis status is %s", value)
	if string(value) == compltedstatus || string(value) == failedstatus {
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

//get id from notification
func getID(msg []byte) string {
	value, _, _, err := jsonparser.Get(msg, "message", "payload", "app_id")
	if err != nil {
		Log.Error(err)
		return ""
	}
	Log.Printf("id is %s", value)
	return string(value)
}

//get name from notification
func getName(msg []byte) string {
	value, _, _, err := jsonparser.Get(msg, "message", "payload", "name")
	if err != nil {
		Log.Error(err)
		return ""
	}
	Log.Printf("name is %s", value)
	return string(value)
}
