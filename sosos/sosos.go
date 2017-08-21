package sosos

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"time"

	"os"

	"bytes"

	"strings"

	"net/url"

	"github.com/hydrogen18/stoppableListener"
)

const (
	STATE_CANCELED = iota + 1
	STATE_SLEEP_FINISHED
)

type CancelServer struct {
	Ch chan int
}

type SlackWebhookContent struct {
	Text string `json:"text"`
}

func (c CancelServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	rw.WriteHeader(http.StatusOK)
	fmt.Fprintln(rw, "Cancel request is accepted")
	go func() {
		c.Ch <- STATE_CANCELED
	}()
}

func getCancelServerUrl(insecureFlag bool, port int) (string, error) {
	protocol := "http"
	if !insecureFlag {
		protocol = protocol + "s"
	}

	hostname, err := os.Hostname()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s://%s:%d", protocol, hostname, port), nil
}

type Slack struct {
	WebhookUrl string
}

func (s *Slack) teeMessage(message string) (*http.Response, error) {
	fmt.Println(message)
	return s.postMessage(message)
}

func (s *Slack) postMessage(message string) (*http.Response, error) {
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

func Execute(commands []string, sleepSec int, port int, insecureFlag bool, webhookUrl string) error {
	slack := Slack{WebhookUrl: webhookUrl}
	cancelServerUrl, err := getCancelServerUrl(insecureFlag, port)
	if err != nil {
		return err
	}

	u, err := url.Parse(cancelServerUrl)
	if err != nil {
		return err
	}

	message := fmt.Sprintf("The command `%s` will be executed after %d seconds on `%s`\n",
		strings.Join(commands, " "),
		sleepSec,
		u.Hostname())

	message += "If you want to cancel this command, please click the following Link\n"
	message += fmt.Sprintf("[Cancel](%s/cancel)", cancelServerUrl)
	res, err := slack.teeMessage(message)
	if err != nil {
		return err
	}

	fmt.Println("http response " + res.Status)

	isCanceled, err := waitWithCancelServer(sleepSec, port)
	if err != nil {
		return err
	}

	if !isCanceled {
		res, err := slack.teeMessage("Command execution is started!")
		if err != nil {
			return err
		}
		fmt.Println("http response " + res.Status)

		out, err := exec.Command(commands[0], commands[1:]...).CombinedOutput()
		if err != nil {
			return err
		}

		resultRes, err := slack.teeMessage(fmt.Sprintf("result:\n```\n%s```", string(out)))
		if err != nil {
			return err
		}
		fmt.Println("http response " + resultRes.Status)
	} else {
		res, err := slack.teeMessage("Command is canceled")
		if err != nil {
			return err
		}
		fmt.Println("http response " + res.Status)
	}
	return nil
}

func waitWithCancelServer(sleepSec int, port int) (bool, error) {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false, err
	}
	sl, err := stoppableListener.New(l)
	if err != nil {
		return false, err
	}

	ch := make(chan int)
	http.Handle("/cancel", CancelServer{ch})
	s := http.Server{}

	go func() {
		s.Serve(sl)
	}()

	go func() {
		time.Sleep(time.Duration(sleepSec) * time.Second)
		ch <- STATE_SLEEP_FINISHED
	}()

	state := <-ch
	sl.Stop()

	return state == STATE_CANCELED, nil
}
