package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"psycode/psy"
	"strings"
	"time"
	"unicode"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/spf13/cobra"
	"github.com/voxtechnica/tuid-go"
)

// TextCommand represents the "text" command for generating text responses.
type TextCommand struct {
	apiClient   *anthropic.Client
	rootCmd     *cobra.Command
	baseCmd     *cobra.Command
	promptCmd   *cobra.Command
	randomCmd   *cobra.Command
	parallelCmd *cobra.Command
	batchCmd    *cobra.Command
	resultsCmd  *cobra.Command

	json           bool
	model          string
	maxTokens      int64
	questionField  string
	questionID     string
	answerID       string
	scoreField     string
	scoreSelect    string
	categoryField  string
	categoryTerms  string
	categorySelect string
}

// NewTextCommand creates a new instance of the TextCommand.
func NewTextCommand(apiClient *anthropic.Client, root *cobra.Command) *TextCommand {
	// Base Command
	c := &TextCommand{
		apiClient: apiClient,
		rootCmd:   root,
	}
	c.baseCmd = &cobra.Command{
		Use:   "text",
		Short: "Generate text",
		Long:  "Generate text from a specified prompt.",
	}
	c.baseCmd.PersistentFlags().StringVarP(&c.model, "model", "m", "claude-sonnet-5-5", "Model ID")
	c.baseCmd.PersistentFlags().Int64VarP(&c.maxTokens, "max-tokens", "t", 5120, "Max tokens to generate")
	c.rootCmd.AddCommand(c.baseCmd)

	// Prompt Command
	// Example: psycode text prompt data/limerick.txt data/system.txt -m claude-sonnet-5-5 -t 2048
	c.promptCmd = &cobra.Command{
		Use:   "prompt <promptFile> [systemFile]",
		Short: "Generate text from a test prompt",
		Long:  "Generate text from a test prompt provided in a specified file.",
		Args:  cobra.MinimumNArgs(1),
		RunE:  c.prompt,
	}
	c.promptCmd.Flags().BoolVarP(&c.json, "json", "j", false, "JSON output?")
	c.promptCmd.Flags().StringVarP(&c.scoreSelect, "score-select", "S", "none", "Score selection: first | last | all | none")
	c.promptCmd.Flags().StringVarP(&c.categorySelect, "category-select", "C", "none", "Category selection: first | last | all | none")
	c.promptCmd.Flags().StringVarP(&c.categoryTerms, "category-terms", "T", "", "Category terms, comma-separated")
	c.baseCmd.AddCommand(c.promptCmd)

	// Random Command
	// Example: psycode text random data/prompt.txt data/system.txt data/answers.csv data/questions.csv -a answer -q question -Q qid=angry
	c.randomCmd = &cobra.Command{
		Use:   "random <promptFile> <systemFile> <answerFile> [questionFile]",
		Short: "Generate text for a random answer",
		Long:  "Generate text for a random answer from a specified file.",
		Args:  cobra.MinimumNArgs(3),
		RunE:  c.random,
	}
	c.randomCmd.Flags().BoolVarP(&c.json, "json", "j", false, "JSON output?")
	c.randomCmd.Flags().StringVarP(&c.scoreSelect, "score-select", "S", "none", "Score selection: first | last | all | none")
	c.randomCmd.Flags().StringVarP(&c.categorySelect, "category-select", "C", "none", "Category selection: first | last | all | none")
	c.randomCmd.Flags().StringVarP(&c.categoryTerms, "category-terms", "T", "", "Category terms, comma-separated")
	c.randomCmd.Flags().StringVarP(&c.questionID, "question-id", "Q", "", "Question ID (optional, name | name=value)")
	c.randomCmd.Flags().StringVarP(&c.questionField, "question-field", "q", "", "Question field name (optional)")
	c.randomCmd.Flags().StringVarP(&c.answerID, "answer-id", "A", "", "Answer ID (optional, name=value)")
	c.baseCmd.AddCommand(c.randomCmd)

	// Parallel Command
	// Example: psycode text parallel data/scores.csv data/prompt.txt data/system.txt data/answers.csv data/questions.csv -a answer -q question -Q qid=angry -m claude-sonnet-5-5
	c.parallelCmd = &cobra.Command{
		Use:   "parallel <outputFile> <promptFile> <systemFile> <answerFile> [questionFile]",
		Short: "Generate text for all answers in parallel",
		Long:  "Generate text for all answers using concurrent client-side requests.",
		Args:  cobra.MinimumNArgs(4),
		RunE:  c.parallel,
	}
	c.parallelCmd.Flags().IntP("batch-size", "b", 20, "Concurrent request batch size")
	c.parallelCmd.Flags().StringVarP(&c.categoryField, "category-field", "c", "category", "Category field name")
	c.parallelCmd.Flags().StringVarP(&c.categorySelect, "category-select", "C", "none", "Category selection: first | last | all | none")
	c.parallelCmd.Flags().StringVarP(&c.categoryTerms, "category-terms", "T", "", "Category terms, comma-separated")
	c.parallelCmd.Flags().StringVarP(&c.scoreField, "score-field", "s", "score", "Score field name")
	c.parallelCmd.Flags().StringVarP(&c.scoreSelect, "score-select", "S", "none", "Score selection: first | last | all | none")
	c.parallelCmd.Flags().StringVarP(&c.questionID, "question-id", "Q", "", "Question ID (optional, name | name=value)")
	c.parallelCmd.Flags().StringVarP(&c.questionField, "question-field", "q", "", "Question field name (optional)")
	c.baseCmd.AddCommand(c.parallelCmd)

	// Batch Command
	// Example: psycode text batch data/scores-batch.csv data/prompt.txt data/system.txt data/answers.csv data/questions.csv -a answer -q question -Q qid=angry -m claude-sonnet-5-5 -w 10
	c.batchCmd = &cobra.Command{
		Use:   "batch <outputFile> <promptFile> <systemFile> <answerFile> [questionFile]",
		Short: "Generate text for all answers as a batch",
		Long:  "Generate text for all answers using a server-side asynchronous batch.",
		Args:  cobra.MinimumNArgs(4),
		RunE:  c.batchCreate,
	}
	c.batchCmd.Flags().IntP("wait", "w", 0, "Wait for results? Polling interval in seconds (recommend 10)")
	c.batchCmd.Flags().StringVarP(&c.categoryField, "category-field", "c", "category", "Category field name")
	c.batchCmd.Flags().StringVarP(&c.categorySelect, "category-select", "C", "none", "Category selection: first | last | all | none")
	c.batchCmd.Flags().StringVarP(&c.categoryTerms, "category-terms", "T", "", "Category terms, comma-separated")
	c.batchCmd.Flags().StringVarP(&c.scoreField, "score-field", "s", "score", "Score field name")
	c.batchCmd.Flags().StringVarP(&c.scoreSelect, "score-select", "S", "none", "Score selection: first | last | all | none")
	c.batchCmd.Flags().StringVarP(&c.questionID, "question-id", "Q", "", "Question ID (optional, name | name=value)")
	c.batchCmd.Flags().StringVarP(&c.questionField, "question-field", "q", "", "Question field name (optional)")
	c.baseCmd.AddCommand(c.batchCmd)

	// Results Command
	// Example: psycode text results <batchID> data/scores.csv
	c.resultsCmd = &cobra.Command{
		Use:   "results <batchID> <outputFile>",
		Short: "Process batch results",
		Long:  "Process the results of a completed batch operation.",
		Args:  cobra.ExactArgs(2),
		RunE:  c.batchResults,
	}
	c.resultsCmd.Flags().StringVarP(&c.categoryField, "category-field", "c", "category", "Category field name")
	c.resultsCmd.Flags().StringVarP(&c.categorySelect, "category-select", "C", "none", "Category selection: first | last | all | none")
	c.resultsCmd.Flags().StringVarP(&c.categoryTerms, "category-terms", "T", "", "Category terms, comma-separated")
	c.resultsCmd.Flags().StringVarP(&c.scoreField, "score-field", "s", "score", "Score field name")
	c.resultsCmd.Flags().StringVarP(&c.scoreSelect, "score-select", "S", "none", "Score selection: first | last | all | none")
	c.baseCmd.AddCommand(c.resultsCmd)

	return c
}

