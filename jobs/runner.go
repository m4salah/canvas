// Package jobs has a Runner that can run registered jobs in parallel.
package jobs

import (
	"context"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"canvas/messaging"
	"canvas/model"
	"canvas/util"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Runner runs jobs.
type Runner struct {
	emailer        *messaging.Emailer
	jobs           map[string]Func
	queue          *messaging.Queue
	jobCount       *prometheus.CounterVec
	jobDurations   *prometheus.CounterVec
	runnerReceives *prometheus.CounterVec
}

type NewRunnerOptions struct {
	Emailer *messaging.Emailer
	Metrics *prometheus.Registry
	Queue   *messaging.Queue
}

func NewRunner(opts NewRunnerOptions) *Runner {
	if opts.Metrics == nil {
		opts.Metrics = prometheus.NewRegistry()
	}
	jobCount := promauto.With(opts.Metrics).NewCounterVec(prometheus.CounterOpts{
		Name: "app_jobs_total",
	}, []string{"name", "success"})

	jobDurations := promauto.With(opts.Metrics).NewCounterVec(prometheus.CounterOpts{
		Name: "app_job_duration_seconds_total",
	}, []string{"name", "success"})

	runnerReceives := promauto.With(opts.Metrics).NewCounterVec(prometheus.CounterOpts{
		Name: "app_job_runner_receives_total",
	}, []string{"success"})
	return &Runner{
		emailer:        opts.Emailer,
		jobs:           map[string]Func{},
		queue:          opts.Queue,
		jobCount:       jobCount,
		jobDurations:   jobDurations,
		runnerReceives: runnerReceives,
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
		r.runnerReceives.WithLabelValues("false").Inc()
		slog.Info("Error receiving message", util.ErrAttr(err))
		// Sleep a bit to not hammer the queue if there's an error with it
		time.Sleep(time.Second)
		return
	}

	// If there was no message there is nothing to do
	if m == nil {
		r.runnerReceives.WithLabelValues("true").Inc()
		return
	}

	name, ok := (*m)["job"]
	if !ok {
		r.runnerReceives.WithLabelValues("false").Inc()
		slog.Info("Error getting job name from message")
		return
	}

	job, ok := r.jobs[name]
	if !ok {
		r.runnerReceives.WithLabelValues("false").Inc()
		slog.Info("No job with this name", slog.String("name", name))
		return
	}

	r.runnerReceives.WithLabelValues("true").Inc()
	wg.Add(1)
	go func() {
		defer wg.Done()

		log := slog.With(slog.String("name", name))

		defer func() {
			if rec := recover(); rec != nil {
				r.jobCount.WithLabelValues(name, "false").Inc()
				log.Info("Recovered from panic in job", slog.Any("recover", rec))
			}
		}()

		before := time.Now()
		err := job(ctx, *m)
		duration := time.Since(before)

		success := strconv.FormatBool(err == nil)
		r.jobCount.WithLabelValues(name, success).Inc()
		r.jobDurations.WithLabelValues(name, success).Add(duration.Seconds())

		if err != nil {
			log.Info("Error running job", util.ErrAttr(err))
			return
		}
		log.Info("Successfully ran job", slog.Duration("duration", duration))

		// We use context.Background as the parent context instead of the existing ctx, because if we've come
		// this far we don't want the deletion to be cancelled.
		deleteCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := r.queue.Delete(deleteCtx, receiptID); err != nil {
			log.Error("Error deleting message, job will be repeated", util.ErrAttr(err))
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
