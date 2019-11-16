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

// timeBalance maps process name to running time
type timeBalance map[string]time.Duration

// dailyTimeBalance maps date to process running time
type dailyTimeBalance map[string]timeBalance

// add adds dur to the balance of the process proc for the day
func (dr *dailyTimeBalance) add(day string, proc string, dur time.Duration) {
	if _, dOk := (*dr)[day]; !dOk {
		(*dr)[day] = make(timeBalance)
	}

	(*dr)[day][proc] = (*dr)[day][proc] + dur
}

// isOverTime returns true if the process proc time balance is above specified duration dur
func (dr *dailyTimeBalance) isOverTime(day string, proc string, dur time.Duration) bool {
	return (*dr)[day][proc] > dur
}

// toText returns string representation of the date of t
func toText(t time.Time) string {
	y, m, d := t.Date()
	return strconv.Itoa(y) + "-" + strconv.Itoa(int(m)) + "-" + strconv.Itoa(d)
}

var dailyReports dailyTimeBalance = make(dailyTimeBalance)

func checkProcesses(ctx context.Context, dur time.Duration) error {

	pss, err := getRunningProcess(ctx)

	if err != nil {
		return err
	}

	day := toText(time.Now())

	for _, p := range pss {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			dailyReports.add(day, p, dur)
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
