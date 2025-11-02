package cmd

import (
	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/conduit"
	"github.com/fogfish/iq/internal/iosystem/processor"
	"github.com/fogfish/iq/internal/service/llm"
	"github.com/fogfish/iq/internal/service/sink"
	"github.com/fogfish/iq/internal/service/source"
	"github.com/fogfish/iq/internal/service/worker"
	"github.com/kshard/chatter"
	"github.com/spf13/cobra"
)

//------------------------------------------------------------------------------

var fmodel optsLLM

type optsLLM struct {
	profile  string
	model    string
	maxEpoch int
	maxUsage chatter.Usage
	think    bool
	debug    bool
}

func (opts *optsLLM) apply(cmd *cobra.Command) {
	f := cmd.PersistentFlags()

	f.StringVarP(&opts.profile, "profile", "p", "iq",
		"Name of the LLM provider profile (from ~/.netrc)")

	f.StringVarP(&opts.model, "llm-id", "m", "",
		"LLM model to use (overrides model set in ~/.netrc)")

	f.IntVar(&opts.maxEpoch, "llm-max-epoch", 0,
		"Maximum number of refinement attempts before giving up")

	f.IntVar(&opts.maxUsage.InputTokens, "llm-max-input-tokens", 0,
		"Limit on total input tokens before aborting")

	f.IntVar(&opts.maxUsage.ReplyTokens, "llm-max-reply-tokens", 0,
		"Limit on total reply tokens before aborting")

	f.BoolVar(&opts.debug, "llm-debug", false,
		"Enable debug logging")

	f.BoolVar(&opts.think, "llm-think", false,
		"Show intermediate reasoning steps")
}

func (opts *optsLLM) build() (chatter.Chatter, error) {
	return llm.New().
		Profile(opts.profile, opts.model).
		Debug(opts.debug).
		Think(opts.think).
		Quota(opts.maxEpoch, opts.maxUsage).
		Build()
}

//------------------------------------------------------------------------------

var fagent optsAgent

type optsAgent struct {
	splitter      string
	splitterChunk int
	splitterChars string
	agent         string
	json          bool
}

func (opts *optsAgent) apply(cmd *cobra.Command) {
	f := cmd.PersistentFlags()

	f.StringVarP(&opts.agent, "agent", "a", "",
		"Path to agent definition yaml file or simple markdown file")

	f.StringVar(&opts.splitter, "splitter", "none",
		"Split input file into sentence, paragraph or chunk")

	f.IntVar(&opts.splitterChunk, "splitter-chunk", 1024,
		"Chunk size for splitter")

	f.StringVar(&opts.splitterChars, "splitter-chars", "",
		"Sequence of characters used by splitter as delimiter")

	f.BoolVar(&opts.json, "json", false,
		"Display output as formatted, colored JSON")
}

func (opts *optsAgent) build(llm chatter.Chatter) (*conduit.Conduit, error) {
	return worker.New().
		// TODO: Runtime config
		Runtime().
		Splitter(processor.ChunkConfig{
			Strategy:       opts.splitter,
			ChunkSize:      opts.splitterChunk,
			DelimiterChars: opts.splitterChars,
		}).
		Workflow(opts.agent, llm).
		Jsonify(opts.json).
		Build()
}

//------------------------------------------------------------------------------

var finput optsInput

type optsInput struct {
	dir   string
	merge bool
}

func (opts *optsInput) apply(cmd *cobra.Command) {
	f := cmd.PersistentFlags()

	f.StringVarP(&opts.dir, "input-dir", "I", "",
		"Input directory or S3 URI containing files to process")

	f.BoolVar(&opts.merge, "merge", false,
		"Combine all input files into a single document before processing")
}

func (opts *optsInput) build(files []string) (iosystem.Source, error) {
	return source.New().
		Files(".", files...).
		Path(opts.dir).
		Merge(opts.merge).
		Stdin().
		None().
		Build()
}

//------------------------------------------------------------------------------

var freply optsReply

type optsReply struct {
	dir    string
	file   string
	quiet  bool
	silent bool
}

func (opts *optsReply) apply(cmd *cobra.Command) {
	f := cmd.PersistentFlags()

	f.StringVarP(&opts.dir, "output-dir", "O", "",
		"Output directory or S3 URI to write files to")

	f.StringVarP(&opts.file, "output", "o", "",
		"Path to the output file")

	f.BoolVarP(&opts.quiet, "quiet", "q", false,
		"Suppress all non-error output")

	f.BoolVar(&opts.silent, "silent", false,
		"Suppress all output, including errors")
}

func (opts *optsReply) build() (iosystem.Sink, error) {
	return sink.New().
		File(opts.file).
		Path(opts.dir).
		Stdout(!opts.quiet && !opts.silent).
		Build()
}
