package chat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type SlackWebhookContent struct {
	Text     string `json:"text"`
	Username string `json:"username"`
}

type Slack struct {
	WebhookUrl string
}

func (s *Slack) TeeMessage(message string) (*http.Response, error) {
	fmt.Println(message)
	return s.PostMessage(message)
}

func (s *Slack) PostMessage(message string) (*http.Response, error) {
	content, err := json.Marshal(SlackWebhookContent{Text: message, Username: "SOSOS"})
	if err != nil {
		return nil, err
	}

	res, err := http.Post(s.WebhookUrl, "application/json", bytes.NewReader(content))
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (s *Slack) PostResultMessage(results []string) (*http.Response, error) {
	value := fmt.Sprintf("```%s```", strings.Join(results, "\n"))
	message := &Message{
		Username: "SOSOS",
		Attachments: []*Attachment{{
			Title:    "result",
			Text:     value,
			MrkdwnIn: []string{"text"},
		}},
	}
	content, err := json.Marshal(message)
	if err != nil {
		return nil, err
	}

	res, err := http.Post(s.WebhookUrl, "application/json", bytes.NewReader(content))
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (s *Slack) GenerateLinkStr(url, title string) string {
	return fmt.Sprintf("<%s|%s>", url, title)
}

func IsSlackWebhookUrl(url string) bool {
	return strings.Contains(url, "hooks.slack.com")
}
