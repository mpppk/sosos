package sosos

import (
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"time"

	"os"

	"bytes"
	"encoding/json"

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

func Execute(commands []string, sleepSec int, port int, insecureFlag bool, webhookUrl string) error {
	isCanceled, err := waitWithCancelServer(sleepSec, port, insecureFlag, webhookUrl)
	if err != nil {
		return err
	}

	if !isCanceled {
		fmt.Println("Start command execution")
		out, err := exec.Command(commands[0], commands[1:]...).CombinedOutput()
		if err != nil {
			return err
		}

		fmt.Println("---- command output ----")
		fmt.Println(string(out))
		fmt.Println("---- command output ----")
		// 結果をwebhookで通知
	} else {
		fmt.Println("command is canceled")
		// cancelされたことをwebhookで通知

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
	hostname, err := os.Hostname()
	if err != nil {
		return false, err
	}

	protocol := "http"
	if !insecureFlag {
		protocol = protocol + "s"
	}

	fmt.Printf("Cancel URL is %s://%s:%d/cancel\n", protocol, hostname, port)

	// webhook
	content, err := json.Marshal(SlackWebhookContent{Text: "hoge"})
	if err != nil {
		return false, err
	}

	fmt.Println(string(content))

	res, err := http.Post(webhookUrl, "application/json", bytes.NewReader(content))
	if err != nil {
		return false, nil
	}

	fmt.Printf("%v", res)

	go func() {
		time.Sleep(time.Duration(sleepSec) * time.Second)
		ch <- STATE_SLEEP_FINISHED
	}()

	state := <-ch
	sl.Stop()

	return state == STATE_CANCELED, nil
}
