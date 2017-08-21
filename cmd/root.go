package cmd

import (
	"fmt"
	"os"

	"strings"

	"log"

	"github.com/mpppk/sosos/etc"
	"github.com/mpppk/sosos/sosos"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var sleepSec int
var port int
var insecureFlag bool
var versionFlag bool
var argWebhook string

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

		if err := sosos.Execute(args, sleepSec, port, insecureFlag, webhookUrl); err != nil {
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

	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.sosos.yaml)")
	RootCmd.PersistentFlags().IntVarP(&sleepSec, "sleep", "s", 60*15, "sleep time(sec)")
	RootCmd.PersistentFlags().IntVarP(&port, "port", "p", 3333, "port of cancel server")
	RootCmd.PersistentFlags().BoolVarP(&insecureFlag, "insecure-server", "i", false, "Use http protocol for cancel server")
	RootCmd.PersistentFlags().BoolVar(&versionFlag, "version", false, "Print version")
	RootCmd.PersistentFlags().StringVarP(&argWebhook, "webhook", "w", "", "Webhook URL")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}

	viper.SetConfigName(".sosos")          // name of config file (without extension)
	viper.AddConfigPath(os.Getenv("HOME")) // adding home directory as first search path
	viper.AddConfigPath(os.Getenv("USERPROFILE"))
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
