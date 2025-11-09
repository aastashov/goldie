package scheduler

import (
	"context"
	"time"

	"github.com/go-co-op/gocron"
)

type JobFunc func(ctx context.Context)

type Scheduler struct {
	s    *gocron.Scheduler
	ctx  context.Context
	jobs []struct {
		spec string
		fn   JobFunc
	}
}

func New(ctx context.Context, loc *time.Location) *Scheduler {
	return &Scheduler{s: gocron.NewScheduler(loc), ctx: ctx}
}

func (sch *Scheduler) Add(spec string, fn JobFunc) {
	sch.jobs = append(sch.jobs, struct {
		spec string
		fn   JobFunc
	}{spec: spec, fn: fn})
}

func (sch *Scheduler) Start() {
	for _, job := range sch.jobs {
		sch.s.Cron(job.spec).Do(func(fn JobFunc) {
			select {
			case <-sch.ctx.Done():
				return
			default:
				fn(sch.ctx)
			}
		}, job.fn)
	}
	sch.s.StartAsync()

	<-sch.ctx.Done()
	sch.s.Stop()
}
