package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"psycode/psy"
	"strings"
	"sync"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/spf13/cobra"
	"github.com/voxtechnica/tuid-go"
)

type JsonCommand struct {
	apiClient   *anthropic.Client
	rootCmd     *cobra.Command
	baseCmd     *cobra.Command
	promptCmd   *cobra.Command
	randomCmd   *cobra.Command
	parallelCmd *cobra.Command
	batchCmd    *cobra.Command
	resultsCmd  *cobra.Command

	model     string
	maxTokens int64
}

// NewJsonCommand creates a new instance of the JsonCommand.
func NewJsonCommand(apiClient *anthropic.Client, root *cobra.Command) *JsonCommand {
	// Base Command
	c := &JsonCommand{
		apiClient: apiClient,
		rootCmd:   root,
	}
	c.baseCmd = &cobra.Command{
		Use:   "json",
		Short: "Generate structured output",
		Long:  "Generate structured output (JSON) from a specified prompt.",
	}
	c.baseCmd.PersistentFlags().StringVarP(&c.model, "model", "m", "claude-opus-5-5", "Model ID")
	c.baseCmd.PersistentFlags().Int64VarP(&c.maxTokens, "max-tokens", "t", 5120, "Max tokens to generate")
	c.rootCmd.AddCommand(c.baseCmd)

	// Prompt Command
	// Example: psycode json prompt data/prompt.txt data/system.txt -m claude-sonnet-5-5 -t 2048
	c.promptCmd = &cobra.Command{
		Use:   "prompt <promptFile> [systemFile]",
		Short: "Generate JSON from a test prompt",
		Long:  "Generate JSON from a test prompt provided in a specified file.",
		Args:  cobra.MinimumNArgs(1),
		RunE:  c.prompt,
	}
	c.promptCmd.Flags().BoolP("verbose", "v", false, "Verbose output?")
	c.baseCmd.AddCommand(c.promptCmd)

	// Random Command
	// Example: psycode json random data/answers.csv data/prompt.txt data/system.txt -A pid=749
	c.randomCmd = &cobra.Command{
		Use:   "random <answerFile> <promptFile> [systemFile]",
		Short: "Generate JSON for a random answer",
		Long:  "Generate JSON for a random answer from a specified file.",
		Args:  cobra.MinimumNArgs(2),
		RunE:  c.random,
	}
	c.randomCmd.Flags().StringP("answer-id", "A", "", "Answer ID (optional, name=value)")
	c.randomCmd.Flags().BoolP("verbose", "v", false, "Verbose output?")
	c.baseCmd.AddCommand(c.randomCmd)

	// Parallel Command
	// Example: psycode json parallel data/results.csv data/answers.csv data/prompt.txt data/system.txt
	c.parallelCmd = &cobra.Command{
		Use:   "parallel <outputFile> <answerFile> <promptFile> [systemFile]",
		Short: "Generate JSON for all answers in parallel",
		Long:  "Generate JSON for all answers using concurrent client-side requests.",
		Args:  cobra.MinimumNArgs(3),
		RunE:  c.parallel,
	}
	c.parallelCmd.Flags().IntP("batch-size", "b", 20, "Concurrent request batch size")
	c.baseCmd.AddCommand(c.parallelCmd)

	// Batch Command
	// Example: psycode json batch data/results.csv data/answers.csv data/prompt.txt data/system.txt -m claude-sonnet-5-5 -w 10
	c.batchCmd = &cobra.Command{
		Use:   "batch <outputFile> <answerFile> <promptFile> [systemFile] ",
		Short: "Generate JSON for all answers as a batch",
		Long:  "Generate JSON for all answers using a server-side asynchronous batch.",
		Args:  cobra.MinimumNArgs(3),
		RunE:  c.batchCreate,
	}
	c.batchCmd.Flags().IntP("wait", "w", 0, "Wait for results? Polling interval in seconds (recommend 10)")
	c.baseCmd.AddCommand(c.batchCmd)

	// Results Command
	// Example: psycode json results msgbatch_xxxxxxxxxxxxxxxxxxxxxxxx data/scores.csv
	c.resultsCmd = &cobra.Command{
		Use:   "results <batchID> <outputFile>",
		Short: "Process batch results",
		Long:  "Process the results of a completed batch operation.",
		Args:  cobra.ExactArgs(2),
		RunE:  c.batchResults,
	}
	c.baseCmd.AddCommand(c.resultsCmd)

	return c
}

