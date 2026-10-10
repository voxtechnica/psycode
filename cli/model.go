package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/packages/param"
	"github.com/spf13/cobra"
)

// ModelCommand is the command for managing models.
type ModelCommand struct {
	apiClient *anthropic.Client
	rootCmd   *cobra.Command
	baseCmd   *cobra.Command
	listCmd   *cobra.Command
	readCmd   *cobra.Command
	raw       bool
}

// NewModelCommand creates and initializes the model commands.
func NewModelCommand(apiClient *anthropic.Client, root *cobra.Command) *ModelCommand {
	// Base Command
	c := &ModelCommand{
		apiClient: apiClient,
		rootCmd:   root,
	}
	c.baseCmd = &cobra.Command{
		Use:   "model",
		Short: "Explore models",
		Long:  "Explore available models",
	}
	c.baseCmd.PersistentFlags().BoolVarP(&c.raw, "raw", "r", false, "Raw JSON Response?")
	c.rootCmd.AddCommand(c.baseCmd)

	// List Command
	c.listCmd = &cobra.Command{
		Use:   "list",
		Short: "List models",
		Long:  "List available models",
		RunE:  c.list,
	}
	c.listCmd.Flags().Int64P("limit", "n", 20, "Page size limit")
	c.listCmd.Flags().StringP("after", "a", "", "Last Model ID received")
	c.baseCmd.AddCommand(c.listCmd)

	// Read Command
	c.readCmd = &cobra.Command{
		Use:   "read <modelID> [modelID]...",
		Short: "Read specified model(s)",
		Long:  "Read the details about one or more models, specified by ID.",
		Args:  cobra.MinimumNArgs(1),
		RunE:  c.read,
	}
	// c.readCmd.Flags().BoolP("verbose", "v", false, "Verbose? (full JSON)")
	c.baseCmd.AddCommand(c.readCmd)

	return c
}

// list the available models.
func (c *ModelCommand) list(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	limit, _ := cmd.Flags().GetInt64("limit")
	parameters := anthropic.ModelListParams{
		Limit: param.Opt[int64]{Value: limit},
	}
	after, _ := cmd.Flags().GetString("after")
	if len(after) > 0 {
		parameters.AfterID = param.Opt[string]{Value: after}
	}

	// Retrieve the page:
	page, err := c.apiClient.Models.List(ctx, parameters)
	if err != nil {
		return err
	}

	if c.raw {
		// Display the raw JSON response
		fmt.Println(page.RawJSON())
	} else {
		// Display a list of model IDs and creation timestamps
		for _, model := range page.Data {
			fmt.Println(model.CreatedAt, model.ID)
		}
	}
	return nil
}

// read the metadata for the specified model(s).
func (c *ModelCommand) read(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	// verbose, _ := cmd.Flags().GetBool("verbose")

	// Retrieve the model(s):
	for _, modelID := range args {
		model, err := c.apiClient.Models.Get(ctx, modelID, anthropic.ModelGetParams{})
		if err != nil {
			return err
		}

		if c.raw {
			fmt.Println(model.RawJSON())
		} else {
			j, err := json.MarshalIndent(model, "", "  ")
			if err != nil {
				return fmt.Errorf("error marshalling JSON model: %w", err)
			}
			fmt.Println(string(j))
		}
	}
	return nil
}
