package main

import (
	"context"
	"log"
	"strconv"
	"sync"
	"time"

	ps "github.com/mitchellh/go-ps"
)

// scheduler runs the work function periodically (every period seconds)
func scheduler(ctx context.Context, wg *sync.WaitGroup, period time.Duration, work func(context.Context, time.Duration) error) {
	defer func() {
		if wg != nil {
			wg.Done()
		}
	}()

	ticker := time.NewTicker(period)
	defer ticker.Stop()

	err := work(ctx, period)
	if err != nil {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			work(ctx, period)
		}
	}
}

func getRunningProcess(ctx context.Context) (processes []string, err error) {
	pss, err := ps.Processes()

	if err != nil {
		log.Println(err)
		return nil, err
	}

	for _, p := range pss {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			processes = append(processes, p.Executable())
		}
	}

	return
}

// procRunningTime maps process name to running time
type procRunningTime map[string]time.Duration

// dailyReport maps date to process running time
type dailyReport map[string]procRunningTime

func (dr *dailyReport) accumulateTime(day string, proc string, dur time.Duration) {
	if _, dOk := (*dr)[day]; !dOk {
		(*dr)[day] = make(procRunningTime)
	}

	(*dr)[day][proc] = (*dr)[day][proc] + dur
}

var dailyReports dailyReport = make(dailyReport)

func checkProcesses(ctx context.Context, dur time.Duration) error {

	pss, err := getRunningProcess(ctx)

	if err != nil {
		return err
	}

	y, m, d := time.Now().Date()
	day := strconv.Itoa(y) + "-" + strconv.Itoa(int(m)) + "-" + strconv.Itoa(d)

	for _, p := range pss {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			dailyReports.accumulateTime(day, p, dur)
		}
	}

	return nil
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	// period defines how often the proccess list is checked
	const period = time.Minute * 3

	var wg sync.WaitGroup
	wg.Add(1)
	go scheduler(ctx, &wg, period, checkProcesses)

	time.Sleep(time.Second * 3)
	cancel()

	wg.Wait()
}
