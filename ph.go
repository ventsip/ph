package ph

import (
	"context"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/mitchellh/go-ps"
)

// DailyTimeLimit maps process name to the daily time limit
type DailyTimeLimit map[string]time.Duration

// timeBalance maps process name to running time
type timeBalance map[string]time.Duration

// dailyTimeBalance maps date to process running time
type dailyTimeBalance map[string]timeBalance

// ProcessHunter is monitoring and killing processes that go overtime for particular day
type ProcessHunter struct {
	limits       DailyTimeLimit
	dailyReports dailyTimeBalance

	period time.Duration
}

// NewProcessHunter initializes and returns a new ProcessHunter
func NewProcessHunter(limits DailyTimeLimit, period time.Duration) *ProcessHunter {
	return &ProcessHunter{limits: limits, period: period, dailyReports: make(dailyTimeBalance)}
}

func (ph *ProcessHunter) checkProcesses(ctx context.Context, dur time.Duration) error {

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
			ph.dailyReports.add(day, p, dur)
		}
	}

	return nil
}

// Run is a goroutine that periodically checks running processes
func (ph *ProcessHunter) Run(ctx context.Context, wg *sync.WaitGroup) {
	scheduler(ctx, wg, ph.period, ph.checkProcesses)
}

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
