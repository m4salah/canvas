// Package jobs has a Runner that can run registered jobs in parallel.
package jobs

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"canvas/messaging"
	"canvas/model"
)

// Runner runs jobs.
type Runner struct {
	emailer *messaging.Emailer
	jobs    map[string]Func
	queue   *messaging.Queue
}

type NewRunnerOptions struct {
	Emailer *messaging.Emailer
	Queue   *messaging.Queue
}

func NewRunner(opts NewRunnerOptions) *Runner {
	return &Runner{
		emailer: opts.Emailer,
		jobs:    map[string]Func{},
		queue:   opts.Queue,
	}
}

// Func is the actual work to do in a job.
// The given context is the root context of the runner, which may be cancelled.
type Func = func(context.Context, model.Message) error

// Start the Runner, blocking until the given context is cancelled.
func (r *Runner) Start(ctx context.Context) {
	slog.Info("Starting")
	r.registerJobs()
	var wg sync.WaitGroup

	for {
		select {
		case <-ctx.Done():
			slog.Info("Stopping")
			wg.Wait()
			return
		default:
			r.receiveAndRun(ctx, &wg)
		}
	}
}

// receiveAndRun jobs.
func (r *Runner) receiveAndRun(ctx context.Context, wg *sync.WaitGroup) {
	m, receiptID, err := r.queue.Receive(ctx)
	if err != nil {
		slog.Info("Error receiving message", err)
		// Sleep a bit to not hammer the queue if there's an error with it
		time.Sleep(time.Second)
		return
	}

	// If there was no message there is nothing to do
	if m == nil {
		return
	}

	name, ok := (*m)["job"]
	if !ok {
		slog.Info("Error getting job name from message")
		return
	}

	job, ok := r.jobs[name]
	if !ok {
		slog.Info("No job with this name", slog.String("name", name))
		return
	}

	wg.Add(1)
	go func() {
		defer wg.Done()

		log := slog.With(slog.String("name", name))

		defer func() {
			if rec := recover(); rec != nil {
				log.Info("Recovered from panic in job", slog.Any("recover", rec))
			}
		}()

		before := time.Now()
		if err := job(ctx, *m); err != nil {
			log.Info("Error running job", err)
			return
		}
		after := time.Now()
		duration := after.Sub(before)
		log.Info("Successfully ran job", slog.Duration("duration", duration))

		// We use context.Background as the parent context instead of the existing ctx, because if we've come
		// this far we don't want the deletion to be cancelled.
		deleteCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := r.queue.Delete(deleteCtx, receiptID); err != nil {
			log.Info("Error deleting message, job will be repeated", err)
		}
	}()
}

// registry provides a way to Register jobs by name.
type registry interface {
	Register(name string, fn Func)
}

// Register implements registry.
func (r *Runner) Register(name string, j Func) {
	r.jobs[name] = j
}
