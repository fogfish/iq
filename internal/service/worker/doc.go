//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/iq
//

package worker

// Package worker provides a builder for creating configured conduits with blueprint processors.
// It assembles an iosystem conduit (pipeline) with processors based on blueprints.
//
// Basic usage:
//
//	// Load blueprint
//	factory := createLLMFactory()
//	bp, err := blueprint.New(ctx, "workflow.yml", factory)
//	if err != nil {
//	    return err
//	}
//
//	// Build conduit with blueprint processor
//	conduit, err := worker.New().
//	    Blueprint(bp).
//	    Build()
//	if err != nil {
//	    return err
//	}
//
//	// Run with source and sink
//	stats, err := conduit.Run(ctx, source, sink)
//
// With all options:
//
//	conduit, err := worker.New().
//	    Blueprint(bp).
//	    Job("process").              // Use specific job from blueprint
//	    Concurrency(4).              // Parallel processing
//	    ErrorMode(conduit.SkipError). // Continue on errors
//	    Progress(progressCallback).   // Progress updates
//	    Metrics(metricsCallback).     // Metrics updates
//	    Build()
//
// Convenience Run method:
//
//	// Build and run in one step
//	stats, err := worker.New().
//	    Blueprint(bp).
//	    Run(ctx, source, sink)
//
// CLI integration pattern:
//
//	func myCmd() *cobra.Command {
//	    var bpFile string
//	    var jobName string
//	    var concurrency int
//
//	    cmd := &cobra.Command{
//	        RunE: func(cmd *cobra.Command, args []string) error {
//	            // Create LLM factory
//	            factory := &myFactory{}
//
//	            // Load blueprint
//	            bp, err := blueprint.New(ctx, bpFile, factory)
//	            if err != nil {
//	                return err
//	            }
//
//	            // Create source and sink
//	            src := createSource(args)
//	            snk := createSink(output)
//
//	            // Build and run conduit
//	            stats, err := worker.New().
//	                Blueprint(bp).
//	                Job(jobName).
//	                Concurrency(concurrency).
//	                Run(ctx, src, snk)
//
//	            if err != nil {
//	                return err
//	            }
//
//	            fmt.Printf("Processed %d documents\n", stats.DocsProcessed)
//	            return nil
//	        },
//	    }
//
//	    cmd.Flags().StringVar(&bpFile, "blueprint", "", "Blueprint file")
//	    cmd.Flags().StringVar(&jobName, "job", "", "Job name")
//	    cmd.Flags().IntVar(&concurrency, "concurrency", 1, "Parallel workers")
//
//	    return cmd
//	}
