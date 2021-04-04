package engine

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const (
	configPath   = "../bin/cfg.json"
	balancePath  = "../bin/balance.json"
	testProcess  = "../bin/test_process"
	testProcess1 = "../bin/test_process1"
	testProcess2 = "../bin/test_process2"
)

func BenchmarkCheckProcesses(b *testing.B) {

	ph := NewProcessHunter(time.Hour, balancePath, time.Hour, nil, configPath)

	err := ph.LoadConfig()

	if err != nil {
		b.Error("Error loading config file", configPath, err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ph.checkProcesses(context.Background(), time.Second)
	}
}

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

func TestIsValidDaySpecification(t *testing.T) {
	valid := []string{
		"*",
		"mon", "tue", "wed", "thu", "fri", "sat", "sun",
		"mon tue wed thu fri sat sun",
		"2019-12-22",
		"2019-1-1",
		"2019-01-01",
		"2019-01-01 2019-1-1 2018-2-3",
		"2019-01-01 mon"}

	for _, v := range valid {
		if !isValidDaySpecification(v) {
			t.Error("couldn't recognize", v, "as a valid week day(s) or date(s) specification")
		}
	}

	invalid := []string{
		"Mon", "MOn", "MON",
		"", "**",
		"* mon",
		"mon *",
		"pon", "vto",
		"pon vto wed thu fri sat sun",
		"2019/2/2",
		"yy-mm-dd",
		"2019-031-01",
		"2039-01-301",
		"20319-01-01"}
	for _, inv := range invalid {
		if isValidDaySpecification(inv) {
			t.Error("accepted", inv, "as a valid week days string")
		}
	}

}
func TestIsValidDailyLimitsFormat(t *testing.T) {
	// nothing much to test here - check TestIsValidDaySpecification
}

func TestIsValidBlackoutFormat(t *testing.T) {

	valid := []string{
		"..12:00",
		"00:00..5:30",
		"0:00..05:03",
		"12:00..",
		".."}

	for _, v := range valid {
		if !isValidBlackoutFormat(BlackOut{"*": []string{v}}) {
			t.Error("couldn't recognize", v, "as a valid blackout spec")
		}
	}

	invalid := []string{
		"..25:00",
		"..01:60",
		"00:00...5:30",
		"0:00..24:00"}
	for _, inv := range invalid {
		if isValidBlackoutFormat(BlackOut{"*": []string{inv}}) {
			t.Error("accepted", inv, "as a valid blackout spec")
		}
	}
}

func TestEvalDailyLimit(t *testing.T) {
	dl := DailyLimits{
		"*":                     time.Second,
		"mon tue":               time.Minute * 2,
		"mon":                   time.Minute,
		"1972-10-16 1973-05-17": time.Hour * 2,
		"1972-10-16":            time.Hour,
	}

	if evalDailyLimit("2019-12-21", "wed", dl) != time.Second {
		t.Error("wrong daily limit when the day is not listed in a group or individually, but matched by \"*\"")
	}

	// week days
	if evalDailyLimit("2019-12-21", "mon", dl) != time.Minute {
		t.Error("wrong daily limit when day of week is individually specified")
	}

	if evalDailyLimit("2019-12-21", "tue", dl) != time.Minute*2 {
		t.Error("wrong daily limit when day of week is listed in a group")
	}

	// dates
	if evalDailyLimit("1972-10-16", "wed", dl) != time.Hour {
		t.Error("wrong daily limit when date is individually specified")
	}

	if evalDailyLimit("1973-05-17", "wed", dl) != time.Hour*2 {
		t.Error("wrong daily limit when date is listed in a group")
	}

	// no match
	dl = DailyLimits{"tue": time.Second}
	if evalDailyLimit("2019-12-21", "mon", dl) != noLimit {
		t.Error("wrong daily limit when time limit cannot be evaluated")
	}
}

func TestCheckProcessNoConfig(t *testing.T) {
	ph := NewProcessHunter(time.Second, "", time.Hour, nil, "")

	err := ph.checkProcesses(context.Background(), time.Second)

	if err != nil {
		t.Error("checkProcess() failed", err)
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

	ph := NewProcessHunter(time.Second, "", time.Hour, f, configPath)

	err = ph.LoadConfig()
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

	collateral := false
	for _, k := range killed {
		target := false
		for _, cmd := range cmds {
			if k == cmd.Process.Pid {
				target = true
			}
		}
		if !target {
			collateral = true
			break
		}
	}

	if collateral {
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
	const (
		checkPeriod = time.Second * 2
		timeOut     = time.Second * 3
		savePeriod  = time.Second * 2
		path        = "tmp.balance.json"
	)

	ph := NewProcessHunter(checkPeriod, path, savePeriod, nil, "")

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

	ph := NewProcessHunter(time.Second, balancePath, time.Hour, nil, configPath)

	err := ph.LoadConfig()
	if err != nil {
		t.Error("Error loading config file", configPath, err)
	}

	ph.balance.add("1", "p1", time.Second)
	ph.balance.add("1", "p2", time.Second)
	ph.balance.add("1", "p2", time.Second)
	ph.balance.add("2", "p1", time.Second)
	ph.balance.add("2", "p2", time.Second)
	ph.balance.add("2", "p2", time.Second)

	err = ph.SaveBalance()
	if err != nil {
		t.Error("Error saving balance to file", balancePath, err)
	}

	ph.balance.add("3", "p1", time.Second)

	err = ph.LoadBalance()
	if err != nil {
		t.Error("Error loading balance from file", balancePath, err)
	}

	if len(ph.balance) != 2 {
		t.Error("Read", len(ph.balance), "days, expected 2")
	}
}

func testConfigLoadedCorrectly(t *testing.T, ph *ProcessHunter) bool {
	if len(ph.limits) != 3 {
		t.Error(len(ph.limits), "limits, expected 3")
		return false
	}

	if len(ph.limits) != 3 ||
		len(ph.limits[0].PG) != 2 ||
		ph.limits[0].PG[0] != "test_process" ||
		ph.limits[0].PG[1] != "test_process.exe" ||
		!reflect.DeepEqual(ph.limits[0].DL, DailyLimits{"mon tue wed": time.Second, "1999-12-25": time.Second}) ||
		len(ph.limits[0].BO) != 2 ||
		len(ph.limits[1].BO) != 1 ||
		len(ph.limits[2].BO) != 0 ||
		len(ph.limits[0].BO["mon"]) != 1 ||
		len(ph.limits[1].BO["*"]) != 3 ||
		ph.limits[1].BO["*"][0] != "..12:00" {
		t.Error("Config file", configPath, "not read correctly")
		return false
	}
	return true
}
func TestReloadConfigIfNeeded(t *testing.T) {
	ph := NewProcessHunter(time.Second, "", time.Hour, nil, configPath)

	err := ph.LoadConfig()
	if err != nil {
		t.Error("Error loading config file", configPath, err)
	}

	ph.limits = nil
	b, err := ph.reloadConfigIfNeeded()
	if err != nil {
		t.Error("Error running reloadConfigIfNeeded", err)
	}
	if b == true {
		t.Error("Unexpectedly reloaded config file")
	}
	if ph.limits != nil {
		t.Error("Unexpectedly modified ph.limits")
	}

	os.Chtimes(ph.cfgPath, time.Now(), time.Now())

	b, err = ph.reloadConfigIfNeeded()
	if err != nil {
		t.Error("Error running reloadConfigIfNeeded", err)
	}
	if b != true {
		t.Error("Didn't reload config file")
	}
	if testConfigLoadedCorrectly(t, ph) != true {
		t.Error("didn't reload config file correctly")
	}
}

func TestSetConfig(t *testing.T) {
	const cfg = `
[
    {
        "processes": [
            "p1",
            "p2"
        ],
        "limits": {
            "mon": "1s"
        }
    },
    {
        "processes": [
            "p3"
        ],
        "limits": {
            "*": "1h"
        }
    }
]`

	const tmpcfg = "tmp_cfg.json"

	err := os.Remove(tmpcfg)
	if err != nil {
		if !os.IsNotExist(err) {
			t.Error("cannot remove:", tmpcfg)
		}
	}

	ph := NewProcessHunter(time.Second, "", time.Hour, nil, tmpcfg)

	err = ph.SetConfig([]byte(cfg))
	if err != nil {
		t.Error("Couldnot set config:", cfg)
	}

	limits, _ := ph.GetLimits()
	if len(limits) != 2 ||
		len(limits[0].PG) != 2 ||
		limits[0].PG[0] != "p1" ||
		limits[0].PG[1] != "p2" ||
		limits[0].DL["mon"] != time.Second ||
		len(limits[1].PG) != 1 ||
		limits[1].PG[0] != "p3" ||
		limits[1].DL["*"] != time.Hour {
		t.Error("config not set correctly")
	}

	b, err := ioutil.ReadFile(ph.cfgPath)
	if err != nil {
		t.Error("cannot read config file", ph.cfgPath)
	}

	if !bytes.Equal(b, []byte(cfg)) {
		t.Error("config file", ph.cfgPath, "was not saved correctly")
	}

	err = os.Remove(tmpcfg)
	if err != nil {
		t.Error("cannot remove:", tmpcfg)
	}
}

func TestLoadConfig(t *testing.T) {
	ph := NewProcessHunter(time.Second, "", time.Hour, nil, configPath)
	ph.limits = []ProcessGroupDailyLimit{
		{[]string{"1"}, DailyLimits{"*": time.Minute}, BlackOut{"mon": {"..08:00"}}},
		{[]string{"2"}, DailyLimits{"*": time.Minute}, BlackOut{"tue wed": {"12:00..14:00"}}},
		{[]string{"3"}, DailyLimits{"*": time.Minute}, BlackOut{"*": {"22:00..", "..12:00"}, "tue": {"06:00..12:00"}}},
		{[]string{"4"}, DailyLimits{"*": time.Minute}, BlackOut{}},
	}

	err := ph.LoadConfig()
	if err != nil {
		t.Error("Error loading config file", configPath, err)
	}

	if testConfigLoadedCorrectly(t, ph) != true {
		t.Error("didn't load config file correctly")
	}
}

func TestDateToText(t *testing.T) {
	dts := toText(time.Date(1972, time.October, 16, 0, 0, 0, 0, time.UTC))
	if dts != "1972-10-16" {
		t.Error("date formatted incorrectly, expected 1972-10-16, got", dts)
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

	go scheduler(ctx, &wg, time.Second*10, nil, func(context.Context, time.Duration) error { return nil })

	start := time.Now()
	cancel()
	wg.Wait()

	if time.Since(start) > time.Second {
		t.Errorf("scheduler didn't stop when context cancelled")
	}
}

func TestSchedulerRunNow(t *testing.T) {
	force := make(chan struct{})
	defer close(force)

	ct := int32(0)
	f := func(context.Context, time.Duration) error {
		atomic.AddInt32(&ct, 1)
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)

	go scheduler(ctx, &wg, time.Second*10, force, f)
	force <- struct{}{}
	cancel()
	wg.Wait()

	if ct != 2 {
		t.Errorf("function not run when now is populated")
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

	var wg sync.WaitGroup
	wg.Add(1)
	go scheduler(ctx, &wg, time.Second, nil, f)

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

func TestMarshalDailyLimit(t *testing.T) {
	d := make(DailyLimits)

	d["second"] = time.Second
	d["minute"] = time.Minute

	b, err := d.MarshalJSON()

	if err != nil {
		t.Error("d.MarshalJSON() failed:", err)
	}

	d = make(DailyLimits)

	err = d.UnmarshalJSON(b)
	if err != nil {
		t.Error("d.UnmarshalJSON(b) failed:", err)
	}

	if len(d) != 2 || d["second"] != time.Second || d["minute"] != time.Minute {
		t.Error("unmarshaled DailyLimits is not correct:", d)
	}
}
