package sosos

import (
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"time"

	"os"

	"strings"

	"net/url"

	"bufio"

	"io/ioutil"

	"sync"

	"os/signal"
	"syscall"

	"io"

	"github.com/hydrogen18/stoppableListener"
	"github.com/mpppk/sosos/chat"
	"github.com/mpppk/sosos/etc"
)

const (
	STATE_CANCELED = iota + 1
	STATE_SLEEP_FINISHED
)

type Executor struct {
	Commands      []string
	remindSeconds []int64
	slack         *chat.Slack
	opt           *ExecutorOption
}

type ExecutorOption struct {
	SleepSec            int64
	Port                int
	WebhookUrl          string
	InsecureFlag        bool
	NoResultFlag        bool
	NoCancelLinkFlag    bool
	NoScriptContentFlag bool
	CustomMessage       string
}

func NewExecutor(commands []string, opt *ExecutorOption) *Executor {
	slack := &chat.Slack{WebhookUrl: opt.WebhookUrl}
	return &Executor{
		Commands:      commands,
		remindSeconds: []int64{60, 300},
		slack:         slack,
		opt:           opt,
	}
}

func generateCancelAndSuspendMessage(cancelServerUrl string, suspendMins []int64) string {
	message := "If you want to suspend or cancel this command, please click the following Link\n"
	message += fmt.Sprintf("[Cancel](%s/cancel)\n", cancelServerUrl)
	for _, suspendMin := range suspendMins {
		message += fmt.Sprintf("[Suspend  %d minutes](%s/suspend?suspendSec=%d)\n",
			suspendMin,
			cancelServerUrl,
			suspendMin*60)
	}

	return message
}

func getScriptContentMessage(commands []string) (string, bool, error) {
	for _, command := range commands {
		if etc.IsScript(command) {
			fileBytes, err := ioutil.ReadFile(command)
			if err != nil {
				return "", false, err
			}
			return fmt.Sprintf("`%s` contents:\n```\n%s\n```\n", command, string(fileBytes)), true, nil
		}
	}
	return "", false, nil
}

func (e *Executor) ExecuteCommand() error {
	var cmdErr error
	if _, err := e.teeMessageWithCode("Command execution is started!"); err != nil {
		return err
	}

	cmd := exec.Command(e.Commands[0], e.Commands[1:]...)
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

	printByScanner := func(reader io.Reader) {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			text := scanner.Text()
			fmt.Println(text)
			resultCh <- text
		}
		wg.Done()
	}

	go printByScanner(stdout)
	go printByScanner(stderr)

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

	if !e.opt.NoResultFlag {
		if _, err := e.teeMessageWithCode(fmt.Sprintf("result:\n```\n%s\n```", strings.Join(results, "\n"))); err != nil {
			return err
		}
	} else {
		e.slack.TeeMessage("finish!")
	}
	return cmdErr
}

func (e *Executor) Execute() error {
	suspendSecCh := make(chan int)
	cancelServerUrl, err := e.getCancelServerUrl()
	if err != nil {
		return err
	}

	u, err := url.Parse(cancelServerUrl)
	if err != nil {
		return err
	}

	message := e.opt.CustomMessage
	if message != "" {
		message += "\n"
	}
	message += fmt.Sprintf("The command `%s` will be executed after %d seconds(%s) on `%s`\n",
		strings.Join(e.Commands, " "),
		e.opt.SleepSec,
		time.Now().Add(time.Duration(e.opt.SleepSec)*time.Second).Format("01/02 15:04:05"),
		u.Hostname())

	if !e.opt.NoScriptContentFlag {
		contentMessage, ok, err := getScriptContentMessage(e.Commands)
		if ok {
			message += contentMessage
		} else if err != nil {
			return err
		}
	}

	if !e.opt.NoCancelLinkFlag {
		message += generateCancelAndSuspendMessage(cancelServerUrl, []int64{5, 20, 60})
	}

	if _, err := e.teeMessageWithCode(message); err != nil {
		return err
	}

	isCanceled, err := e.waitWithCancelServer(suspendSecCh)
	if err != nil {
		return err
	}

	if !isCanceled {
		if err := e.ExecuteCommand(); err != nil {
			return err
		}
	} else {
		if _, err := e.teeMessageWithCode("Command is canceled!"); err != nil {
			return err
		}
	}
	return nil
}

func (e *Executor) waitWithCancelServer(suspendSecCh chan int) (bool, error) {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", e.opt.Port))
	if err != nil {
		return false, err
	}
	sl, err := stoppableListener.New(l)
	if err != nil {
		return false, err
	}

	ch := make(chan int)
	http.Handle("/cancel", CancelHandler{ch})
	http.Handle("/suspend", SuspendHandler{suspendSecCh})
	s := http.Server{}

	go func() {
		s.Serve(sl)
	}()

	go func() {
		executeTime := time.Now().Add(time.Duration(e.opt.SleepSec) * time.Second)
		ticker := time.NewTicker(500 * time.Millisecond)

		for _, second := range e.remindSeconds {
			if second > e.opt.SleepSec {
				e.remindSeconds = etc.Remove(e.remindSeconds, second)
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
				e.slack.TeeMessage(message)
			case <-sigintCh:
				e.slack.TeeMessage("The command is terminated by SIGINT signal")
				os.Exit(0)
			default:
			}

			remainSec := executeTime.Unix() - time.Now().Unix()
			if remainSec <= 0 {
				ch <- STATE_SLEEP_FINISHED
				return
			}

			for _, second := range e.remindSeconds {
				if second > remainSec {
					message := fmt.Sprintf("Remind: The command will be executed after %d seconds(%s)\n",
						remainSec,
						executeTime.Format("01/02 15:04:05"),
					)
					e.slack.TeeMessage(message)
					e.remindSeconds = etc.Remove(e.remindSeconds, second)
				}
			}
		}
	}()

	state := <-ch
	sl.Stop()

	return state == STATE_CANCELED, nil
}

func (e *Executor) getCancelServerUrl() (string, error) {
	protocol := "http"
	if !e.opt.InsecureFlag {
		protocol = protocol + "s"
	}

	hostname, err := os.Hostname()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s://%s:%d", protocol, hostname, e.opt.Port), nil
}

func (e *Executor) teeMessageWithCode(message string) (*http.Response, error) {
	res, err := e.slack.TeeMessage(message)
	if err != nil {
		return nil, err
	}
	fmt.Println("http response " + res.Status)
	return res, err
}
