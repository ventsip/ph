package main

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDateToText(t *testing.T) {
	dts := toText(time.Date(1972, time.October, 16, 0, 0, 0, 0, time.UTC))
	if dts != "1972-10-16" {
		t.Error("date formatted incorectly, expected 1972-10-16, got", dts)
	}
}
func TestCheckOverTime(t *testing.T) {
	var drs dailyTimeBalance = make(dailyTimeBalance)

	drs.add("1", "p1", time.Second)
	drs.add("1", "p2", time.Second)
	drs.add("1", "p2", time.Second)
	drs.add("2", "p1", time.Second)
	drs.add("2", "p2", time.Second)
	drs.add("2", "p2", time.Second)

	if drs.isOverTime("1", "p1", time.Second) == true {
		t.Error("process p1 for day 1 must not report overtime")
	}
	if drs.isOverTime("1", "p2", time.Second) == false {
		t.Error("process p2 for day 1 must report overtime")
	}
	if drs.isOverTime("2", "p1", time.Second) == true {
		t.Error("process p1 for day 2 must not report overtime")
	}
	if drs.isOverTime("2", "p2", time.Second) == false {
		t.Error("process p2 for day 2 must report overtime")
	}
}
func TestAddToDailyTimeBalance(t *testing.T) {
	var drs dailyTimeBalance = make(dailyTimeBalance)

	drs.add("1", "p1", time.Second)
	drs.add("1", "p2", time.Second)
	drs.add("1", "p2", time.Second)
	drs.add("2", "p1", time.Second)
	drs.add("2", "p2", time.Second)
	drs.add("2", "p2", time.Second)

	for _, day := range []int{1, 2} {
		dstr := strconv.Itoa(day)

		if _, ok := drs[dstr]; !ok {
			t.Error("time balance for day", dstr, "was not created")
		} else {
			for _, pr := range []int{1, 2} {

				prs := "p" + strconv.Itoa(pr)

				if _, ok := drs[dstr][prs]; !ok {
					t.Error("time balance for day", dstr, "and process", prs, "not accumulated")
				} else {
					if drs[dstr][prs] != time.Duration(pr)*time.Second {
						t.Error("time balance for day", dstr, "and process", prs, "!=", pr, "seconds")
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

	go scheduler(ctx, &wg, time.Second*10, func(context.Context, time.Duration, []string) error { return nil })

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

	f := func(context.Context, time.Duration, []string) error {
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
