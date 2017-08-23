package sosos

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os/exec"
	"strconv"
	"time"

	"os"

	"strings"

	"net/url"

	"bufio"

	"io/ioutil"

	"sync"

	"os/signal"
	"syscall"

	"github.com/hydrogen18/stoppableListener"
	"github.com/mpppk/sosos/chat"
	"github.com/mpppk/sosos/etc"
)

const (
	STATE_CANCELED = iota + 1
	STATE_SLEEP_FINISHED
)

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

func Execute(commands []string, sleepSec int64, port int, insecureFlag bool, webhookUrl string, noResultFlag bool, noCancelLinkFlag bool, noScriptContentFlag bool, customMessage string) error {
	suspendSecCh := make(chan int)
	slack := &chat.Slack{WebhookUrl: webhookUrl}
	cancelServerUrl, err := getCancelServerUrl(insecureFlag, port)
	if err != nil {
		return err
	}

	u, err := url.Parse(cancelServerUrl)
	if err != nil {
		return err
	}

	message := customMessage
	if message != "" {
		message += "\n"
	}
	message += fmt.Sprintf("The command `%s` will be executed after %d seconds(%s) on `%s`\n",
		strings.Join(commands, " "),
		sleepSec,
		time.Now().Add(time.Duration(sleepSec)*time.Second).Format("01/02 15:04:05"),
		u.Hostname())

	if !noScriptContentFlag {
		for _, command := range commands {
			if etc.IsScript(command) {
				fileBytes, err := ioutil.ReadFile(command)
				if err != nil {
					return err
				}

				message += fmt.Sprintf("`%s` contents:\n```\n%s\n```\n", command, string(fileBytes))
				break
			}
		}
	}

	if !noCancelLinkFlag {
		message += "If you want to suspend or cancel this command, please click the following Link\n"
		message += fmt.Sprintf("[Cancel](%s/cancel)\n", cancelServerUrl)
		message += fmt.Sprintf("[Suspend  5 minutes](%s/suspend?suspendSec=300)\n", cancelServerUrl)
		message += fmt.Sprintf("[Suspend 20 minutes](%s/suspend?suspendSec=1200)\n", cancelServerUrl)
		message += fmt.Sprintf("[Suspend 60 minutes](%s/suspend?suspendSec=3600)\n", cancelServerUrl)
	}

	res, err := slack.TeeMessage(message)
	if err != nil {
		return err
	}

	fmt.Println("http response " + res.Status)

	isCanceled, err := waitWithCancelServer(sleepSec, port, suspendSecCh, slack)
	if err != nil {
		return err
	}

	var cmdErr error
	if !isCanceled {
		res, err := slack.TeeMessage("Command execution is started!")
		if err != nil {
			return err
		}
		fmt.Println("http response " + res.Status)

		cmd := exec.Command(commands[0], commands[1:]...)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return err
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return err
		}

		if err := cmd.Start(); err != nil {
			return err
		}

		wg := &sync.WaitGroup{}
		wg.Add(2)
		resultCh := make(chan string, 0)
		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				text := scanner.Text()
				fmt.Println(text)
				resultCh <- text
			}
			wg.Done()
		}()

		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				text := scanner.Text()
				fmt.Println(text)
				resultCh <- text
			}
			wg.Done()
		}()

		go func() {
			wg.Wait()
			close(resultCh)
		}()

		var results []string
		for result := range resultCh {
			results = append(results, result)
		}

		if err := cmd.Wait(); err != nil {
			cmdErr = err
		}

		fmt.Println("finish!")

		if !noResultFlag {
			resultRes, err := slack.PostMessage(fmt.Sprintf("result:\n```\n%s\n```", strings.Join(results, "\n")))
			if err != nil {
				return err
			}
			fmt.Println("http response " + resultRes.Status)
		} else {
			slack.TeeMessage("finish!")
		}
	} else {
		res, err := slack.TeeMessage("Command is canceled")
		if err != nil {
			return err
		}
		fmt.Println("http response " + res.Status)
	}
	return cmdErr
}

func waitWithCancelServer(sleepSec int64, port int, suspendSecCh chan int, slack *chat.Slack) (bool, error) {
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
		executeTime := time.Now().Add(time.Duration(sleepSec) * time.Second)
		ticker := time.NewTicker(500 * time.Millisecond)

		remindSeconds := []int64{60, 300}
		for _, second := range remindSeconds {
			if second > sleepSec {
				remindSeconds = etc.Remove(remindSeconds, second)
			}
		}

		sigintCh := make(chan os.Signal, 1)
		signal.Notify(sigintCh, syscall.SIGINT)

		for range ticker.C {
			select {
			case state := <-ch:
				switch state {
				case STATE_CANCELED:
					return
				}
			case suspendSec := <-suspendSecCh:
				executeTime = executeTime.Add(time.Duration(suspendSec) * time.Second)
				message := fmt.Sprintf("Time to command execution has been suspended by %d seconds.(%s)",
					suspendSec,
					executeTime.Format("01/02 15:04:05"),
				)
				slack.TeeMessage(message)
			case <-sigintCh:
				slack.TeeMessage("The command is terminated by SIGINT signal")
				os.Exit(0)
			default:
			}

			remainSec := executeTime.Unix() - time.Now().Unix()
			if remainSec <= 0 {
				ch <- STATE_SLEEP_FINISHED
				return
			}

			for _, second := range remindSeconds {
				if second > remainSec {
					message := fmt.Sprintf("Remind: The command will be executed after %d seconds(%s)\n",
						remainSec,
						executeTime.Format("01/02 15:04:05"),
					)
					slack.TeeMessage(message)
					remindSeconds = etc.Remove(remindSeconds, second)
				}
			}
		}
	}()

	state := <-ch
	sl.Stop()

	return state == STATE_CANCELED, nil
}
