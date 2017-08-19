package sosos

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"time"
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
	time.Sleep(time.Duration(sleepSec) * time.Second)
	logIfErrExist(cmd.Start())
	var person struct {
		Name string
		Age  int
	}
	logIfErrExist(json.NewDecoder(stdout).Decode(&person))
	logIfErrExist(cmd.Wait())
	fmt.Printf("%s is %d years old\n", person.Name, person.Age)
}
