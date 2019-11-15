package main

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestAccumulateDailyReport(t *testing.T) {
	var drs dailyReport = make(dailyReport)

	drs.accumulateTime("1", "p1", time.Second)
	drs.accumulateTime("1", "p2", time.Second)
	drs.accumulateTime("1", "p2", time.Second)
	drs.accumulateTime("2", "p1", time.Second)
	drs.accumulateTime("2", "p2", time.Second)
	drs.accumulateTime("2", "p2", time.Second)

	for _, day := range []int{1, 2} {
		dstr := strconv.Itoa(day)

		if _, ok := drs[dstr]; !ok {
			t.Error("report for", dstr, "not accumulated")
		} else {
			for _, pr := range []int{1, 2} {

				prs := "p" + strconv.Itoa(pr)

				if _, ok := drs[dstr][prs]; !ok {
					t.Error("report for", dstr, prs, "not accumulated")
				} else {
					if drs[dstr][prs] != time.Duration(pr)*time.Second {
						t.Error("report for", dstr, prs, "!=", pr, "seconds")
					}
				}
			}
		}
	}
}

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

	go scheduler(ctx, &wg, time.Second*10, func(context.Context, time.Duration) error { return nil })

	start := time.Now()
	cancel()
	wg.Wait()

	if time.Since(start) > time.Second {
		t.Errorf("scheduler didn't stop when context cancelled")
	}
}

func TestSchedulerPeriod(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	funcCalled := make(chan struct{})

	ct := int32(2)

	f := func(context.Context, time.Duration) error {
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
