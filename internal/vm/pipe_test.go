package vm_test

import (
	"context"
	"testing"

	"github.com/fogfish/golem/pipe/v2"
)

func TestPipeDependency(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := pipe.Seq(1, 2, 3)
	out, _ := pipe.Map(ctx, ch, pipe.Lift(func(x int) (int, error) {
		return x * 2, nil
	}))

	var results []int
	for v := range out {
		results = append(results, v)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
}