// prompt handles the "json prompt" command to generate structured output from a prompt file.
func (c *JsonCommand) prompt(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	verbose, _ := cmd.Flags().GetBool("verbose")
	promptFile := args[0]
	systemFile := ""
	if len(args) > 1 {
		systemFile = args[1]
	}

	// Read the prompt and system files:
	prompt, err := psy.ReadTextFile(promptFile)
	if err != nil {
		return fmt.Errorf("prompt file: %w", err)
	}
	system, err := psy.ReadTextFile(systemFile)
	if err != nil {
		return fmt.Errorf("system file: %w", err)
	}

	// Generate JSON structured output:
	j := psy.Json{
		ID:        tuid.NewID().String(),
		MaxTokens: c.maxTokens,
		Model:     c.model,
		System:    system,
		Prompt:    prompt,
	}
	err = j.Generate(ctx, c.apiClient)
	if err != nil || verbose {
		fmt.Print(j.String())
	}
	if len(j.Codes) > 0 {
		b, _ := json.MarshalIndent(j.Codes, "", "  ")
		fmt.Print(string(b))
	}
	return err
}

// random handles the "text random" command to generate text for a random answer.
func (c *JsonCommand) random(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	verbose, _ := cmd.Flags().GetBool("verbose")
	answerID, _ := cmd.Flags().GetString("answer-id")
	answerFile := args[0]
	promptFile := args[1]
	systemFile := ""
	if len(args) > 2 {
		systemFile = args[2]
	}

	// Read the prompt and system files:
	prompt, err := psy.ReadTextFile(promptFile)
	if err != nil {
		return fmt.Errorf("prompt file: %w", err)
	}
	var system string
	if systemFile != "" {
		system, err = psy.ReadTextFile(systemFile)
		if err != nil {
			return fmt.Errorf("system file: %w", err)
		}
	}

	// Fetch the table of answers:
	answers, err := psy.ReadCSVTable(answerFile)
	if err != nil {
		return fmt.Errorf("answer file: %w", err)
	}

	// Select an answer record
	var a psy.Record
	if answerID == "" {
		// Unspecified? Select randomly.
		a = answers.Random()
		if a == nil {
			return fmt.Errorf("no records found in file %s", answerFile)
		}
	} else {
		// Select the specified answer by ID:
		nameValue := strings.Split(answerID, "=")
		if len(nameValue) != 2 {
			return fmt.Errorf("answer ID %s is not a name=value pair", answerID)
		}
		if !answers.HasField(nameValue[0]) {
			return fmt.Errorf("answer ID field %s not found in file %s", nameValue[0], answerFile)
		}
		a = answers.Record(nameValue[0], nameValue[1])
		if a == nil {
			return fmt.Errorf("answer %s not found in file %s", answerID, answerFile)
		}
	}

	// Prompt template substitutions
	for _, name := range answers.FieldNames {
		value := psy.CleanText(a[name])
		prompt = strings.ReplaceAll(prompt, "{{"+name+"}}", value)
	}

	// Generate a structured output request/response transaction
	j := psy.Json{
		ID:        tuid.NewID().String(),
		MaxTokens: c.maxTokens,
		Model:     c.model,
		System:    system,
		Prompt:    prompt,
	}
	err = j.Generate(ctx, c.apiClient)
	if err != nil || verbose {
		fmt.Print(j.String())
	}
	if len(j.Codes) > 0 {
		b, _ := json.MarshalIndent(j.Codes, "", "  ")
		fmt.Println(string(b))
	}
	return err
}

