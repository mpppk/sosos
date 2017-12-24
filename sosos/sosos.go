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

	"sync"

	"os/signal"
	"syscall"

	"io"

	"github.com/hydrogen18/stoppableListener"
	"github.com/mpppk/sosos/chat"
)

const (
	STATE_CANCELED = iota + 1
	STATE_SLEEP_FINISHED
)

type Executor struct {
	Commands     []string
	ch           chan int
	suspendSecCh chan int
	executeNowCh chan bool
	chatService  chat.Service
	timeKeeper   *TimeKeeper
	opt          *ExecutorOption
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
	SuspendMinutes      []int64
	RemindSeconds       []int64
	ScriptExtList       []string
}

func NewExecutor(commands []string, opt *ExecutorOption) *Executor {

	var chatAdapter chat.Service

	if chat.IsSlackWebhookUrl(opt.WebhookUrl) {
		chatAdapter = &chat.Slack{WebhookUrl: opt.WebhookUrl}
	} else {
		chatAdapter = &chat.Mattermost{Slack: &chat.Slack{WebhookUrl: opt.WebhookUrl}}
	}

	return &Executor{
		Commands:     commands,
		ch:           make(chan int),
		suspendSecCh: make(chan int),
		executeNowCh: make(chan bool),
		chatService:  chatAdapter,
		timeKeeper:   NewTimeKeeper(opt.SleepSec, opt.RemindSeconds, opt.SuspendMinutes),
		opt:          opt,
	}
}

func (e *Executor) ExecuteCommand() ([]string, error) {
	cmd := exec.Command(e.Commands[0], e.Commands[1:]...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
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

	return results, cmd.Wait()
}

func (e *Executor) Execute() error {
	isCanceled := false
	if e.opt.SleepSec != 0 {
		cancelServerUrl, err := getCancelServerUrl(e.opt.Port, e.opt.InsecureFlag)
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
			e.timeKeeper.sleepSec,
			e.timeKeeper.commandExecuteTime.Format("01/02 15:04:05"),
			u.Hostname())

		if !e.opt.NoScriptContentFlag {
			contentMessage, ok, err := getScriptContentMessage(e.Commands, e.opt.ScriptExtList)
			if ok {
				message += contentMessage
			} else if err != nil {
				return err
			}
		}

		if !e.opt.NoCancelLinkFlag {
			message += generateActionMessages(cancelServerUrl, e.timeKeeper.suspendMinutes, e.chatService)
		}

		if _, err := e.teeMessageWithCode(message); err != nil {
			return err
		}

		isCanceled, err = e.waitWithCancelServer()
		if err != nil {
			return err
		}
	}

	if isCanceled {
		if _, err := e.teeMessageWithCode("Command is canceled!"); err != nil {
			return err
		}
	}

	message := fmt.Sprintf("Command `%s` execution is started!", strings.Join(e.Commands, " "))

	if !e.opt.NoScriptContentFlag {
		contentMessage, ok, _ := getScriptContentMessage(e.Commands, e.opt.ScriptExtList)
		if ok {
			message += "\n" + contentMessage
		}
	}

	if _, err := e.teeMessageWithCode(message); err != nil {
		return err
	}
	results, cmdErr := e.ExecuteCommand()
	if !e.opt.NoResultFlag {
		var message string
		if cmdErr != nil {
			message = fmt.Sprintf("command failed:\n```\n%s\n```", cmdErr.Error())
		} else {
			message = fmt.Sprintf("result:\n```\n%s\n```", strings.Join(results, "\n"))
		}

		if _, err := e.chatService.PostMessage(message); err != nil {
			return err
		}
	} else {
		e.chatService.TeeMessage("finish!")
	}

	return cmdErr
}

func (e *Executor) tick() {
	ticker := time.NewTicker(500 * time.Millisecond)

	sigintCh := make(chan os.Signal, 1)
	signal.Notify(sigintCh, syscall.SIGINT)

	for range ticker.C {
		select {
		case state := <-e.ch:
			switch state {
			case STATE_CANCELED:
				return
			}
		case suspendSec := <-e.suspendSecCh:
			e.timeKeeper.SuspendCommandExecuteTime(suspendSec)
			message := fmt.Sprintf("Time to command execution has been suspended by %d seconds. (%s)",
				suspendSec,
				e.timeKeeper.commandExecuteTime.Format("01/02 15:04:05"),
			)
			e.chatService.TeeMessage(message)
		case <-e.executeNowCh:
			e.ch <- STATE_SLEEP_FINISHED
			return
		case <-sigintCh:
			e.chatService.TeeMessage("The command is terminated by SIGINT signal")
			os.Exit(0)
		default:
		}

		e.timeKeeper.UpdateRemainSec()
		if e.timeKeeper.remainSec <= 0 {
			e.ch <- STATE_SLEEP_FINISHED
			return
		}

		if remainSec, ok := e.timeKeeper.GetNewRemind(); ok {
			message := fmt.Sprintf("Remind: The command `%s` will be executed after %d seconds(%s)\n",
				strings.Join(e.Commands, " "),
				remainSec,
				e.timeKeeper.commandExecuteTime.Format("01/02 15:04:05"),
			)

			if !e.opt.NoScriptContentFlag {
				contentMessage, ok, _ := getScriptContentMessage(e.Commands, e.opt.ScriptExtList)
				if ok {
					message += contentMessage
				}
			}
			e.chatService.TeeMessage(message)
		}
	}
}

func (e *Executor) createStoppableCancelServer(cancelHandler http.Handler, suspendHandler http.Handler, executeNowHandler http.Handler) (*http.Server, *stoppableListener.StoppableListener, error) {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", e.opt.Port))
	if err != nil {
		return nil, nil, err
	}
	sl, err := stoppableListener.New(l)
	if err != nil {
		return nil, nil, err
	}

	http.Handle("/cancel", cancelHandler)
	http.Handle("/suspend", suspendHandler)
	http.Handle("/execute-now", executeNowHandler)
	return &http.Server{}, sl, nil
}

func (e *Executor) waitWithCancelServer() (bool, error) {
	s, sl, err := e.createStoppableCancelServer(CancelHandler{e.ch}, SuspendHandler{e.suspendSecCh}, ExecuteNowHandler{e.executeNowCh})
	if err != nil {
		return false, err
	}

	go s.Serve(sl)
	go e.tick()
	state := <-e.ch
	sl.Stop()

	return state == STATE_CANCELED, nil
}

func (e *Executor) teeMessageWithCode(message string) (*http.Response, error) {
	res, err := e.chatService.TeeMessage(message)
	if err != nil {
		return nil, err
	}
	fmt.Println("http response " + res.Status)
	return res, err
}
