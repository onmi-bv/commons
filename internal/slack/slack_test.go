package slack

import (
	"flag"
	"os"
	"os/user"
	"testing"
)

var (
	client *Client
)

func TestMain(m *testing.M) {
	client = &Client{}
	flag.StringVar(&client.URL, "url", "", "webhook url")
	flag.Parse()
	os.Exit(m.Run())
}

func TestSendMessage(t *testing.T) {
	if client.URL == "" {
		t.Error("-url flag must be specified.")
		return
	}
	msg := &Message{}
	msg.Channel = "#slack-go-test"
	msg.Text = "Slack API Test from go"
	user, _ := user.Current()
	msg.Username = user.Username
	client.SendMessage(msg)
}

func TestSendMessageWithAttachement(t *testing.T) {
	if client.URL == "" {
		t.Error("-url flag must be specified.")
		return
	}
	msg := &Message{}
	msg.Channel = "#slack-go-test"
	msg.Text = "Slack API Test from go - with attachment"
	user, _ := user.Current()
	msg.Username = user.Username

	attach := msg.NewAttachment()
	attach.Text = "This is an attachment!"
	attach.Pretext = "This is the pretext of an attachment"
	attach.Color = "good"
	attach.Fallback = "That's the fallback field"

	field := attach.NewField()
	field.Title = "Field one"
	field.Value = "Field one value"

	client.SendMessage(msg)
}
