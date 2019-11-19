package lib

import (
	"context"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/mitchellh/go-ps"
)

// DailyTimeLimit specify total time limit l for one or more processes p
type DailyTimeLimit struct {
	PG []string      `json:"processes"`
	L  time.Duration `json:"limit"`
}

// timeBalance maps process name to running time
type timeBalance map[string]time.Duration

// dailyTimeBalance maps date to process running time
type dailyTimeBalance map[string]timeBalance

// ProcessHunter is monitoring and killing processes that go overtime for particular day
type ProcessHunter struct {
	limits  []DailyTimeLimit
	balance dailyTimeBalance
	period  time.Duration
	killer  func(pid int) error
}

// NewProcessHunter initializes and returns a new ProcessHunter
func NewProcessHunter(limits []DailyTimeLimit, period time.Duration, killer func(int) error) *ProcessHunter {
	return &ProcessHunter{
		limits:  limits,
		period:  period,
		balance: make(dailyTimeBalance),
		killer:  killer,
	}
}

// checkProcesses updates processes time balance (addint t), checks for overtime and kills processes
func (ph *ProcessHunter) checkProcesses(ctx context.Context, t time.Duration) error {

	// 1. get all processes and update their time balance for the day
	// ---------------

	pss, err := ps.Processes()

	if err != nil {
		log.Println(err)
		return err
	}

	day := toText(time.Now())

	for _, p := range pss {
		ph.balance.add(day, p.Executable(), t)
	}

	// 2. check which processes are overtime and kill them
	// ---------------

	d := ph.balance[day]
	for _, l := range ph.limits { // iterate all limits
		bg := time.Duration(0)
		for _, p := range l.PG { // iterate all processes in the process group
			bg = bg + d[p]
		}
		if bg > l.L {
			log.Println("process group", l.PG, "total balance of", bg, "exceeds time limit of", l.L)
			for _, p := range l.PG { // iterate all processes in the process group
				if d[p] > 0 {
					log.Println("process", p, "from process group", l.PG, "has running time of", d[p])
					for _, a := range pss { // iterate all running processes
						if a.Executable() == p {
							log.Println("killing", a.Pid())
							// check if context is cancelled before attempting to kill
							select {
							case <-ctx.Done():
								return ctx.Err()
							default:
								if ph.killer != nil {
									err := ph.killer(a.Pid())
									if err != nil {
										log.Println("error killing process", a.Pid(), ":", err.Error())
									}
								}
							}
						}
					}
				}
			}
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

	err := work(ctx, 0) // don't add anything to process balance on the first call
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

// add adds t to the balance of the process proc for the day
func (dr *dailyTimeBalance) add(day string, proc string, t time.Duration) {
	if _, dOk := (*dr)[day]; !dOk {
		(*dr)[day] = make(timeBalance)
	}

	(*dr)[day][proc] = (*dr)[day][proc] + t
}

// toText returns string representation of the date of t
func toText(t time.Time) string {
	y, m, d := t.Date()
	return strconv.Itoa(y) + "-" + strconv.Itoa(int(m)) + "-" + strconv.Itoa(d)
}
