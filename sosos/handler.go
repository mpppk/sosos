package sosos

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
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
		s.SuspendSecCh <- suspendSec
	}()
}

type ExecuteNowHandler struct {
	ExecuteNowCh chan bool
}

func (c ExecuteNowHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	rw.WriteHeader(http.StatusOK)
	fmt.Fprintln(rw, "Command execution started!")
	go func() {
		c.ExecuteNowCh <- true
	}()
}
