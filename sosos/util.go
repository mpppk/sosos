package sosos

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/mpppk/sosos/etc"
)

func getCancelServerUrl(port int, insecureFlag bool) (string, error) {
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

func generateCancelAndSuspendMessage(cancelServerUrl string, suspendMinutes []int64) string {
	message := "If you want to suspend or cancel this command, please click the following Link\n"
	for _, suspendMin := range suspendMinutes {
		message += fmt.Sprintf("<%s/suspend?suspendSec=%d|Suspend  %d minutes>\n",
			cancelServerUrl,
			suspendMin*60,
			suspendMin)
	}
	message += fmt.Sprintf("<%s/cancel|Cancel>\n", cancelServerUrl)

	return message
}

func getScriptContentMessage(commands []string, extList []string) (string, bool, error) {
	for _, command := range commands {
		if etc.IsScript(command, extList) {
			fileBytes, err := ioutil.ReadFile(command)
			if err != nil {
				return "", false, err
			}
			return fmt.Sprintf("`%s` contents:\n```\n%s\n```\n", command, string(fileBytes)), true, nil
		}
	}
	return "", false, nil
}
