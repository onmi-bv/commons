package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// Client ...
type Client struct {
	URL string
}

// Message ...
type Message struct {
	Text        string        `json:"text"`
	Username    string        `json:"username"`
	IconURL     string        `json:"icon_url"`
	IconEmoji   string        `json:"icon_emoji"`
	Channel     string        `json:"channel"`
	UnfurlLinks bool          `json:"unfurl_links"`
	Attachments []*Attachment `json:"attachments"`
}

// Attachment ...
type Attachment struct {
	Title    string   `json:"title"`
	Fallback string   `json:"fallback"`
	Text     string   `json:"text"`
	Pretext  string   `json:"pretext"`
	Color    string   `json:"color"`
	Fields   []*Field `json:"fields"`
}

// Field ...
type Field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// Error ...
type Error struct {
	Code int
	Body string
}

func (e *Error) Error() string {
	return fmt.Sprintf("SlackError: %d %s", e.Code, e.Body)
}

// NewClient ...
func NewClient(url string) *Client {
	return &Client{url}
}

// SendMessage ...
func (c *Client) SendMessage(msg *Message) error {

	body, _ := json.Marshal(msg)
	buf := bytes.NewReader(body)

	http.NewRequest("POST", c.URL, buf)
	resp, err := http.Post(c.URL, "application/json", buf)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t, _ := ioutil.ReadAll(resp.Body)
		return &Error{resp.StatusCode, string(t)}
	}

	return nil
}

// NewAttachment ...
func (m *Message) NewAttachment() *Attachment {
	a := &Attachment{}
	m.AddAttachment(a)
	return a
}

// AddAttachment ...
func (m *Message) AddAttachment(a *Attachment) {
	m.Attachments = append(m.Attachments, a)
}

// NewField ...
func (a *Attachment) NewField() *Field {
	f := &Field{}
	a.AddField(f)
	return f
}

// AddField ...
func (a *Attachment) AddField(f *Field) {
	a.Fields = append(a.Fields, f)
}