// gatherFlagParameters gathers and validates the flag parameters
func (c *TextCommand) gatherFlagParameters() (psy.TextParameters, error) {
	p := psy.TextParameters{
		QuestionField:  c.questionField,
		QuestionID:     c.questionID,
		AnswerID:       c.answerID,
		CategoryField:  c.categoryField,
		CategorySelect: psy.Selection(strings.ToLower(c.categorySelect)),
		CategoryTerms:  strings.Split(c.categoryTerms, ","),
		ScoreField:     c.scoreField,
		ScoreSelect:    psy.Selection(strings.ToLower(c.scoreSelect)),
		Model:          c.model,
		MaxTokens:      c.maxTokens,
	}

	// Validate the category selection:
	if !p.CategorySelect.IsValid() {
		return p, fmt.Errorf("invalid category selection (expect first, last, all, or none): %s", p.CategorySelect)
	}

	// Validate and tidy up category terms (leading/trailing white space and punctuation)
	if len(p.CategoryTerms) == 0 && p.CategorySelect != psy.None {
		return p, fmt.Errorf("missing comma-separated category terms")
	}
	for i, term := range p.CategoryTerms {
		p.CategoryTerms[i] = strings.TrimFunc(term, func(r rune) bool {
			return unicode.IsSpace(r) || unicode.IsPunct(r)
		})
	}

	// Validate the score selection:
	if !p.ScoreSelect.IsValid() {
		return p, fmt.Errorf("invalid score selection (expect first, last, all, or none): %s", p.ScoreSelect)
	}

	// Validate the model:
	_, err := c.apiClient.Models.Get(context.Background(), p.Model, anthropic.ModelGetParams{})
	if err != nil {
		return p, fmt.Errorf("model %s is invalid: %w", p.Model, err)
	}

	return p, nil
}

