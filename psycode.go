package main

import (
	"fmt"
	"os"
	"psycode/cli"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/spf13/viper"
)

func main() {
	var apiKey string
	var workspaceID string
	var ok bool

	// Application configuration
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	err := viper.ReadInConfig()
	if err == nil {
		// Use .env file values
		apiKey = viper.GetString("ANTHROPIC_API_KEY")
		workspaceID = viper.GetString("ANTHROPIC_WORKSPACE")
	} else {
		// Look for values in environment variables
		apiKey, ok = os.LookupEnv("ANTHROPIC_API_KEY")
		if !ok {
			fmt.Println("Missing: ANTHROPIC_API_KEY")
		}
		workspaceID, ok = os.LookupEnv("ANTHROPIC_WORKSPACE")
		if !ok {
			fmt.Println("Missing: ANTHROPIC_WORKSPACE")
		}
	}

	// Initialize the API client
	apiClient := anthropic.NewClient(
		option.WithAPIKey(apiKey),
		option.WithHeader("anthropic-workspace-id", workspaceID),
	)

	// Initialize the Command Line Interface:
	rootCmd := cli.NewRootCommand(apiKey, workspaceID, &apiClient)

	// Execute the specified command:
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
