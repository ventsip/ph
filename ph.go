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
	limits  DailyTimeLimit
	balance dailyTimeBalance
	period  time.Duration
	killer  func(pid int, force bool) error
}

// NewProcessHunter initializes and returns a new ProcessHunter
func NewProcessHunter(limits DailyTimeLimit, period time.Duration, killer func(int, bool) error) *ProcessHunter {
	return &ProcessHunter{
		limits:  limits,
		period:  period,
		balance: make(dailyTimeBalance),
		killer:  killer,
	}
}

// checkProcesses updates processes time balance and checks for overtime
func (ph *ProcessHunter) checkProcesses(ctx context.Context, dur time.Duration) error {

	// 1. get all processes and update their time balance for the day
	// ---------------

	pss, err := ps.Processes()

	if err != nil {
		log.Println(err)
		return err
	}

	day := toText(time.Now())

	for _, p := range pss {
		ph.balance.add(day, p.Executable(), dur)
	}

	// 2. check which processes are overtime and kill them
	// ---------------

	d := ph.balance[day]
	for p, l := range ph.limits {
		if d[p] > l {
			force := d[p] > l+time.Second*15
			log.Println("process", p, "running time of", d[p], "is over the time limit of", l)
			pidExists := false
			for _, a := range pss {
				if a.Executable() == p {
					pidExists = true
					log.Println("killing", a.Pid(), "force:", force)
					// check if context is cacelled before attempting to kill
					select {
					case <-ctx.Done():
						return ctx.Err()
					default:
						if ph.killer != nil {
							err := ph.killer(a.Pid(), force)
							if err != nil {
								log.Println("error killing process", a.Pid(), ":", err.Error())
							}
						}
					}
				}
			}
			if pidExists == false {
				delete(ph.balance[day], p)
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

// add adds dur to the balance of the process proc for the day
func (dr *dailyTimeBalance) add(day string, proc string, dur time.Duration) {
	if _, dOk := (*dr)[day]; !dOk {
		(*dr)[day] = make(timeBalance)
	}

	(*dr)[day][proc] = (*dr)[day][proc] + dur
}

// toText returns string representation of the date of t
func toText(t time.Time) string {
	y, m, d := t.Date()
	return strconv.Itoa(y) + "-" + strconv.Itoa(int(m)) + "-" + strconv.Itoa(d)
}