// prompt handles the "text prompt" command to generate text from a prompt file.
func (c *TextCommand) prompt(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	p, err := c.gatherFlagParameters()
	if err != nil {
		return err
	}
	p.PromptFile = args[0]
	p.SystemFile = ""
	if len(args) > 1 {
		p.SystemFile = args[1]
	}

	// Read the prompt and system files:
	prompt, err := psy.ReadTextFile(p.PromptFile)
	if err != nil {
		return fmt.Errorf("prompt file: %w", err)
	}
	system, err := psy.ReadTextFile(p.SystemFile)
	if err != nil {
		return fmt.Errorf("system file: %w", err)
	}

	// Generate and output a text response:
	text := psy.Text{
		ID:             tuid.NewID().String(),
		MaxTokens:      p.MaxTokens,
		Model:          p.Model,
		System:         system,
		Prompt:         prompt,
		CategorySelect: p.CategorySelect,
		CategoryTerms:  p.CategoryTerms,
		ScoreSelect:    p.ScoreSelect,
	}
	return c.generateTextResponse(ctx, text)
}

// random handles the "text random" command to generate text for a random answer.
func (c *TextCommand) random(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	p, err := c.gatherFlagParameters()
	if err != nil {
		return err
	}
	if p.AnswerID == "" {
		p.AnswerID = "random"
	}
	p.PromptFile = args[0]
	p.SystemFile = args[1]
	p.AnswerFile = args[2]
	p.QuestionFile = ""
	if len(args) > 3 {
		p.QuestionFile = args[3]
	}

	// Generate the text requests:
	texts, _, err := c.generateTextRequests(p)
	if err != nil {
		return fmt.Errorf("generate text requests: %w", err)
	}
	if len(texts) == 0 {
		return fmt.Errorf("no text request generated")
	}
	text := texts[0]

	// Generate and output a text response:
	return c.generateTextResponse(ctx, text)
}

