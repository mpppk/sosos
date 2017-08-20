package cmd

import (
	"fmt"
	"os"

	"github.com/mpppk/sosos/sosos"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var sleepSec int
var port int
var insecureFlag bool

var RootCmd = &cobra.Command{
	Use:   "sosos",
	Short: "delay & notify tool",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if err := sosos.Execute(args, sleepSec, port, insecureFlag); err != nil {
			fmt.Println(err)
		}
	},
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
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}

	viper.SetConfigName(".sosos")          // name of config file (without extension)
	viper.AddConfigPath(os.Getenv("HOME")) // adding home directory as first search path
	viper.AutomaticEnv()                   // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
