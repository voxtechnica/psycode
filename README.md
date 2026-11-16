# PsyCode

This project provides a command-line application that uses a large language 
model (LLM) for text analysis, supporting content coding in psychology research.
[psycode](https://github.com/voxtechnica/psycode) uses the Anthropic 
[Claude API](https://platform.claude.com/docs/en/api/overview), and it's similar 
to [gpt](https://github.com/voxtechnica/gpt), which uses the 
[OpenAI API](https://platform.openai.com/docs/api-reference).

PsyCode can be used for scoring and/or categorizing textual responses provided by 
study participants. You create a prompt template file with substitution variables 
like `{{fieldName}}` and you specify a CSV file containing the study data. PsyCode
will work through each record in your dataset, sending prompts to the LLM, recording 
the responses, and extracting scores and/or categories as specified. The results are
stored in an output CSV file that matches the input CSV file, except with a few new 
columns containing the content coding results.

## Developer Workstation Setup

* Install the [Go Language](https://golang.org/doc/install), and set up
  a [GOPATH environment variable](https://go.dev/wiki/SettingGOPATH).
* Install an IDE, such as [VSCode](https://code.visualstudio.com/) with
  the [Go extension](https://marketplace.visualstudio.com/items?itemName=golang.go).
* Install some Go tools used in the [Makefile](Makefile) in your GOPATH bin folder:

```bash
make tools
```

* Configure the following environment variables for 
  your [Claude API account](https://platform.claude.com/settings/keys). For
  simplicity, you could put them in your `.bashrc`, `.zshrc`, or equivalent.
  Substitute the placeholders below with your own values, of course.

```bash
export ANTHROPIC_API_KEY="sk-ant-usr-xxxxxxxxxxxxxxxxxxxxxxxxxxx-..."
export ANTHROPIC_WORKSPACE="wrkspc_xxxxxxxxxxxxxxxxxxxxxxxx"
```

* Alternatively, you can create a file named `.env` and place the above environment
variables there. Example contents:

```text
ANTHROPIC_API_KEY="sk-ant-usr-xxxxxxxxxxxxxxxxxxxxxxxxxxx-..."
ANTHROPIC_WORKSPACE="wrkspc_xxxxxxxxxxxxxxxxxxxxxxxx"
```

* Build the application for running in your local development environment:

```bash
make build
```

## Install the Command-line Application

If you don't intend to modify the application, you can install it on your workstation
by simply downloading the appropriate [executable application](/dist/) and running it
locally from the command-line.

If your workstation is set up for Go language development, you can install the latest
version from source using the following command:

```bash
go install github.com/voxtechnica/psycode@latest
```

## Run the Command-line Application

To run the command-line application, be sure to either set the environment variables
identified above or create a `.env` file for your Claude API account. To verify that the
application is configured properly, you can run the `psycode about` command. It will show
your Workspace ID and API key.

You can explore the available commands, subcommands, and optional flags using
the `-h` or `--help` flags with the executable application. For example:

```bash
./psycode -h
./psycode model -h
./psycode model list -h
./psycode model read -h
./psycode text -h
./psycode text prompt -h
./psycode text random -h
./psycode text batch -h
```

Listing the models is a convenient way to verify that you can access the Claude API
as expected.

Also, you can explore the [CLI documentation](/docs/psycode.md).

## Working with Text and CSV Files

Some of the commands (e.g. `text random` and `text batch`) use CSV files for data
inputs and outputs. The first line of these files is expected to be column labels,
and they need to be unique. If the column label is missing, it gets labeled as
"columnX" where X is the column number. The subsequent lines of the CSV file are
individual rows of data.

When you're using the `text random` and `text batch` commands, you can refer to
different fields in your dataset by using the column labels. They're case-sensitive,
and ideally, have no spaces. Good examples are `id`, `question`, and `answer`.
You can include extra columns in your CSV files, and they're either ignored or
reproduced faithfully in your output file.

If you want to refer to a specific field in a specific row of your dataset, you'll
be supplying two arguments: the name and value of an "id" field, and the name of
the field that contains the value you're referring to. The "id" field will be a
name=value pair, such as `pid=61324`.

## Using the text Command

The [text command](/docs/psycode_text.md) has four subcommands: `prompt`, `random`,
`parallel`, and `batch`. The first two commands will typically be used when
engineering an effective prompt for use with a batch of data. The `prompt` command
works with just two text files: a prompt file and an optional system file. These
files are plain UTF-8 text. The system file can be used to inform psycode of its
identity and the role it is expected to play. The prompt file then provides 
instructions for psycode. See Anthropic's [prompting best practices](https://platform.claude.com/docs/en/build-with-claude/prompt-engineering/claude-prompting-best-practices) 
for more information.

Note that for the `random`, `parallel`, and `batch` commands, the prompt file can
contain template substitution variables. `{{question}}` is used for an optional
question coming from a CSV file containing a column of question IDs and a column
of questions. You'll be specifying these column names as command line flags.
Whereas the question variable is optional (because the question could simply be 
stated as explicit text in your prompt), you'll definitely want to include one or
more `{{fieldName}}` variables in your prompt file. Substitute the actual column
names in your answers CSV file for "fieldName", and note the lack of spaces and 
case-sensitivity of the substitution variables.

So, the `prompt` command is used for simple prompts, and when you're ready to
start experimenting with question(s) and answers in your CSV dataset, you can
use the `random` command to test your prompt with different values. If you want
to finesse the prompt with a specific answer, you can use the `--answer-id` flag
with the `random` command to force it to select the specified answer (which, of
course, is not random).

Once you're happy with the results you're seeing, you can use the `parallel` or
`batch` commands to process the entire dataset. The `parallel` command works
through your records in real time, sending multiple requests in parallel. The
`batch` command generates an asynchronous batch job, wherein Anthropic manages
the concurrency. Using `batch` is recommended, as it costs about half as much
as the real-time requests. Also, in testing, it appears to complete very quickly.

The `text` commands can also parse "scores" (numbers) from the psycode response text.
The `--score-select` flag indicates whether you'd like the first number found in
the text, the last number, all the numbers, or none of the numbers (i.e. don't
bother parsing scores). You'll probably just want the last number. And, of course,
your prompt file will be providing instructions to the LLM for scoring the results.

The `text` commands can also be used for categorizing text. You'll provide the
expected categorization terms using the `category-terms` flag and a comma-separated
list of expected terms. Example: `-T Good,Bad,Neutral`. Use the `category-select`
flag to parse out the first, last, or all of the expected terms in the response.
If you select `first` or `last`, then you'll probably want to specify the 
`category-field` name for storing results in your output CSV file. If you select
`all` category terms present in the response, then the terms become column names
in your output CSV file, and if the term is present, the column will contain a `1`
value. If the term is not present in the response for that record, then the column 
will be blank.

When you use the `parallel` or `batch` commands, the output CSV file will contain
all the data provided in your input answer file, along with a few new columns. The
`textID` column will contain a unique ID used with psycode, and the `generation`
column will contain the exact text of the psycode response. If you're also
parsing score(s) or category terms from the response, those column(s) will be 
included too. The default name for a score column is `score` or `scoreX` where 
X is a number indicating the position of the score in the list, if multiple 
scores are being parsed from the response text. Note that you can change the 
name of the score column using the `--score-field` flag. This can be useful if 
you're storing the results from multiple runs in the same CSV file. You can 
just give each run a different score field name, reusing the output file as 
an input file. The new score columns will be appended with each run.

One more tip on using questions and answers: if you have multiple questions,
and your answer file includes answers to different questions, you can include
a question ID column in your answers file. Then, when processing each answer,
the system will look up the appropriate question to use for that answer in
the questions CSV file. The question ID column name (e.g. `qid`) must be the
same in both the questions file and the answers file. The `--question-id` flag
controls this behavior. If you specify just a question ID field name, then it
will be assumed that the field exists in both files. If you specify a
`name=value` pair, then just that specific question will be used for the
entire set of answers. Also, note that questions are optional. If all you have
to process are "answers", then you can ignore the question bits.
