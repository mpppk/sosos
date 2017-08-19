package sosos

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/exec"
	"time"

	"github.com/hydrogen18/stoppableListener"
)

func logIfErrExist(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func Execute(commands []string, sleepSec int) {
	cmd := exec.Command(commands[0], commands[1:]...)
	stdout, err := cmd.StdoutPipe()
	logIfErrExist(err)
	waitWithCancelServer(sleepSec)
	logIfErrExist(cmd.Start())
	var person struct {
		Name string
		Age  int
	}
	logIfErrExist(json.NewDecoder(stdout).Decode(&person))
	logIfErrExist(cmd.Wait())
	fmt.Printf("%s is %d years old\n", person.Name, person.Age)
}

func Hello(rw http.ResponseWriter, req *http.Request) {
	rw.WriteHeader(http.StatusOK)
	fmt.Fprintf(rw, "Hello\n")
}

func waitWithCancelServer(sleepSec int) {
	l, err := net.Listen("tcp", ":3333")
	if err != nil {
		panic(err)
	}
	sl, err := stoppableListener.New(l)
	if err != nil {
		panic(err)
	}
	http.HandleFunc("/", Hello)
	s := http.Server{}

	go func() {
		s.Serve(sl)
	}()
	fmt.Printf("Serving HTTP\n")

	time.Sleep(time.Duration(sleepSec) * time.Second)

	fmt.Println("stopping server..")
	sl.Stop()
}
