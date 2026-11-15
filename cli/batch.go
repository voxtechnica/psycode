package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/spf13/cobra"
)

// BatchCommand is the command for managing batch operations.
type BatchCommand struct {
	apiClient  *anthropic.Client
	rootCmd    *cobra.Command
	baseCmd    *cobra.Command
	listCmd    *cobra.Command
	readCmd    *cobra.Command
	monitorCmd *cobra.Command
	cancelCmd  *cobra.Command
	deleteCmd  *cobra.Command
	raw        bool
}

// NewBatchCommand creates and initializes the batch commands.
func NewBatchCommand(apiClient *anthropic.Client, root *cobra.Command) *BatchCommand {
	// Base Command
	c := &BatchCommand{
		apiClient: apiClient,
		rootCmd:   root,
	}
	c.baseCmd = &cobra.Command{
		Use:   "batch",
		Short: "Manage batch operations",
		Long:  "Manage batch operations.",
	}
	c.baseCmd.PersistentFlags().BoolVarP(&c.raw, "raw", "r", false, "Raw JSON Response")
	c.rootCmd.AddCommand(c.baseCmd)

	// Read Command
	c.readCmd = &cobra.Command{
		Use:   "read <batchID> [batchID]...",
		Short: "Read specified batch operation(s)",
		Long:  "Read the metadata about one or more batch operations, specified by ID.",
		Args:  cobra.MinimumNArgs(1),
		RunE:  c.read,
	}
	c.baseCmd.AddCommand(c.readCmd)

	// Monitor Command
	c.monitorCmd = &cobra.Command{
		Use:   "monitor <batchID>",
		Short: "Monitor specified batch operation",
		Long:  "Monitor the progress of a batch operation, specified by ID.",
		Args:  cobra.ExactArgs(1),
		RunE:  c.monitor,
	}
	c.monitorCmd.Flags().IntP("wait", "w", 10, "Wait interval (seconds)")
	c.baseCmd.AddCommand(c.monitorCmd)

	// Cancel Command
	c.cancelCmd = &cobra.Command{
		Use:   "cancel <batchID> [batchID]...",
		Short: "Cancel specified batch operation(s)",
		Long:  "Cancel one or more batch operations, specified by ID.",
		Args:  cobra.MinimumNArgs(1),
		RunE:  c.cancel,
	}
	c.baseCmd.AddCommand(c.cancelCmd)

	// Delete Command
	c.deleteCmd = &cobra.Command{
		Use:   "delete <batchID> [batchID]...",
		Short: "Delete specified batch operation(s)",
		Long:  "Delete one or more batches, specified by ID. Cancel first!",
		Args:  cobra.MinimumNArgs(1),
		RunE:  c.delete,
	}
	c.baseCmd.AddCommand(c.deleteCmd)

	// List Command
	c.listCmd = &cobra.Command{
		Use:   "list",
		Short: "List batch operations",
		Long:  "List metadata of available batch operations.",
		RunE:  c.list,
	}
	c.listCmd.Flags().BoolP("all", "a", false, "Complete list")
	c.listCmd.Flags().Int64P("limit", "l", 20, "Limit")
	c.listCmd.Flags().StringP("offset", "o", "", "Offset: last ID received")
	c.listCmd.Flags().BoolP("verbose", "v", false, "Verbose JSON output")
	c.baseCmd.AddCommand(c.listCmd)

	return c
}

