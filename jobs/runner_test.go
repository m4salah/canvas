package jobs_test

import (
	"context"
	"testing"

	"github.com/matryer/is"

	"canvas/integrationtest"
	"canvas/jobs"
	"canvas/model"
)

type testRegistry map[string]jobs.Func

func (r testRegistry) Register(name string, fn jobs.Func) {
	r[name] = fn
}

func TestRunner_Start(t *testing.T) {
	integrationtest.SkipIfShort(t)

	t.Run("starts the runner and runs jobs until the context is cancelled", func(t *testing.T) {
		is := is.New(t)

		queue, cleanup := integrationtest.CreateQueue()
		defer cleanup()

		runner := jobs.NewRunner(jobs.NewRunnerOptions{
			Queue: queue,
		})

		ctx, cancel := context.WithCancel(context.Background())

		runner.Register("test", func(ctx context.Context, m model.Message) error {
			foo, ok := m["foo"]
			is.True(ok)
			is.Equal("bar", foo)

			cancel()
			return nil
		})

		err := queue.Send(context.Background(), model.Message{"job": "test", "foo": "bar"})
		is.NoErr(err)

		// This blocks until the context is cancelled by the job function
		runner.Start(ctx)

	})
}
