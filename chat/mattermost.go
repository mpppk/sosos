package chat

import "fmt"

type Mattermost struct {
	*Slack
}

func (s *Mattermost) GenerateLinkStr(url, title string) string {
	return fmt.Sprintf("[%s](%s)", url, title)
}
