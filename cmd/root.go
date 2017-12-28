package cmd

import (
	"fmt"
	"os"

	"strings"

	"log"

	"path/filepath"

	"strconv"

	"io/ioutil"

	"github.com/mitchellh/go-homedir"
	"github.com/mpppk/sosos/etc"
	"github.com/mpppk/sosos/sosos"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

var cfgFile string
var sleepSec int64
var port int
var insecureFlag bool
var versionFlag bool
var noResultFlag bool
var noCancelLinkFlag bool
var noScriptContentFlag bool
var argWebhook string
var message string

const version = "0.8.1"

var RootCmd = &cobra.Command{
	Use:   "sosos",
	Short: "delay & notify tool",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if versionFlag {
			fmt.Println(version)
			os.Exit(0)
		}

		config, err := etc.LoadConfigFromFile()
		if err != nil {
			fmt.Println(err)
		}

		var webhookUrl string
		if strings.Contains(argWebhook, "http") {
			webhookUrl = argWebhook
		} else {
			if webhook, ok := config.FindWebhook(argWebhook); ok {
				webhookUrl = webhook.Url
			} else {
				log.Fatal("There is no webhook called ", argWebhook, " in the config file")
			}
		}

		suspendMinutes := []int64{}
		for _, suspendMinuteStr := range strings.Split(viper.GetString("suspendMinutes"), ",") {
			suspendMinute, err := strconv.Atoi(suspendMinuteStr)
			if err != nil {
				errors.New("suspend-minutes: invalid argument")
			}
			suspendMinutes = append(suspendMinutes, int64(suspendMinute))
		}

		remindSeconds := []int64{}
		for _, remindSecondStr := range strings.Split(viper.GetString("remindSeconds"), ",") {
			remindSecond, err := strconv.Atoi(remindSecondStr)
			if err != nil {
				errors.New("remind-seconds: invalid argument")
			}
			remindSeconds = append(remindSeconds, int64(remindSecond))
		}

		scriptExtList := []string{}
		for _, scriptExt := range strings.Split(viper.GetString("scriptExt"), ",") {
			scriptExtList = append(scriptExtList, scriptExt)
		}

		executor := sosos.NewExecutor(args, &sosos.ExecutorOption{
			SleepSec:            sleepSec,
			Port:                port,
			WebhookUrl:          webhookUrl,
			InsecureFlag:        insecureFlag,
			NoResultFlag:        noResultFlag,
			NoCancelLinkFlag:    noCancelLinkFlag,
			NoScriptContentFlag: noScriptContentFlag,
			SuspendMinutes:      suspendMinutes,
			RemindSeconds:       remindSeconds,
			CustomMessage:       message,
			ScriptExtList:       scriptExtList,
		})
		if err := executor.Execute(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
	Args: MinimumNArgsWithoutVersionOption(1),
}

func MinimumNArgsWithoutVersionOption(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) < n && !versionFlag {
			return fmt.Errorf("requires at least %d arg(s), only received %d", n, len(args))
		}
		return nil
	}
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Config file (default is $HOME/.config/sosos/.sosos.yaml)")
	RootCmd.PersistentFlags().Int64VarP(&sleepSec, "sleep", "s", 60*15, "Sleep time(sec)")
	viper.BindPFlag("sleep", RootCmd.PersistentFlags().Lookup("sleep"))
	RootCmd.PersistentFlags().IntVarP(&port, "port", "p", 50505, "Port of cancel server")
	viper.BindPFlag("port", RootCmd.PersistentFlags().Lookup("port"))
	RootCmd.PersistentFlags().BoolVarP(&insecureFlag, "insecure-server", "i", false, "Use http protocol for cancel server")
	RootCmd.PersistentFlags().BoolVar(&versionFlag, "version", false, "Print version")
	RootCmd.PersistentFlags().StringVarP(&argWebhook, "webhook", "w", "", "Webhook URL")
	RootCmd.PersistentFlags().BoolVar(&noResultFlag, "no-result", false, "Not display results of command")
	RootCmd.PersistentFlags().BoolVar(&noCancelLinkFlag, "no-cancel-link", false, "Not display cancel link")
	RootCmd.PersistentFlags().BoolVar(&noScriptContentFlag, "no-script-content", false, "Not display script content")
	RootCmd.PersistentFlags().StringVarP(&message, "message", "m", "", "Send custom message to chat")
	viper.BindPFlag("message", RootCmd.PersistentFlags().Lookup("message"))
	RootCmd.PersistentFlags().String("suspend-minutes", "5,20,60", "List of suspend minutes link(comma separated)")
	viper.BindPFlag("suspendMinutes", RootCmd.PersistentFlags().Lookup("suspend-minutes"))
	RootCmd.PersistentFlags().String("remind-seconds", "60,300", "List of remind seconds link(comma separated)")
	viper.BindPFlag("remindSeconds", RootCmd.PersistentFlags().Lookup("remind-seconds"))
	RootCmd.PersistentFlags().String("script-ext", "sh,bat,ps1,rb,py,pl,php", "List of script ext for show contents(comma separated)")
	viper.BindPFlag("scriptExt", RootCmd.PersistentFlags().Lookup("script-ext"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}

	viper.SetConfigName(".sosos") // name of config file (without extension)

	homeDir, err := homedir.Dir()
	if err != nil {
		log.Fatal(err)
	}
	configPath := filepath.Join(homeDir, ".config", "sosos")
	viper.AddConfigPath(configPath) // adding home directory as first search path
	viper.AutomaticEnv()            // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	} else {
		config := &etc.Config{
			Webhooks: []etc.Webhook{
				{
					Name: "default",
					Url:  "https://hooks.slack.com/services/your/url",
				},
			},
			Sleep:          900,
			Port:           50505,
			RemindSeconds:  "60,300",
			SuspendMinutes: "5,20,60",
			ScriptExt:      "sh,bat,ps1,rb,py,pl,php",
			Message:        "",
		}

		content, err := yaml.Marshal(config)
		if err != nil {
			log.Fatal(err)
		}
		err = ioutil.WriteFile(filepath.Join(configPath, ".sosos.yaml"), content, 0777)
		if err != nil {
			log.Fatal(err)
		}
	}
}
