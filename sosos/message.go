package sosos

import (
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/mpppk/sosos/chat"
	"github.com/mpppk/sosos/etc"
)

func generateActionMessages(cancelServerUrl string, suspendMinutes []int64, chatService chat.Service) string {
	message := "If you want to suspend or cancel this command, please click the following Link\n"
	for _, suspendMin := range suspendMinutes {
		suspendUrl := fmt.Sprintf("%s/suspend?suspendSec=%d",
			cancelServerUrl,
			suspendMin*60,
		)
		suspendLinkTitle := fmt.Sprintf("Suspend  %d minutes", suspendMin)
		message += chatService.GenerateLinkStr(suspendUrl, suspendLinkTitle) + "\n"
	}
	message += fmt.Sprintf("<%s/execute-now|Execute now>\n", cancelServerUrl)
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

func getCommandStartMessage(commands []string, sleepSec int64, executeTime time.Time, hostname string) string {
	return fmt.Sprintf("The command `%s` will be executed after %d seconds(%s) on `%s`\n",
		strings.Join(commands, " "),
		sleepSec,
		executeTime.Format("01/02 15:04:05"),
		hostname)
}

func getCommandExecutionStartMessage(commands []string) string {
	return fmt.Sprintf("Command `%s` execution is started!",
		strings.Join(commands, " "))
}

func getCommandSuspendMessage(sec int, executeTime time.Time) string {
	return fmt.Sprintf("Time to command execution has been suspended by %d seconds. (%s)",
		sec, executeTime.Format("01/02 15:04:05"),
	)
}

func getCommandRemindMessage(commands []string, sec int64, executeTime time.Time) string {
	return fmt.Sprintf("Remind: The command `%s` will be executed after %d seconds(%s)\n",
		strings.Join(commands, " "), sec, executeTime.Format("01/02 15:04:05"),
	)
}

func getCommandCancelMessage() string {
	return "Command is canceled!"
}

func getCommandFailedMessage(err error) string {
	return fmt.Sprintf("command failed:\n```\n%s\n```", err.Error())
}

func getCommandTerminateMessage() string {
	return "The command is terminated by SIGINT signal"
}
