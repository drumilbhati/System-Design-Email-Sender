package scheduler

import (
	"log"
	"time"
)

type Job func()

type Scheduler struct {
	interval time.Duration
	job      Job
	stop     chan struct{}
}

func NewScheduler(interval time.Duration, job Job) *Scheduler {
	return &Scheduler{
		interval: interval,
		job:      job,
		stop:     make(chan struct{}),
	}
}

func (s *Scheduler) Start() {
	ticker := time.NewTicker(s.interval)
	go func() {
		log.Println("Scheduler started. First run in", s.interval)
		for {
			select {
			case <-ticker.C:
				log.Println("Scheduler triggered job")
				s.job()
			case <-s.stop:
				ticker.Stop()
				log.Println("Scheduler stopped")
				return
			}
		}
	}()
}

func (s *Scheduler) Stop() {
	close(s.stop)
}
