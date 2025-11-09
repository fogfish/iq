//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package cmd

import (
	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/processor"
	"github.com/fogfish/iq/internal/progress"
	"github.com/fogfish/iq/internal/service/batch"
	"github.com/fogfish/iq/internal/service/llm"
	"github.com/fogfish/iq/internal/service/mcp"
	"github.com/fogfish/iq/internal/service/sink"
	"github.com/fogfish/iq/internal/service/source"
	"github.com/fogfish/iq/internal/service/worker"
	"github.com/fogfish/stream/spool"
	"github.com/kshard/chatter"
	"github.com/spf13/cobra"
)

//------------------------------------------------------------------------------

// Global options
var fglobal optsGlobal

type optsGlobal struct {
	debug  bool
	quiet  bool
	silent bool
}

func (opts *optsGlobal) apply(cmd *cobra.Command) {
	f := cmd.PersistentFlags()

	f.BoolVar(&opts.debug, "debug", false,
		"Enable debug logging")

	f.BoolVarP(&opts.quiet, "quiet", "q", false,
		"Suppress progress output (errors still shown)")

	f.BoolVar(&opts.silent, "silent", false,
		"Suppress all output, including errors")

}

func (opts *optsGlobal) reporter() *progress.Reporter {
	return progress.New(opts.quiet)
}

//------------------------------------------------------------------------------

// LLM options
var fmodel optsLLM

type optsLLM struct {
	profile  string
	model    string
	maxEpoch int
	maxUsage chatter.Usage
	think    bool
	cache    string
}

func (opts *optsLLM) apply(cmd *cobra.Command) {
	f := cmd.PersistentFlags()

	f.StringVarP(&opts.profile, "profile", "p", "base",
		"Name of the LLM provider profile (from ~/.iqrc)")

	f.StringVarP(&opts.model, "llm-id", "m", "",
		"LLM model to use (overrides model set in ~/.iqrc)")

	f.IntVar(&opts.maxEpoch, "llm-max-epoch", 0,
		"Maximum number of refinement attempts before giving up")

	f.IntVar(&opts.maxUsage.InputTokens, "llm-max-input-tokens", 0,
		"Limit on total input tokens before aborting")

	f.IntVar(&opts.maxUsage.ReplyTokens, "llm-max-reply-tokens", 0,
		"Limit on total reply tokens before aborting")

	f.BoolVar(&opts.think, "llm-think", false,
		"Show intermediate reasoning steps")

	f.StringVar(&opts.cache, "llm-cache", "",
		"Path to the LLM response cache database (enables caching)")
}

func (opts *optsLLM) build(reporter *progress.Reporter) (chatter.Chatter, error) {
	return llm.New().
		Profile(opts.profile, opts.model).
		Quota(opts.maxEpoch, opts.maxUsage).
		Debug(fglobal.debug).
		Think(opts.think).
		Cache(opts.cache).
		Reporter(reporter).
		Build()
}

//------------------------------------------------------------------------------

// Agent options
var fagent optsAgent

type optsAgent struct {
	file          string
	splitter      string
	splitterChunk int
	splitterChars string
	json          bool
}

func (opts *optsAgent) apply(cmd *cobra.Command) {
	f := cmd.PersistentFlags()

	f.StringVarP(&opts.file, "file", "f", "",
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

func (opts *optsAgent) build(llm chatter.Chatter, reporter *progress.Reporter) (*worker.ConduitWithReporter, error) {
	return worker.New().
		// TODO: ErrorMode
		Reporter(reporter).
		Runtime().
		Splitter(processor.ChunkConfig{
			Strategy:       opts.splitter,
			ChunkSize:      opts.splitterChunk,
			DelimiterChars: opts.splitterChars,
		}).
		Workflow(opts.file, llm).
		Jsonify(opts.json).
		Build()
}

//------------------------------------------------------------------------------

// Input options
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

// Output options
var freply optsReply

type optsReply struct {
	dir  string
	file string
}

func (opts *optsReply) apply(cmd *cobra.Command) {
	f := cmd.PersistentFlags()

	f.StringVarP(&opts.dir, "output-dir", "O", "",
		"Output directory or S3 URI to write files to")

	f.StringVarP(&opts.file, "output", "o", "",
		"Path to the output file")

}

func (opts *optsReply) build() (iosystem.Sink, error) {
	return sink.New().
		File(opts.file).
		Path(opts.dir).
		Stdout(!fglobal.quiet && !fglobal.silent).
		Build()
}

//------------------------------------------------------------------------------

// Batch/Spool options
var fspool optsSpool

type optsSpool struct {
	mutable bool
	strict  bool
}

func (opts *optsSpool) apply(cmd *cobra.Command) {
	f := cmd.Flags()

	f.BoolVar(&opts.mutable, "mutable", false,
		"Remove input files after successful processing (fail-safe mode)")

	f.BoolVar(&opts.strict, "strict", false,
		"Fail fast on the first error during processing")
}

func (opts *optsSpool) build() (*spool.Spool, error) {
	return batch.New().
		Reader(finput.dir).
		Writer(freply.dir).
		Mutable(opts.mutable).
		Strict(opts.strict).
		Build()
}

//------------------------------------------------------------------------------

// MCP Server options
var fservmcp optsServeMCP

type optsServeMCP struct{}

func (opts *optsServeMCP) apply(cmd *cobra.Command) {}

func (opts *optsServeMCP) build() (mcp.Server, error) {
	return mcp.New().
		Server(fagent.file).
		Build()
}