// parallel generates text for all answers in the specified file.
// Text generations are processed concurrently in batches of the specified size.
// The results (responses plus scores) are written to the specified CSV file.
// If the question-id is just 'name' instead of 'name=value', then that field
// name is used in both the question file and the answer file to look up the
// question for each answer.
func (c *TextCommand) parallel(cmd *cobra.Command, args []string) error {
	startTime := time.Now()
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	batchSize, _ := cmd.Flags().GetInt("batch-size")
	p, err := c.gatherFlagParameters()
	if err != nil {
		return err
	}
	p.OutputFile = args[0]
	p.PromptFile = args[1]
	p.SystemFile = args[2]
	p.AnswerFile = args[3]
	p.QuestionFile = ""
	if len(args) > 4 {
		p.QuestionFile = args[4]
	}

	// Generate the text requests:
	texts, answers, err := c.generateTextRequests(p)
	if err != nil {
		return fmt.Errorf("generate text requests: %w", err)
	}
	if len(texts) == 0 {
		return fmt.Errorf("no text requests generated")
	}

	// Process the text generations concurrently, in batches:
	var count int
	var errorCount int
	var maxScoreCount int
	results := make(map[string]psy.Text, len(texts))
	retries := make([]psy.Text, 0)
	batches := psy.Batch(texts, batchSize)
	fmt.Printf("Processing %d texts in %d batches of %d each...\n", len(texts), len(batches), batchSize)
	for i, batch := range batches {
		// Process the batch:
		batchStart := time.Now()
		r := psy.GenerateTextParallel(ctx, c.apiClient, batch)

		// Gather the results:
		for _, text := range r {
			count++
			if text.ErrMsg != "" {
				errorCount++
				retries = append(retries, text)
				fmt.Printf("%d: %s %dms %s\n", count, text.ID, text.Millis, text.ErrMsg)
			} else {
				fmt.Printf("%d: %s %dms\n", count, text.ID, text.Millis)
			}
			if len(text.Scores) > maxScoreCount {
				maxScoreCount = len(text.Scores)
			}
			results[text.ID] = text
		}

		// Report batch time taken, progress, and predicted time remaining:
		batchDuration := time.Since(batchStart)
		totalDuration := time.Since(startTime)
		averageDuration := totalDuration / time.Duration(count)
		timeRemaining := time.Duration(len(texts)-count) * averageDuration
		percentComplete := float32(count) / float32(len(texts)) * 100
		fmt.Printf("batch %d of %d: %d texts in %s, %s avg, %.2f%% complete, %s remaining\n", i+1,
			len(batches), len(batch), batchDuration, averageDuration, percentComplete, timeRemaining)
	}

	// Retry any failed requests:
	if len(retries) > 0 {
		fmt.Printf("retrying %d failed requests\n", len(retries))
		var retryCount int
		r := psy.GenerateTextParallel(ctx, c.apiClient, retries)
		for _, text := range r {
			retryCount++
			if text.ErrMsg != "" {
				fmt.Printf("%d: %s %dms %s\n", count, text.ID, text.Millis, text.ErrMsg)
			} else {
				fmt.Printf("%d: %s %dms\n", retryCount, text.ID, text.Millis)
			}
			if len(text.Scores) > maxScoreCount {
				maxScoreCount = len(text.Scores)
			}
			results[text.ID] = text
		}
	}

	// Write the results to the specified CSV file:
	err = c.saveResults(results, answers, p, maxScoreCount)

	// Report the total time taken:
	fmt.Printf("completed %d text generations (%d errors) in %s\n", len(texts), errorCount, time.Since(startTime))
	return err
}

