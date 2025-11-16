package main

import (
	"fmt"
	"os"

	"github.com/censys/scan-takehome/cmd/consumer"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "mini-scan",
	Short: "Mini scan tool for scanning and consuming scan data",
	Long:  "A CLI tool for scanning services and consuming scan results via GCP PubSub",
}

func init() {
	rootCmd.AddCommand(consumer.NewConsumerCmd())
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
