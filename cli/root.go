package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

// RootCommand is the root command for the application.
type RootCommand struct {
	apiKey      string
	workspaceID string
	apiClient   *anthropic.Client
	rootCmd     *cobra.Command
	aboutCmd    *cobra.Command
	docCmd      *cobra.Command
	limerickCmd *cobra.Command
	// batchCmd  *BatchCommand
	// fileCmd   *FileCommand
	modelCmd *ModelCommand
	// textCmd   *TextCommand
}

// NewRootCommand creates and initializes the root command and all its subcommands.
func NewRootCommand(apiKey string, workspaceID string, apiClient *anthropic.Client) *RootCommand {
	// Root Command (application name)
	c := &RootCommand{
		apiKey:      apiKey,
		workspaceID: workspaceID,
		apiClient:   apiClient,
	}
	c.rootCmd = &cobra.Command{
		Use:     "psycode",
		Short:   "psycode: LLM Text Analysis Command Line Tool",
		Long:    "psycode is a command-line tool for analyzing text in psychology research projects, using a large language model.",
		Version: "0.1.0",
	}

	// About Command
	c.aboutCmd = &cobra.Command{
		Use:   "about",
		Short: "Print application information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(cmd.Root().Long)
			fmt.Println("Version:", cmd.Root().Version)
			fmt.Println("Workspace ID:", c.workspaceID)
			fmt.Println("API Key:", c.apiKey)
		},
	}
	c.rootCmd.AddCommand(c.aboutCmd)

	// Documentation Command:
	c.docCmd = &cobra.Command{
		Use:   "docs",
		Short: "Generate psycode markdown documentation",
		Long:  "Generate psycode markdown documentation in the ./docs directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			// If the docs directory doesn't exist, create it
			if _, err := os.Stat("./docs"); os.IsNotExist(err) {
				err = os.Mkdir("./docs", 0755)
				if err != nil {
					return err
				}
			}
			// Generate the documentation
			return doc.GenMarkdownTree(c.rootCmd, "./docs")
		},
	}
	c.rootCmd.AddCommand(c.docCmd)

	// Limerick Command:
	c.limerickCmd = &cobra.Command{
		Use:   "limerick",
		Short: "Test LLM API access by generating a limerick",
		Long:  "Test LLM API access by generating a limerick about psychology research.",
		RunE: func(cmd *cobra.Command, args []string) error {
			message, err := apiClient.Messages.New(context.Background(), anthropic.MessageNewParams{
				MaxTokens: 1024,
				Messages: []anthropic.MessageParam{
					anthropic.NewUserMessage(anthropic.NewTextBlock("Compose a limerick about psychology research.")),
				},
				Model: anthropic.ModelClaudeSonnet5_5,
			})
			for _, block := range message.Content {
				if textBlock, ok := block.AsAny().(anthropic.TextBlock); ok {
					fmt.Println(textBlock.Text)
				}
			}
			return err
		},
	}
	c.rootCmd.AddCommand(c.limerickCmd)

	// Other Commands
	// c.batchCmd = NewBatchCommand(apiClient, c.rootCmd)
	// c.fileCmd = NewFileCommand(apiClient, c.rootCmd)
	c.modelCmd = NewModelCommand(apiClient, c.rootCmd)
	// c.textCmd = NewTextCommand(apiClient, c.rootCmd)

	return c
}

// Execute executes the root command.
func (c *RootCommand) Execute() error {
	return c.rootCmd.Execute()
}
