package lib

import (
	"context"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const configPath = "../bin/cfg.json"
const balancePath = "../bin/balance.json"
const testProcess = "../bin/test_process"
const testProcess1 = "../bin/test_process1"
const testProcess2 = "../bin/test_process2"

func startTestProcesses(t *testing.T, names ...string) (cmds [](*exec.Cmd), err error) {
	for _, n := range names {
		cmd := exec.Command(n)
		e := cmd.Start()
		if e != nil {
			t.Error("Cannot start test process", n)
			err = e
		}
		cmds = append(cmds, cmd)
	}
	return
}

func stopTestProcesses(t *testing.T, cmds [](*exec.Cmd)) (err error) {
	for _, cmd := range cmds {
		e := cmd.Process.Kill()
		if e != nil {
			t.Error("Cannot stop test process", cmd)
			err = e
		}
	}
	return
}

func TestIsValidDailyLimitsFormat(t *testing.T) {

	valid := []string{
		"*",
		"mon", "tue", "wed", "thu", "fri", "sat", "sun",
		"mon tue wed thu fri sat sun",
	}

	for _, v := range valid {
		if !isValidDailyLimitsFormat(DailyLimits{v: time.Second}) {
			t.Error("couldn't recognize", v, "as valid week days string")
		}
	}

	invalid := []string{
		"Mon", "MOn", "MON",
		"", "**",
		"* mon",
		"mon *",
		"pon", "vto",
		"pon vto wed thu fri sat sun",
	}
	for _, inv := range invalid {
		if isValidDailyLimitsFormat(DailyLimits{inv: time.Second}) {
			t.Error("accepted", inv, "as valid week days string")
		}
	}
}

func TestEvalDailyLimit(t *testing.T) {
	dl := DailyLimits{"*": time.Second, "mon tue": time.Minute, "mon": time.Hour}
	if evalDailyLimit("mon", dl) != time.Hour {
		t.Error("wrong daily limit when day is individually specified")
	}

	if evalDailyLimit("tue", dl) != time.Minute {
		t.Error("wrong daily limit when day is listed in a group")
	}

	if evalDailyLimit("wed", dl) != time.Second {
		t.Error("wrong daily limit when day is not listed in a group or individually, but matched by \"*\"")
	}

	dl = DailyLimits{"tue": time.Second}
	if evalDailyLimit("mon", dl) != time.Hour*25 {
		t.Error("wrong daily limit when time limit cannot be evaluated")
	}
}

func TestKillProcess(t *testing.T) {
	cmds, err := startTestProcesses(t, testProcess1, testProcess2)
	if err != nil {
		t.Error("Cannot start test processes", err)
	}

	var killed []int

	f := func(pid int) error {
		killed = append(killed, pid)
		return nil
	}

	ph := NewProcessHunter(nil, time.Second, "", time.Hour, f)

	err = ph.LoadConfig(configPath)
	if err != nil {
		t.Error("Error loading config file", configPath, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*4)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	ph.Run(ctx, &wg)
	wg.Wait()

	allKilled := true
	for _, cmd := range cmds {
		dead := false
		for _, k := range killed {
			if k == cmd.Process.Pid {
				dead = true
			}
		}
		if !dead {
			allKilled = false
			break
		}
	}
	if !allKilled {
		t.Error("One or more of the target processes was not killed")
	}

	colateral := false
	for _, k := range killed {
		target := false
		for _, cmd := range cmds {
			if k == cmd.Process.Pid {
				target = true
			}
		}
		if !target {
			colateral = true
			break
		}
	}

	if colateral {
		t.Error("Killed one or more wrong processes")
	}

	err = stopTestProcesses(t, cmds)
	if err != nil {
		t.Error("Cannot stop test processes", err)
	}
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
func TestPeriodicSaveBalance(t *testing.T) {
	const checkPeriod = time.Second * 2
	const timeOut = time.Second * 3
	const savePeriod = time.Second * 2
	const path = "tmp.balance.json"

	ph := NewProcessHunter(nil, checkPeriod, path, savePeriod, nil)

	err := os.Remove(path)
	if fileExists(path) {
		t.Error("cannot remove file", path)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeOut)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	ph.Run(ctx, &wg)
	wg.Wait()

	if !fileExists(path) {
		t.Error("balance file", path, "not created on-time")
	} else {
		err = os.Remove(path)
		if err != nil {
			t.Error("cannot cleanup file", path, ":", err)
		}
	}
}

func TestSaveBalance(t *testing.T) {

	ph := NewProcessHunter(nil, time.Second, "", time.Hour, nil)

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
	l := []ProcessGroupDailyLimit{
		{[]string{"1"}, DailyLimits{"*": time.Minute}},
		{[]string{"2"}, DailyLimits{"*": time.Minute}},
		{[]string{"3"}, DailyLimits{"*": time.Minute}},
		{[]string{"4"}, DailyLimits{"*": time.Minute}},
	}
	ph := NewProcessHunter(l, time.Second, "", time.Hour, nil)

	err := ph.LoadConfig(configPath)

	if err != nil {
		t.Error("Error loading config file", configPath, err)
	}

	if len(ph.limits) != 3 {
		t.Error("Read", len(ph.limits), "limits, expected 3")
	}

	if len(ph.limits) != 3 ||
		len(ph.limits[0].PG) != 2 ||
		ph.limits[0].PG[0] != "test_process" ||
		ph.limits[0].PG[1] != "test_process.exe" ||
		!reflect.DeepEqual(ph.limits[0].DL, DailyLimits{"mon tue wed": time.Second}) {
		t.Error("Config file", configPath, "not read correctly")
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

	funcCalled := make(chan struct{}, 100)

	ct := int32(2)

	f := func(context.Context, time.Duration) error {
		atomic.AddInt32(&ct, -1)
		if 0 == ct {
			funcCalled <- struct{}{}
		}

		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go scheduler(ctx, &wg, time.Second, f)

	timeout := time.NewTimer(time.Second * 2)
	defer timeout.Stop()

	select {
	case <-funcCalled:

	case <-timeout.C:
		t.Errorf("function not called on time")
	}
	cancel()
	wg.Wait()
}