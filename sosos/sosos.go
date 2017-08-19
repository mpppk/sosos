package sosos

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
)

func logIfErrExist(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func Execute() {
	cmd := exec.Command("echo", "-n", `{"Name": "Bob", "Age": 32}`)
	stdout, err := cmd.StdoutPipe()
	logIfErrExist(err)
	logIfErrExist(cmd.Start())
	var person struct {
		Name string
		Age  int
	}
	logIfErrExist(json.NewDecoder(stdout).Decode(&person))
	logIfErrExist(cmd.Wait())
	fmt.Printf("%s is %d years old\n", person.Name, person.Age)
}
