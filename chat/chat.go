package chat

import (
	"net/http"
)

type Service interface {
	PostMessage(message string) (*http.Response, error)
	TeeMessage(message string) (*http.Response, error)
	GenerateLinkStr(url, title string) string
}
