package main

import (
	"context"
	"log"
	"os"

	"github.com/fogfish/iq/internal/blueprint"
	"github.com/kshard/chatter"
	"github.com/kshard/chatter/aio"
	"github.com/kshard/chatter/provider/autoconfig"
)

func main() {
	ctx := context.Background()

	bp, err := blueprint.New(ctx, "../../../examples/01_chain/run.yml", &factory{})
	if err != nil {
		log.Fatalf("failed to load blueprint: %v", err)
	}

	// Use Run() to execute the entrypoint job
	result, err := bp.Run(ctx, "The new laptop model features a 3.5 GHz octa-core processor, 16GB of RAM, and a 1TB NVMe SSD.")
	log.Printf("==> %v\n|%v\n", err, result)
}

type factory struct{}

func (f *factory) LLM(name string) (chatter.Chatter, error) {
	llm, err := autoconfig.FromNetRC("iq")
	if err != nil {
		return nil, err
	}
	llm = aio.NewJsonLogger(os.Stderr, llm)
	return llm, nil
}