// batchCreate processes text generations in an asynchronous batch.
// It's similar to the 'parallel' command, but the concurrent batch
// processing is managed by Anthropic. Batches are usually processed
// within an hour, but may take up to 24 hours. If it takes longer than
// 24 hours, then the job times out. A Message Batch is limited to either
// 100,000 Message requests or 256 MB in size, whichever is reached first.
// If instructed to wait, batchCreate polls for progress and results.
// Once the batch is complete, it downloads the results file, processes it,
// and writes the results to the specified CSV output file.
func (c *TextCommand) batchCreate(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	wait, _ := cmd.Flags().GetInt("wait")
	p, err := c.gatherFlagParameters()
	if err != nil {
		return err
	}
	p.OutputFile = args[0]
	p.PromptFile = args[1]
	p.SystemFile = args[2]
	p.AnswerFile = args[3]
	p.QuestionFile = ""
	if len(args) > 4 {
		p.QuestionFile = args[4]
	}

	// Generate the text requests:
	texts, answers, err := c.generateTextRequests(p)
	if err != nil {
		return fmt.Errorf("generate text requests: %w", err)
	}
	if len(texts) == 0 {
		return fmt.Errorf("no text requests generated")
	}

	// Generate a batch request:
	requests := make([]anthropic.MessageBatchNewParamsRequest, 0, len(texts))
	for _, text := range texts {
		params := anthropic.MessageBatchNewParamsRequestParams{
			MaxTokens: text.MaxTokens,
			Model:     anthropic.Model(text.Model),
			Messages: []anthropic.MessageParam{
				anthropic.NewUserMessage(anthropic.NewTextBlock(text.Prompt)),
			},
		}
		if len(text.System) > 0 {
			params.System = []anthropic.TextBlockParam{{
				Text:         text.System,
				CacheControl: anthropic.NewCacheControlEphemeralParam(),
			}}
		}
		request := anthropic.MessageBatchNewParamsRequest{
			CustomID: text.ID,
			Params:   params,
		}
		requests = append(requests, request)
	}

	// Upload the batch of text message requests
	batch, err := c.apiClient.Messages.Batches.New(ctx,
		anthropic.MessageBatchNewParams{
			Requests: requests,
		})
	if err != nil {
		return fmt.Errorf("generate new text batch: %w", err)
	}

	// Display the resulting batch request
	batchID := batch.ID
	j, err := json.MarshalIndent(batch, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshalling Batch %s JSON: %w", batchID, err)
	}
	fmt.Println(string(j))

	// Save the answers table as an incomplete results CSV file with Text IDs:
	err = answers.WriteCSV(p.OutputFile)
	if err != nil {
		return fmt.Errorf("save incomplete results file %s: %w", p.OutputFile, err)
	}
	fmt.Printf("saved incomplete results file (with text IDs): %s\n", p.OutputFile)

	// Poll the batch operation for completion:
	if wait > 0 {
		fmt.Println("polling for batch completion... (Ctrl+C to cancel)")
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
		// Process the results:
		return c.processBatchResults(ctx, batchID, p)
	}

	// If not waiting, provide instructions for monitoring progress:
	fmt.Println("Use the following command to monitor progress:")
	fmt.Printf("psycode batch monitor %s\n", batch.ID)
	fmt.Println("Once the batch is done, use the following command to process the results:")
	flags := ""
	switch p.CategorySelect {
	case psy.All:
		flags += " -C all -T " + c.categoryTerms
	case psy.First, psy.Last:
		flags += " -C " + c.categorySelect + " -c " + c.categoryField + " -T " + c.categoryTerms
	}
	if p.ScoreSelect != psy.None {
		flags += " -S " + c.scoreSelect + " -s " + c.scoreField
	}
	fmt.Printf("psycode text results%s %s %s\n", flags, batch.ID, p.OutputFile)
	return nil
}

// batchResults processes the results of a completed batch operation.
// It downloads the results file, processes it, and writes the results
// to the specified CSV output file.
func (c *TextCommand) batchResults(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	p, err := c.gatherFlagParameters()
	if err != nil {
		return err
	}
	p.OutputFile = args[1]
	batchID := args[0]
	return c.processBatchResults(ctx, batchID, p)
}

