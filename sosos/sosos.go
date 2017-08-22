package sosos

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/exec"
	"strconv"
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

type SlackWebhookContent struct {
	Text string `json:"text"`
}

type CancelHandler struct {
	Ch chan int
}

func (c CancelHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	rw.WriteHeader(http.StatusOK)
	fmt.Fprintln(rw, "Cancel request is accepted")
	go func() {
		c.Ch <- STATE_CANCELED
	}()
}

type SuspendHandler struct {
	Ch           chan int
	SuspendSecCh chan int
}

func (s SuspendHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		fmt.Fprintln(rw, "Query parsing failed")
		log.Println("Query parsing failed")
		return
	}

	suspendSecStrs, ok := req.Form["suspendSec"]
	if !ok || suspendSecStrs == nil || len(suspendSecStrs) < 1 {
		rw.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(rw, "Parameter named \"SuspendSec\" is not found in request")
		log.Println("Parameter named \"SuspendSec\" is not found in request")
		return
	}

	suspendSec, err := strconv.Atoi(suspendSecStrs[0])

	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(rw, "Parameter named \"SuspendSec\" is invalid in request")
		log.Println(err)
		return
	}
	rw.WriteHeader(http.StatusOK)
	fmt.Fprintf(rw, "Suspend request(%d seconds) is accepted\n", suspendSec)

	go func() {
		//s.Ch <- STATE_SUSPENDED
		s.SuspendSecCh <- suspendSec
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

func Execute(commands []string, sleepSec int, port int, insecureFlag bool, webhookUrl string, noResultFlag bool, noCancelLinkFlag bool) error {
	suspendSecCh := make(chan int)
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

	if !noCancelLinkFlag {
		message += "If you want to suspend or cancel this command, please click the following Link\n"
		message += fmt.Sprintf("[Cancel](%s/cancel)\n", cancelServerUrl)
		message += fmt.Sprintf("[Suspend  5 minutes](%s/suspend?suspendSec=300)\n", cancelServerUrl)
		message += fmt.Sprintf("[Suspend 20 minutes](%s/suspend?suspendSec=1200)\n", cancelServerUrl)
		message += fmt.Sprintf("[Suspend 60 minutes](%s/suspend?suspendSec=3600)\n", cancelServerUrl)
	}

	res, err := slack.teeMessage(message)
	if err != nil {
		return err
	}

	fmt.Println("http response " + res.Status)

	isCanceled, err := waitWithCancelServer(sleepSec, port, suspendSecCh, slack)
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

		if !noResultFlag {
			resultRes, err := slack.teeMessage(fmt.Sprintf("result:\n```\n%s```", string(out)))
			if err != nil {
				return err
			}
			fmt.Println("http response " + resultRes.Status)
		} else {
			slack.teeMessage("finish!")
		}
	} else {
		res, err := slack.teeMessage("Command is canceled")
		if err != nil {
			return err
		}
		fmt.Println("http response " + res.Status)
	}
	return nil
}

func waitWithCancelServer(sleepSec int, port int, suspendSecCh chan int, slack Slack) (bool, error) {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false, err
	}
	sl, err := stoppableListener.New(l)
	if err != nil {
		return false, err
	}

	ch := make(chan int)
	http.Handle("/cancel", CancelHandler{ch})
	http.Handle("/suspend", SuspendHandler{ch, suspendSecCh})
	s := http.Server{}

	go func() {
		s.Serve(sl)
	}()

	go func() {
		baseTime := time.Now()
		ticker := time.NewTicker(500 * time.Millisecond)

		for t := range ticker.C {
			select {
			case state := <-ch:
				switch state {
				case STATE_CANCELED:
					return
				}
			case suspendSec := <-suspendSecCh:
				message := fmt.Sprintf("Time to command execution has been suspended by %d seconds.", suspendSec)
				slack.teeMessage(message)
				baseTime = baseTime.Add(time.Duration(suspendSec) * time.Second)
			default:
			}

			if t.Sub(baseTime).Seconds() > float64(sleepSec) {
				ch <- STATE_SLEEP_FINISHED
				return
			}
		}
	}()

	state := <-ch
	sl.Stop()

	return state == STATE_CANCELED, nil
}
