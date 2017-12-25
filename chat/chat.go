package chat

import (
	"net/http"
)

type Service interface {
	PostMessage(message string) (*http.Response, error)
	PostResultMessage(results []string) (*http.Response, error)
	TeeMessage(message string) (*http.Response, error)
	GenerateLinkStr(url, title string) string
}

type Message struct {
	Username    string        `json:"username"`
	Text        string        `json:"text"`
	Attachments []*Attachment `json:"attachments"`
}

type Attachment struct {
	Title          string    `json:"title"`
	Text           string    `json:"text"`
	Fallback       string    `json:"fallback"`
	CallbackID     string    `json:"callback_id"`
	Color          string    `json:"color"`
	AttachmentType string    `json:"attachment_type"`
	Actions        []*Action `json:"actions"`
	Fields         []*Field  `json:"fields"`
	MrkdwnIn       []string  `json:"mrkdwn_in"`
}

type Action struct {
	Name    string   `json:"name"`
	Text    string   `json:"text"`
	Type    string   `json:"type"`
	Value   string   `json:"value"`
	Style   string   `json:"style,omitempty"`
	Confirm *Confirm `json:"confirm,omitempty"`
}

type Field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

type Confirm struct {
	Title       string `json:"title"`
	Text        string `json:"text"`
	OkText      string `json:"ok_text"`
	DismissText string `json:"dismiss_text"`
}

func NewMessage(text string) *Message {
	return &Message{
		Text: text,
		Attachments: []*Attachment{{
			Text:           text,
			Fallback:       "fallback",
			CallbackID:     "callbackID",
			Color:          "#3AA3E3",
			AttachmentType: "default",
			Actions: []*Action{{
				Name:  "name",
				Text:  "text",
				Type:  "button",
				Value: "value",
				Style: "danger",
				Confirm: &Confirm{
					Title:       "Are you sure?",
					Text:        "text",
					OkText:      "OK",
					DismissText: "Dismiss",
				},
			}},
		}},
	}
}