// generateTextRequests generates text requests based on the provided TextParameters.
func (c *TextCommand) generateTextRequests(p psy.TextParameters) ([]psy.Text, *psy.Table, error) {
	var texts []psy.Text
	var err error

	// Fetch the optional system template:
	var system string
	if p.SystemFile != "" {
		system, err = psy.ReadTextFile(p.SystemFile)
		if err != nil {
			return texts, nil, fmt.Errorf("system file: %w", err)
		}
	}

	// Fetch the prompt template:
	template, err := psy.ReadTextFile(p.PromptFile)
	if err != nil {
		return texts, nil, fmt.Errorf("prompt file: %w", err)
	}

	// Read the optional question(s):
	var questions map[string]string
	var question string
	var lookupQuestion bool
	if p.QuestionFile != "" {
		if strings.Contains(p.QuestionID, "=") {
			// Lookup the question by ID:
			question, err = psy.ReadCSVField(p.QuestionFile, p.QuestionID, p.QuestionField)
			if err != nil {
				return texts, nil, fmt.Errorf("read question: %w", err)
			}
		} else {
			// Read all the questions:
			lookupQuestion = true
			questions, err = psy.ReadCSVFields(p.QuestionFile, p.QuestionID, p.QuestionField)
			if err != nil {
				return texts, nil, fmt.Errorf("read questions: %w", err)
			}
		}
	}

	// Fetch the table of answers:
	answers, err := psy.ReadCSVTable(p.AnswerFile)
	if err != nil {
		return texts, nil, fmt.Errorf("answer file: %w", err)
	}
	if lookupQuestion {
		if !answers.HasField(p.QuestionID) {
			return texts, answers, fmt.Errorf("question ID field %s not found in %s", p.QuestionID, p.AnswerFile)
		}
		// Validate the question IDs in the answer file:
		unknownQuestions := make([]string, 0)
		for _, a := range answers.Records {
			qid := a[p.QuestionID]
			if _, ok := questions[qid]; !ok {
				unknownQuestions = append(unknownQuestions, qid)
			}
		}
		if len(unknownQuestions) > 0 {
			return texts, answers, fmt.Errorf("unknown question IDs in answer file %s: %s", p.AnswerFile, strings.Join(unknownQuestions, ", "))
		}
	}

	// Select one or all records, as specified:
	var records []psy.Record
	if p.AnswerID == "random" {
		// Select a random answer:
		record := answers.Random()
		if record == nil {
			return texts, answers, fmt.Errorf("no records found in file %s", p.AnswerFile)
		}
		records = []psy.Record{record}
	} else if p.AnswerID != "" {
		// Select the specified answer by ID:
		nameValue := strings.Split(p.AnswerID, "=")
		if len(nameValue) != 2 {
			return texts, answers, fmt.Errorf("answer ID %s is not a name=value pair", p.AnswerID)
		}
		if !answers.HasField(nameValue[0]) {
			return texts, answers, fmt.Errorf("answer ID field %s not found in file %s", nameValue[0], p.AnswerFile)
		}
		record := answers.Record(nameValue[0], nameValue[1])
		if record == nil {
			return texts, answers, fmt.Errorf("answer %s not found in file %s", p.AnswerID, p.AnswerFile)
		}
		records = []psy.Record{record}
	} else {
		// Select all records:
		records = answers.Records
	}

	// Generate a text request for each answer, skipping blanks. Also, add a
	// new column to the answer table, indicating its unique text ID. This is
	// used to reconcile the answers with the generated text responses.
	texts = make([]psy.Text, 0, len(records))
	answers.AddField("textID")
	for _, a := range records {
		// Generate a unique text ID:
		textID := tuid.NewID().String()
		a["textID"] = textID
		// Prepare the prompt from the template:
		var q string
		if lookupQuestion {
			qid := a[p.QuestionID]
			q = questions[qid]
		} else {
			q = question
		}
		prompt := strings.ReplaceAll(template, "{{question}}", q)
		for _, name := range answers.FieldNames {
			value := psy.CleanText(a[name])
			prompt = strings.ReplaceAll(prompt, "{{"+name+"}}", value)
		}
		// Generate the text request:
		text := psy.Text{
			ID:             textID,
			MaxTokens:      p.MaxTokens,
			Model:          p.Model,
			System:         system,
			Prompt:         prompt,
			CategorySelect: p.CategorySelect,
			CategoryTerms:  p.CategoryTerms,
			ScoreSelect:    p.ScoreSelect,
		}
		texts = append(texts, text)
	}

	return texts, answers, nil
}

// generateTextResponse generates a text response based on the provided Text object and selection.
func (c *TextCommand) generateTextResponse(ctx context.Context, text psy.Text) error {
	// Generate the text response:
	text, err := psy.GenerateText(ctx, c.apiClient, text)
	if err != nil {
		return fmt.Errorf("text generation: %w", err)
	}
	if c.json {
		b, _ := json.MarshalIndent(text, "", "  ")
		fmt.Println(string(b))
	} else {
		fmt.Print(text.String())
	}
	return nil
}