// parallel generates structured output for all answers in the specified file.
// Generations are processed concurrently in batches of the specified size.
// The results (responses and codes) are written to the specified CSV file.
func (c *JsonCommand) parallel(cmd *cobra.Command, args []string) error {
	startTime := time.Now()
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	batchSize, _ := cmd.Flags().GetInt("batch-size")
	outputFile := args[0]
	answerFile := args[1]
	promptFile := args[2]
	systemFile := ""
	if len(args) > 3 {
		systemFile = args[3]
	}

	// Create structured output transactions for all answers in the answerFile:
	jsons, answers, err := c.createTransactions(answerFile, promptFile, systemFile)
	if err != nil {
		return err
	}

	// Process the structured output generations concurrently, in batches:
	var count int
	var errorCount int
	results := make(map[string]psy.Json, len(jsons))
	retries := make([]psy.Json, 0)
	batches := psy.Batch(jsons, batchSize)
	fmt.Printf("Processing %d texts in %d batches of %d each...\n", len(jsons), len(batches), batchSize)
	for i, batch := range batches {
		// Process the batch:
		batchStart := time.Now()
		res := make(chan psy.Json, len(batch))
		var wg sync.WaitGroup
		wg.Add((len(batch)))
		for _, j := range batch {
			go func(j psy.Json) {
				defer wg.Done()
				j.Generate(ctx, c.apiClient)
				res <- j
			}(j)
		}
		wg.Wait()
		close(res)
		for j := range res {
			results[j.ID] = j
			count++
			if j.ErrMsg != "" {
				errorCount++
				retries = append(retries, j)
				fmt.Printf("%d: %s %dms %s\n", count, j.ID, j.Millis, j.ErrMsg)
			} else {
				fmt.Printf("%d: %s %dms\n", count, j.ID, j.Millis)
			}
		}

		// Report batch time taken, progress, and predicted time remaining:
		batchDuration := time.Since(batchStart)
		totalDuration := time.Since(startTime)
		averageDuration := totalDuration / time.Duration(count)
		timeRemaining := time.Duration(len(jsons)-count) * averageDuration
		percentComplete := float32(count) / float32(len(jsons)) * 100
		fmt.Printf("batch %d of %d: %d jsons in %s, %s avg, %.2f%% complete, %s remaining\n", i+1,
			len(batches), len(batch), batchDuration, averageDuration, percentComplete, timeRemaining)
	}

	// Retry any failed requests:
	if len(retries) > 0 {
		fmt.Printf("retrying %d failed requests\n", len(retries))
		var retryCount int
		res := make(chan psy.Json, len(retries))
		var wg sync.WaitGroup
		wg.Add((len(retries)))
		for _, j := range retries {
			go func(j psy.Json) {
				defer wg.Done()
				j.Generate(ctx, c.apiClient)
				res <- j
			}(j)
		}
		wg.Wait()
		close(res)
		for j := range res {
			results[j.ID] = j
			retryCount++
			if j.ErrMsg != "" {
				fmt.Printf("%d: %s %dms %s\n", count, j.ID, j.Millis, j.ErrMsg)
			} else {
				fmt.Printf("%d: %s %dms\n", retryCount, j.ID, j.Millis)
			}
		}
	}

	// Write the results to the specified CSV file:
	err = c.saveResults(results, answers, outputFile)

	// Report the total time taken:
	fmt.Printf("completed %d structured output generations (%d errors) in %s\n", len(jsons), errorCount, time.Since(startTime))
	return err
}

