package sink_test

import (
	"context"
	"strings"
	"testing"

	"github.com/fogfish/iq/internal/iosystem"
	"github.com/fogfish/iq/internal/iosystem/sink"
	"github.com/fogfish/it/v2"
)

func TestStdoutSink(t *testing.T) {
	s := sink.NewStdoutSink()
	defer s.Close()

	ctx := context.Background()
	doc := iosystem.NewDocument("test.txt", strings.NewReader("test content"))

	err := s.Write(ctx, doc)
	it.Then(t).Should(it.Nil(err))
}

func TestStdoutSink_MultipleWrites(t *testing.T) {
	s := sink.NewStdoutSink()
	defer s.Close()

	ctx := context.Background()

	for i := 1; i <= 3; i++ {
		doc := iosystem.NewDocument("test.txt", strings.NewReader("content"))
		err := s.Write(ctx, doc)
		it.Then(t).Should(it.Nil(err))
	}
}

func TestStdoutSink_Close(t *testing.T) {
	s := sink.NewStdoutSink()

	err := s.Close()
	it.Then(t).Should(it.Nil(err))

	// Close should be idempotent
	err = s.Close()
	it.Then(t).Should(it.Nil(err))
}
