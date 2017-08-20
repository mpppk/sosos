package sosos

import (
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"time"

	"github.com/hydrogen18/stoppableListener"
)

const (
	STATE_CANCELED = iota + 1
	STATE_SLEEP_FINISHED
)

type CancelServer struct {
	Ch chan int
}

func (c CancelServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	rw.WriteHeader(http.StatusOK)
	fmt.Fprintf(rw, "Cancel request is accepted")
	go func() {
		c.Ch <- STATE_CANCELED
	}()
}

func Execute(commands []string, sleepSec int) error {
	isCanceled := waitWithCancelServer(sleepSec)
	if !isCanceled {
		fmt.Println("Start command execution")
		out, _ := exec.Command(commands[0], commands[1:]...).CombinedOutput()
		fmt.Println("---- command output ----")
		fmt.Println(string(out))
		fmt.Println("---- command output ----")
	} else {
		fmt.Println("command is canceled")
	}
	return nil
}

func waitWithCancelServer(sleepSec int) bool {
	l, err := net.Listen("tcp", ":3333")
	if err != nil {
		panic(err)
	}
	sl, err := stoppableListener.New(l)
	if err != nil {
		panic(err)
	}

	ch := make(chan int)
	http.Handle("/cancel", CancelServer{ch})
	s := http.Server{}

	go func() {
		s.Serve(sl)
	}()
	fmt.Println("Cancel URL is http://localhost:3333/cancel")

	go func() {
		time.Sleep(time.Duration(sleepSec) * time.Second)
		ch <- STATE_SLEEP_FINISHED
	}()

	state := <-ch
	sl.Stop()

	return state == STATE_CANCELED
}
