package lib

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/mitchellh/go-ps"
)

// DailyLimits maps days of the week to time limit
// The key can be "*" (meaing 'any day of the week') or space separated string of
// three-letter abbreviations of the days of week, i.e.
// * Mon Tue Wed Thu Fri Sat Sun
type DailyLimits map[string]time.Duration

// ProcessGroupDailyLimit specifies daily time limit DL
// for one or more processes in PG
type ProcessGroupDailyLimit struct {
	PG []string    `json:"processes"`
	DL DailyLimits `json:"limits"`
}

// timeBalance maps process name to running time
type timeBalance map[string]time.Duration

// dailyTimeBalance maps date to process running time
type dailyTimeBalance map[string]timeBalance

// ProcessHunter is monitoring and killing processes that go overtime for particular day
type ProcessHunter struct {
	limits      []ProcessGroupDailyLimit
	balance     dailyTimeBalance
	path        string // path is where the balance is periodically stored
	savePeriod  time.Duration
	checkPeriod time.Duration
	killer      func(pid int) error
}

// NewProcessHunter initializes and returns a new ProcessHunter
func NewProcessHunter(limits []ProcessGroupDailyLimit, checkPeriod time.Duration, path string, savePeriod time.Duration, killer func(int) error) *ProcessHunter {
	return &ProcessHunter{
		limits:      limits,
		checkPeriod: checkPeriod,
		balance:     make(dailyTimeBalance),
		path:        path,
		savePeriod:  savePeriod,
		killer:      killer,
	}
}

// GetLimits returns current daily limits (which are normally loaded from a config file)
func (ph *ProcessHunter) GetLimits() []ProcessGroupDailyLimit {
	return ph.limits
}

// savePeriod is when the balance was last saved
var lastSaved = time.Now()

var weekDays = [...]string{
	"sun",
	"mon",
	"tue",
	"wed",
	"thu",
	"fri",
	"sat",
}

// evalDailyLimit returns the daily time limit, parsing dl map
// prioritizing more concrete, to more generic specifications, in order:
// - specific day, e.g. "wed"
// - a day from a list: "mon wed fri"
// - any day "*"
func evalDailyLimit(wd string, dl DailyLimits) (l time.Duration) {
	l = time.Hour * 25 // effectively - no limit
	ingoreAny := false
	for k, v := range dl {
		if k == wd {
			l = v
			break
		}
		if strings.Contains(k, wd) {
			l = v
			ingoreAny = true
		}
		if k == "*" && !ingoreAny {
			l = v
		}
	}
	return
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

	now := time.Now()
	date := toText(now)
	weekDay := weekDays[now.Weekday()]

	for _, p := range pss {
		ph.balance.add(date, p.Executable(), t)
	}

	// 2. check which processes are overtime and kill them
	// ---------------

	d := ph.balance[date]
	for _, pdl := range ph.limits { // iterate all processes daily limits
		bg := time.Duration(0)
		for _, p := range pdl.PG { // iterate all processes in the process group
			bg = bg + d[p]
		}

		l := evalDailyLimit(weekDay, pdl.DL)

		if bg > l {
			log.Println(pdl.PG, ":", bg, "/", l)
			for _, p := range pdl.PG { // iterate all processes in the process group
				if d[p] > 0 {
					log.Println(p, ":", d[p])
					for _, a := range pss { // iterate all running processes
						if a.Executable() == p {
							// check if context is cancelled before attempting to kill
							select {
							case <-ctx.Done():
								return ctx.Err()
							default:
								if ph.killer != nil {
									log.Println("killing", a.Pid())
									err := ph.killer(a.Pid())
									if err != nil {
										log.Println("error killing", a.Pid(), ":", err.Error())
									}
								}
							}
						}
					}
				}
			}
		} else {
			log.Println(pdl.PG, "remaining:", l-bg)
		}
	}

	if (lastSaved.Add(ph.savePeriod)).Before(time.Now()) {
		if ph.path != "" {
			log.Println("saving balance", ph.path)
			err := ph.SaveBalance(ph.path)

			if err != nil {
				log.Println("error saving balance to", ph.path, ":", err)
			} else {
				lastSaved = time.Now()
			}
		}
	}

	return nil
}

// Run is a goroutine that periodically checks running processes
func (ph *ProcessHunter) Run(ctx context.Context, wg *sync.WaitGroup) {
	scheduler(ctx, wg, ph.checkPeriod, ph.checkProcesses)
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

	t := time.Now()
	err := work(ctx, 0) // don't add anything to process balance on the first call
	if err != nil {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			work(ctx, time.Now().Sub(t))
			t = time.Now()
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
	return t.Format("2006-01-02")
}