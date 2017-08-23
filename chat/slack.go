package chat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type SlackWebhookContent struct {
	Text string `json:"text"`
}

type Slack struct {
	WebhookUrl string
}

func (s *Slack) TeeMessage(message string) (*http.Response, error) {
	fmt.Println(message)
	return s.PostMessage(message)
}

func (s *Slack) PostMessage(message string) (*http.Response, error) {
	content, err := json.Marshal(SlackWebhookContent{Text: message})
	if err != nil {
		return nil, err
	}

	res, err := http.Post(s.WebhookUrl, "application/json", bytes.NewReader(content))
	if err != nil {
		return nil, err
	}
	return res, nil
}