// batchCreate processes structured output generations in an asynchronous batch.
// It's similar to the 'parallel' command, but the concurrent batch
// processing is managed by Anthropic. Batches are usually processed
// in minutes, but may take up to 24 hours. If it takes longer than
// 24 hours, then the job times out. A Message Batch is limited to either
// 100,000 Message requests or 256 MB in size, whichever is reached first.
// If instructed to wait, batchCreate polls for progress and results.
// Once the batch is complete, it downloads the results file, processes it,
// and writes the results to the specified CSV output file.
func (c *JsonCommand) batchCreate(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	wait, _ := cmd.Flags().GetInt("wait")
	outputFile := args[0]
	answerFile := args[1]
	promptFile := args[2]
	systemFile := ""
	if len(args) > 3 {
		systemFile = args[3]
	}

	// Create structured output transactions for all answers in the answerFile:
	jsons, answers, err := c.createTransactions(answerFile, promptFile, systemFile)
	if err != nil {
		return err
	}

	// Create a batch of structured output requests:
	requests := make([]anthropic.MessageBatchNewParamsRequest, 0, len(jsons))
	for _, j := range jsons {
		requests = append(requests, j.BatchRequest())
	}
	b, err := c.apiClient.Messages.Batches.New(ctx,
		anthropic.MessageBatchNewParams{
			Requests: requests,
		})
	if err != nil {
		return fmt.Errorf("create batch: %w", err)
	}
	batchID := b.ID

	// Save the answers table as an incomplete results CSV file with transaction IDs:
	err = answers.WriteCSV(outputFile)
	if err != nil {
		return fmt.Errorf("save incomplete results file %s: %w", outputFile, err)
	}
	fmt.Printf("Saved incomplete results file %s with transaction IDs\n", outputFile)

	// If not waiting, provide instructions for monitoring progress:
	if wait == 0 {
		fmt.Printf("Created batch %s with %d requests.\n", b.ID, b.RequestCounts.Processing)
		fmt.Println("Use the following command to monitor progress:")
		fmt.Printf("psycode batch monitor %s\n", b.ID)
		fmt.Println("Once the batch is done, use the following command to process the results:")
		fmt.Printf("psycode json results %s %s\n", b.ID, outputFile)
		return nil
	}

	// Poll the batch operation for completion:
	fmt.Println("Polling for batch completion... (Ctrl+C to cancel)")
	fmt.Println("BatchID\tCreatedAt\tElapsed\tStatus\tSucceeded\tErrored\tProcessing")
	for {
		b, err = c.apiClient.Messages.Batches.Get(ctx, batchID, anthropic.MessageBatchGetParams{})
		if err != nil {
			return fmt.Errorf("read batch %s: %w", batchID, err)
		}
		t := time.Now()
		if !b.EndedAt.IsZero() {
			t = b.EndedAt
		}
		elapsed := t.Sub(b.CreatedAt)
		fmt.Printf("%s\t%s\t%v\t%s\t%d\t%d\t%d\n", b.ID, b.CreatedAt.Format(time.DateTime), elapsed,
			b.ProcessingStatus, b.RequestCounts.Succeeded, b.RequestCounts.Errored, b.RequestCounts.Processing)
		if b.ProcessingStatus == anthropic.MessageBatchProcessingStatusEnded {
			break
		}
		time.Sleep(time.Duration(wait) * time.Second)
	}

	// Process the results:
	return c.processBatchResults(ctx, batchID, outputFile)
}

// batchResults processes the results of a completed batch operation.
// It downloads the results file, processes it, and writes the results
// to the specified CSV output file.
func (c *JsonCommand) batchResults(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	batchID := args[0]
	outputFile := args[1]
	return c.processBatchResults(ctx, batchID, outputFile)
}

// createTransactions for all the answers in the answerFile.
func (c *JsonCommand) createTransactions(answerFile, promptFile, systemFile string) ([]psy.Json, *psy.Table, error) {
	// Fetch the table of answers:
	answers, err := psy.ReadCSVTable(answerFile)
	if err != nil {
		return []psy.Json{}, nil, fmt.Errorf("answer file: %w", err)
	}

	// Fetch the optional system template:
	var system string
	if systemFile != "" {
		system, err = psy.ReadTextFile(systemFile)
		if err != nil {
			return []psy.Json{}, answers, fmt.Errorf("system file: %w", err)
		}
	}

	// Fetch the prompt template:
	template, err := psy.ReadTextFile(promptFile)
	if err != nil {
		return []psy.Json{}, answers, fmt.Errorf("prompt file: %w", err)
	}

	// Create structured output transactions
	jsons := make([]psy.Json, 0, len(answers.Records))
	if !answers.HasField("tuid") {
		answers.AddField("tuid")
	}
	for _, a := range answers.Records {
		id := tuid.NewID().String()
		a["tuid"] = id
		prompt := template
		for _, name := range answers.FieldNames {
			value := psy.CleanText(a[name])
			prompt = strings.ReplaceAll(prompt, "{{"+name+"}}", value)
		}
		j := psy.Json{
			ID:        id,
			MaxTokens: c.maxTokens,
			Model:     c.model,
			System:    system,
			Prompt:    prompt,
		}
		jsons = append(jsons, j)
	}

	return jsons, answers, nil
}