// processBatchResults processes the results of a completed batch operation.
func (c *TextCommand) processBatchResults(ctx context.Context, batchID string, p psy.TextParameters) error {
	// Read the batch metadata:
	batch, e := c.apiClient.Messages.Batches.Get(ctx, batchID, anthropic.MessageBatchGetParams{})
	if e != nil {
		return fmt.Errorf("read batch %s: %w", batchID, e)
	}

	// Verify that the batch has results:
	if len(batch.ResultsURL) == 0 {
		return fmt.Errorf("batch %s has no results", batchID)
	}

	// Read the incomplete results file:
	results, err := psy.ReadCSVTable(p.OutputFile)
	if err != nil {
		return fmt.Errorf("read incomplete results file %s: %w", p.OutputFile, err)
	}

	// Gather the batch results into a map indexed by text ID:
	var successCount int
	var errorCount int
	var maxScoreCount int
	texts := make(map[string]psy.Text)
	stream := c.apiClient.Messages.Batches.ResultsStreaming(ctx, batchID, anthropic.MessageBatchResultsParams{})
	for stream.Next() {
		response := stream.Current()
		text := psy.Text{
			ID: response.CustomID,
		}
		if len(text.ID) == 0 {
			continue
		}
		switch variant := response.Result.AsAny().(type) {
		case anthropic.MessageBatchSucceededResult:
			successCount++
			message := response.Result.Message
			text.Response = ""
			for _, block := range message.Content {
				if textBlock, ok := block.AsAny().(anthropic.TextBlock); ok {
					text.Response += textBlock.Text
				}
			}
			text.Categories = psy.SelectCategories(text.Response, p.CategoryTerms, p.CategorySelect)
			text.Scores = psy.SelectScores(text.Response, p.ScoreSelect)
			if len(text.Scores) > maxScoreCount {
				maxScoreCount = len(text.Scores)
			}
			text.StopReason = string(message.StopReason)
			if len(message.StopDetails.Explanation) > 0 {
				text.StopReason += " " + string(message.StopDetails.Category) + " " + message.StopDetails.Explanation
			}
			text.InputTokens = message.Usage.InputTokens
			text.OutputTokens = message.Usage.OutputTokens
		case anthropic.MessageBatchErroredResult:
			errorCount++
			text.ErrMsg = variant.Error.Error.Message
		case anthropic.MessageBatchExpiredResult:
			errorCount++
			text.ErrMsg = "Request expired."
		case anthropic.MessageBatchCanceledResult:
			errorCount++
			text.ErrMsg = "Request canceled."
		}
		texts[text.ID] = text
	}
	err = stream.Err()
	if err != nil {
		return fmt.Errorf("streaming batch %s results: %w", batchID, err)
	}

	// Write the results to the specified output CSV file:
	err = c.saveResults(texts, results, p, maxScoreCount)
	fmt.Printf("completed %d texts (%d failed)\n", successCount, errorCount)
	return err
}

// SaveResults persists the generated texts, categories, and scores in the
// specified table and its associated CSV file.
func (c *TextCommand) saveResults(texts map[string]psy.Text, results *psy.Table,
	p psy.TextParameters, maxScoreCount int) error {
	for _, record := range results.Records {
		textID := record["textID"]
		if textID == "" {
			continue
		}
		text, ok := texts[textID]
		if !ok {
			continue
		}
		var output string
		if len(text.ErrMsg) > 0 {
			output += text.ErrMsg + "\n"
		}
		if text.StopReason != "end_turn" {
			output += text.StopReason + "\n"
		}
		output += text.Response
		record["generation"] = output
		if p.CategorySelect != psy.None {
			for _, category := range text.Categories {
				if p.CategorySelect == psy.First || p.CategorySelect == psy.Last {
					record[p.CategoryField] = category
				} else {
					record[category] = "1"
				}
			}
		}
		if p.ScoreSelect != psy.None {
			for i, score := range text.Scores {
				field := p.ScoreField
				if maxScoreCount > 1 {
					field = fmt.Sprintf("%s%d", p.ScoreField, i+1)
				}
				record[field] = fmt.Sprintf("%f", score)
			}
		}
	}

	// Add field names to the results table:
	results.AddField("generation")
	if p.CategorySelect != psy.None {
		if p.CategorySelect == psy.First || p.CategorySelect == psy.Last {
			results.AddField(p.CategoryField)
		} else {
			for _, term := range p.CategoryTerms {
				results.AddField(term)
			}
		}
	}
	if p.ScoreSelect != psy.None {
		if maxScoreCount == 1 {
			results.AddField(p.ScoreField)
		} else if maxScoreCount > 1 {
			for i := 1; i <= maxScoreCount; i++ {
				field := fmt.Sprintf("%s%d", p.ScoreField, i)
				results.AddField(field)
			}
		}
	}

	// Write the results to the specified output CSV file:
	return results.WriteCSV(p.OutputFile)
}
