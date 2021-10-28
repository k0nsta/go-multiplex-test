package semaphore

import (
	"errors"
	"time"
)

var ErrTimeout = errors.New("semaphore timeout exceeded")

type Semaphore struct {
	timeout time.Duration
	semChan chan struct{}
}

// New semaphore constructor
func New(maxConcur uint, timeout time.Duration) *Semaphore {
	return &Semaphore{
		timeout: timeout,
		semChan: make(chan struct{}, maxConcur),
	}
}

func (s *Semaphore) Acquire() error {
	select {
	case s.semChan <- struct{}{}:
		return nil
	case <-time.After(s.timeout):
		return ErrTimeout
	}
}

func (s *Semaphore) Release() {
	<-s.semChan
}