// list is the handler for the "batch list" command.
func (c *BatchCommand) list(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	all, _ := cmd.Flags().GetBool("all")
	limit, _ := cmd.Flags().GetInt64("limit")
	offset, _ := cmd.Flags().GetString("offset")
	verbose, _ := cmd.Flags().GetBool("verbose")
	parameters := anthropic.MessageBatchListParams{
		Limit: anthropic.Int(limit),
	}
	if len(offset) > 0 && !all {
		parameters.AfterID = anthropic.String(offset)
	}
	if !c.raw && !verbose {
		fmt.Println("BatchID\tCreatedAt\tStatus\tSucceeded\tErrored\tProcessing")
	}

	if all {
		// List all the batches
		iter := c.apiClient.Messages.Batches.ListAutoPaging(ctx, parameters)
		for iter.Next() {
			b := iter.Current()
			if verbose {
				// Display formatted JSON
				j, err := json.MarshalIndent(b, "", "  ")
				if err != nil {
					return fmt.Errorf("error marshalling Batch JSON: %w", err)
				}
				fmt.Println(string(j))
			} else {
				// Display a tabular list batches
				fmt.Printf("%s\t%s\t%s\t%d\t%d\t%d\n", b.ID, b.CreatedAt.Format(time.DateTime), b.ProcessingStatus,
					b.RequestCounts.Succeeded, b.RequestCounts.Errored, b.RequestCounts.Processing)
			}
		}
	} else {
		// List the selected page of batches
		page, err := c.apiClient.Messages.Batches.List(ctx, parameters)
		if err != nil {
			return err
		}
		if c.raw {
			// Display the raw JSON response
			fmt.Println(page.RawJSON())
		} else if verbose {
			// Display formatted JSON
			j, err := json.MarshalIndent(page, "", "  ")
			if err != nil {
				return fmt.Errorf("error marshalling Batch JSON: %w", err)
			}
			fmt.Println(string(j))
		} else {
			// Display a tabular list batches
			for _, b := range page.Data {
				fmt.Printf("%s\t%s\t%s\t%d\t%d\t%d\n", b.ID, b.CreatedAt.Format(time.DateTime), b.ProcessingStatus,
					b.RequestCounts.Succeeded, b.RequestCounts.Errored, b.RequestCounts.Processing)
			}
			if page.HasMore {
				fmt.Printf("More results available. Use --limit=%d --offset=%s to retrieve.\n", limit, page.LastID)
			}
		}
	}
	return nil
}

// read is the handler for the "batch read" command.
func (c *BatchCommand) read(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	for _, batchID := range args {
		// Read the specified batch job
		batch, e := c.apiClient.Messages.Batches.Get(ctx, batchID, anthropic.MessageBatchGetParams{})
		if e != nil {
			return fmt.Errorf("read batch %s: %w", batchID, e)
		}
		if c.raw {
			// Display the raw JSON response
			fmt.Println(batch.RawJSON())
		} else {
			// Display a formatted JSON response
			j, err := json.MarshalIndent(batch, "", "  ")
			if err != nil {
				return fmt.Errorf("error marshalling Batch JSON: %w", err)
			}
			fmt.Println(string(j))
		}
	}
	return nil
}

// monitor is the handler for the "batch monitor" command.
func (c *BatchCommand) monitor(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	wait, _ := cmd.Flags().GetInt("wait")
	batchID := args[0]
	fmt.Println("BatchID\tCreatedAt\tStatus\tSucceeded\tErrored\tProcessing")
	for {
		b, err := c.apiClient.Messages.Batches.Get(ctx, batchID, anthropic.MessageBatchGetParams{})
		if err != nil {
			return fmt.Errorf("read batch %s: %w", batchID, err)
		}
		fmt.Printf("%s\t%s\t%s\t%d\t%d\t%d\n", b.ID, b.CreatedAt.Format(time.DateTime), b.ProcessingStatus,
			b.RequestCounts.Succeeded, b.RequestCounts.Errored, b.RequestCounts.Processing)
		if b.ProcessingStatus == anthropic.MessageBatchProcessingStatusEnded {
			break
		}
		time.Sleep(time.Duration(wait) * time.Second)
	}
	return nil
}

// cancel is the handler for the "batch cancel" command.
func (c *BatchCommand) cancel(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	for _, batchID := range args {
		// Cancel the specified batch job
		batch, err := c.apiClient.Messages.Batches.Cancel(ctx, batchID, anthropic.MessageBatchCancelParams{})
		if err != nil {
			return err
		}
		if c.raw {
			// Display the raw JSON response
			fmt.Println(batch.RawJSON())
		} else {
			// Display a formatted JSON response
			j, err := json.MarshalIndent(batch, "", "  ")
			if err != nil {
				return fmt.Errorf("error marshalling Batch JSON: %w", err)
			}
			fmt.Println(string(j))
		}
	}
	return nil
}

// delete is the handler for the "batch delete" command.
func (c *BatchCommand) delete(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	for _, batchID := range args {
		// Delete the specified batch job
		b, err := c.apiClient.Messages.Batches.Delete(ctx, batchID, anthropic.MessageBatchDeleteParams{})
		if err != nil {
			return err
		}
		if c.raw {
			// Display the raw JSON response
			fmt.Println(b.RawJSON())
		} else {
			// Display a formatted JSON response
			j, err := json.MarshalIndent(b, "", "  ")
			if err != nil {
				return fmt.Errorf("error marshalling DeletedMessageBatch JSON: %w", err)
			}
			fmt.Println(string(j))
		}
	}
	return nil
}