// processBatchResults processes and persists the results of a completed batch operation.
func (c *JsonCommand) processBatchResults(ctx context.Context, batchID string, outputFile string) error {
	// Read the batch metadata:
	batch, e := c.apiClient.Messages.Batches.Get(ctx, batchID, anthropic.MessageBatchGetParams{})
	if e != nil {
		return fmt.Errorf("read batch %s: %w", batchID, e)
	}

	// Verify that the batch has results:
	if len(batch.ResultsURL) == 0 {
		return fmt.Errorf("batch %s has no results (yet?)", batchID)
	}

	// Read the incomplete results file:
	results, err := psy.ReadCSVTable(outputFile)
	if err != nil {
		return fmt.Errorf("read incomplete results file %s: %w", outputFile, err)
	}

	// Gather the batch results into a map indexed by transaction ID:
	var successCount int
	var errorCount int
	jsons := make(map[string]psy.Json)
	stream := c.apiClient.Messages.Batches.ResultsStreaming(ctx, batchID, anthropic.MessageBatchResultsParams{})
	for stream.Next() {
		r := stream.Current()
		j := psy.Json{
			ID: r.CustomID,
		}
		if len(j.ID) == 0 {
			continue
		}
		switch variant := r.Result.AsAny().(type) {
		case anthropic.MessageBatchSucceededResult:
			successCount++
			msg := r.Result.Message
			j.Response = ""
			for _, block := range msg.Content {
				if textBlock, ok := block.AsAny().(anthropic.TextBlock); ok {
					j.Response += textBlock.Text
				}
			}
			if len(j.Response) > 0 {
				e := json.Unmarshal([]byte(j.Response), &j.Codes)
				if e != nil {
					j.Response += "\nError parsing structured output: " + e.Error()
				}
			}
			j.StopReason = string(msg.StopReason)
			if len(msg.StopDetails.Explanation) > 0 {
				j.StopReason += " " + string(msg.StopDetails.Category) + " " + msg.StopDetails.Explanation
			}
			j.InputTokens = msg.Usage.InputTokens
			j.OutputTokens = msg.Usage.OutputTokens
		case anthropic.MessageBatchErroredResult:
			errorCount++
			j.ErrMsg = variant.Error.Error.Message
		case anthropic.MessageBatchExpiredResult:
			errorCount++
			j.ErrMsg = "Request expired."
		case anthropic.MessageBatchCanceledResult:
			errorCount++
			j.ErrMsg = "Request canceled."
		}
		jsons[j.ID] = j
	}
	err = stream.Err()
	if err != nil {
		return fmt.Errorf("streaming batch %s results: %w", batchID, err)
	}

	// Write the results to the specified output CSV file:
	err = c.saveResults(jsons, results, outputFile)
	fmt.Printf("completed %d texts (%d failed)\n", successCount, errorCount)
	return err
}

// Copy structured output from the results map into the CSV table and persist it.
func (c *JsonCommand) saveResults(jsons map[string]psy.Json, results *psy.Table, outputFile string) error {
	for _, r := range results.Records {
		id := r["tuid"]
		if id == "" {
			continue
		}
		j, ok := jsons[id]
		if !ok {
			continue
		}
		var response string
		if len(j.ErrMsg) > 0 {
			response += j.ErrMsg + "\n"
		}
		if j.StopReason != "end_turn" {
			response += j.StopReason + "\n"
		}
		response += j.Response
		r["generation"] = response
		if !results.HasField("generation") {
			results.AddField("generation")
		}
		for _, c := range j.Codes {
			name := c.Topic
			if c.Subtopic != "" {
				name += "_" + c.Subtopic
			}
			if name == "" {
				name = "code"
			}
			r[name] = c.Code
			if !results.HasField(name) {
				results.AddField(name)
			}
		}
	}
	return results.WriteCSV(outputFile)
}
