package main

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestGetRunningProcessesContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	cancel()

	_, err := getRunningProcess(ctx)

	if err != ctx.Err() {
		t.Errorf("not responding to cancelled context")
	}
}

func TestSchedulerContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)

	go scheduler(ctx, &wg, time.Second*10, func(context.Context) error { return nil })

	start := time.Now()
	cancel()
	wg.Wait()

	if time.Since(start) > time.Second {
		t.Errorf("scheduler didn't stop when context cancelled")
	}
}

func TestSchedulerPeriod(t *testing.T) {

	ctx, _ := context.WithCancel(context.Background())

	funcCalled := make(chan struct{})

	ct := int32(2)

	f := func(context.Context) error {
		atomic.AddInt32(&ct, -1)
		if 0 == ct {
			funcCalled <- struct{}{}
		}

		return nil
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go scheduler(ctx, &wg, time.Second, f)

	timeout := time.NewTimer(time.Second * 2)
	defer timeout.Stop()

	select {
	case <-funcCalled:
		return
	case <-timeout.C:
		t.Errorf("function not called on time")
	}

	wg.Wait()
}
