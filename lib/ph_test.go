package lib

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const configPath = "../testdata/cfg.json"
const balancePath = "../testdata/balance.json"

func TestPersistBalance(t *testing.T) {

	ph := NewProcessHunter(nil, time.Second, nil)

	err := ph.LoadConfig(configPath)

	if err != nil {
		t.Error("Error loading config file", configPath, err)
	}

	ph.balance.add("1", "p1", time.Second)
	ph.balance.add("1", "p2", time.Second)
	ph.balance.add("1", "p2", time.Second)
	ph.balance.add("2", "p1", time.Second)
	ph.balance.add("2", "p2", time.Second)
	ph.balance.add("2", "p2", time.Second)

	err = ph.SaveBalance(balancePath)

	ph.balance.add("3", "p1", time.Second)

	if err != nil {
		t.Error("Error saving balance to file", balancePath, err)
	}

	err = ph.LoadBalance(balancePath)

	if err != nil {
		t.Error("Error loading balance from file", balancePath, err)
	}

	if len(ph.balance) != 2 {
		t.Error("Read", len(ph.balance), "days, expected 2")
	}
}
func TestLoadConfig(t *testing.T) {
	l := make(DailyTimeLimit)
	l["should_disappear"] = time.Minute
	ph := NewProcessHunter(l, time.Second, nil)

	err := ph.LoadConfig(configPath)

	if err != nil {
		t.Error("Error loading config file", configPath, err)
	}

	if len(ph.limits) != 3 {
		t.Error("Read", len(ph.limits), "limits, expected 3")
	}

	if ph.limits["test_process"] != time.Second ||
		ph.limits["test_process.exe"] != time.Second ||
		ph.limits["FortniteClient-Win64-Shipping.exe"] != 120*time.Second {
		t.Error("Config file", configPath, "not read correctly")
	}
}

func TestCheckRunningProcesses(t *testing.T) {
	l := make(DailyTimeLimit)
	l["zsh"] = time.Second
	ph := NewProcessHunter(l, time.Second, nil)

	err := ph.checkProcesses(context.Background(), time.Second*2)

	if err != nil {
		t.Error("checkProcesses returned unexpected error", err)
	}

	if len(ph.balance) == 0 {
		t.Error("checkProcess produced zero balance")
	}
}

func TestDateToText(t *testing.T) {
	dts := toText(time.Date(1972, time.October, 16, 0, 0, 0, 0, time.UTC))
	if dts != "1972-10-16" {
		t.Error("date formatted incorectly, expected 1972-10-16, got", dts)
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
	go scheduler(context.Background(), &wg, time.Second, f)

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
