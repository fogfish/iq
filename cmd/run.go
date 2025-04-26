package cmd

import (
	"context"
	"os"

	spec "github.com/fogfish/iq/internal/prompt"
	"github.com/spf13/cobra"
)

var (
	runDir     string
	runOut     string
	runWorkDir string
	runMutable bool
	runStrict  bool
)

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.PersistentFlags().StringVarP(&runDir, "dir", "d", ".", "input directory or S3 path to read files from")
	runCmd.PersistentFlags().StringVarP(&runOut, "out", "o", "", "output directory or S3 path to write results")
	runCmd.PersistentFlags().StringVar(&runWorkDir, "workdir", "", "work directory for tools and commands")
	runCmd.PersistentFlags().BoolVar(&runMutable, "mutable", false, "enable mutable processing, input file is removed\nafter results are saved (allows implementation of fail safe pipelines)")
	runCmd.PersistentFlags().BoolVar(&runStrict, "strict", false, "enforce strict processing, failing on the first error")
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "process files in mounted dir with LLM agent",
	Long: `
A workflow in 'iq' represents a structured sequence of operations that an LLM
can execute autonomously using available commands and tools. Unlike a single
prompt, a workflow allows the model to reason through multiple steps — such as
reading, generating, modifying, and combining files — to accomplish a complex
goal. Each step builds on the last, forming a coherent process that mimics how
a human might complete a task using a shell or scripting environment.
For example, a workflow might involve generating content into files,
transforming or replacing parts of that content, aggregating results, and
applying formatting — all before producing a final output. By giving the LLM
both a plan and access to command-level operations, workflows unlock a higher
level of automation and creativity within your file system.

'run' treats the files in your input folder as a queue: each file's content is
loaded, passed to the workflow, and the result is saved to the output folder — one
output per file. 

The command support mounting of AWS S3 bucket. Use s3:// prefix
prefix to direct the utility (e.g. s3://bucket/path).

  echo "What ..." | iq ask -d s3://my/example -o s3://my/result

Processing a large number of files may require the ability to start, stop, and
resume the utility reliably. To support this, you can use the --mutable flag,
which removes each input file immediately after it has been successfully processed.
This enables fault-tolerant, resumable execution by ensuring already-processed
files are skipped on subsequent runs.


See more info https://github.com/fogfish/iq
	`,
	Example: `
	iq run -p task.yml -o ./output
	`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          withUsage(run),
}

func run(cmd *cobra.Command, args []string) error {
	fPrintf(os.Stderr, " 📂 running workflow with files in %s\n", runDir)
	if runMutable {
		fPrintf(os.Stderr, " ‼️ removes each input file immediately after it has been processed.\n")
	}

	spinner := createSpinner()
	defer spinner.Finish()

	q, err := createSpool(runDir, runOut, runMutable, runStrict)
	if err != nil {
		return err
	}

	agt, req, err := agentForTasks(runWorkDir)
	if err != nil {
		return err
	}

	q.ForEachFile(context.Background(), "/",
		func(ctx context.Context, path string, b []byte) ([]byte, error) {
			spinner.Describe(ellipses(path))
			defer respinner(spinner)

			req.Set(spec.YAML_BLOB, string(b))
			reply, err := agt.PromptOnce(ctx, req)
			if err != nil {
				return nil, err
			}

			return reply, nil
		})

	return nil
}
