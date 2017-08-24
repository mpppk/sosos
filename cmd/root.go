package cmd

import (
	"fmt"
	"os"

	"strings"

	"log"

	"path/filepath"

	"github.com/mitchellh/go-homedir"
	"github.com/mpppk/sosos/etc"
	"github.com/mpppk/sosos/sosos"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

var RootCmd = &cobra.Command{
	Use:   "sosos",
	Short: "delay & notify tool",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if versionFlag {
			fmt.Println("0.0.1")
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
				fmt.Println(webhook.Url)
				webhookUrl = webhook.Url
			} else {
				log.Fatal("There is no webhook called ", argWebhook, " in the config file")
			}
		}

		executor := sosos.NewExecutor(args, &sosos.ExecutorOption{
			SleepSec:            sleepSec,
			Port:                port,
			WebhookUrl:          webhookUrl,
			InsecureFlag:        insecureFlag,
			NoResultFlag:        noResultFlag,
			NoCancelLinkFlag:    noCancelLinkFlag,
			NoScriptContentFlag: noScriptContentFlag,
			CustomMessage:       message,
		})
		if err := executor.Execute(); err != nil {
			fmt.Println(err)
		}
	},
	Args: cobra.MinimumNArgs(1),
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Config file (default is $HOME/.sosos.yaml)")
	RootCmd.PersistentFlags().Int64VarP(&sleepSec, "sleep", "s", 60*15, "Sleep time(sec)")
	RootCmd.PersistentFlags().IntVarP(&port, "port", "p", 3333, "Port of cancel server")
	RootCmd.PersistentFlags().BoolVarP(&insecureFlag, "insecure-server", "i", false, "Use http protocol for cancel server")
	RootCmd.PersistentFlags().BoolVar(&versionFlag, "version", false, "Print version")
	RootCmd.PersistentFlags().StringVarP(&argWebhook, "webhook", "w", "", "Webhook URL")
	RootCmd.PersistentFlags().BoolVar(&noResultFlag, "no-result", false, "Not display results of command")
	RootCmd.PersistentFlags().BoolVar(&noCancelLinkFlag, "no-cancel-link", false, "Not display cancel link")
	RootCmd.PersistentFlags().BoolVar(&noScriptContentFlag, "no-script-content", false, "Not display script content")
	RootCmd.PersistentFlags().StringVarP(&message, "message", "m", "", "Send custom message to chat")
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
	viper.AddConfigPath(filepath.Join(homeDir, ".config", "sosos")) // adding home directory as first search path
	viper.AutomaticEnv()                                            // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
