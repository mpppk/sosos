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

func postMessageToSlack(url, message string) (*http.Response, error) {
	content, err := json.Marshal(SlackWebhookContent{Text: message})
	if err != nil {
		return nil, err
	}

	res, err := http.Post(url, "application/json", bytes.NewReader(content))
	if err != nil {
		return nil, err
	}
	return res, nil
}

func Execute(commands []string, sleepSec int, port int, insecureFlag bool, webhookUrl string) error {
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
	message += fmt.Sprintf("[Cancel](%s)", cancelServerUrl)

	// webhook
	res, err := postMessageToSlack(webhookUrl, message)
	if err != nil {
		return err
	}

	fmt.Println("http response " + res.Status)

	isCanceled, err := waitWithCancelServer(sleepSec, port, insecureFlag, webhookUrl)
	if err != nil {
		return err
	}

	if !isCanceled {
		message := "Command execution is started!"
		fmt.Println(message)
		res, err := postMessageToSlack(webhookUrl, message)
		if err != nil {
			return err
		}

		fmt.Println("http response " + res.Status)

		out, err := exec.Command(commands[0], commands[1:]...).CombinedOutput()
		if err != nil {
			return err
		}

		fmt.Println("---- command output ----")
		fmt.Println(string(out))
		fmt.Println("---- command output ----")

		resultMessage := fmt.Sprintf("result:\n```%s```", string(out))
		resultRes, err := postMessageToSlack(webhookUrl, resultMessage)
		if err != nil {
			return err
		}

		fmt.Println("http response " + resultRes.Status)
	} else {
		fmt.Println("command is canceled")

	}
	return nil
}

func waitWithCancelServer(sleepSec int, port int, insecureFlag bool, webhookUrl string) (bool, error) {
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

	cancelServerUrl, err := getCancelServerUrl(insecureFlag, port)
	if err != nil {
		return false, err
	}

	fmt.Printf("Cancel URL is %s/cancel\n", cancelServerUrl)

	go func() {
		time.Sleep(time.Duration(sleepSec) * time.Second)
		ch <- STATE_SLEEP_FINISHED
	}()

	state := <-ch
	sl.Stop()

	return state == STATE_CANCELED, nil
}
