package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

// MarshalJSON marshals pgdb using 12h35m46s duration format
func (pd prettyDuration) MarshalJSON() ([]byte, error) {
	return json.Marshal(pd.String())
}

// MarshalJSON marshals tb using 12h35m46s duration format
func (tb TimeBalance) MarshalJSON() ([]byte, error) {
	aux := make(map[string]string)

	for k, v := range tb {
		aux[k] = time.Duration.String(v)
	}

	return json.Marshal(aux)
}

// UnmarshalJSON unmarshals dtl using 12h35m46s duration format
func (tb *TimeBalance) UnmarshalJSON(data []byte) error {
	aux := make(map[string]string)

	err := json.Unmarshal(data, &aux)

	if err != nil {
		return err
	}

	(*tb) = make(TimeBalance)

	for k, v := range aux {
		l, err := time.ParseDuration(v)
		if err != nil {
			break
		}
		(*tb)[k] = l
	}
	return err
}

// MarshalJSON marshals dtl using 12h35m46s duration format
func (dtl DailyLimits) MarshalJSON() ([]byte, error) {
	aux := make(map[string]string)

	for k, v := range dtl {
		aux[k] = time.Duration.String(v)
	}

	return json.Marshal(aux)
}

// UnmarshalJSON unmarshals dtl, lowercasing the key and using 12h35m46s duration format
func (dtl *DailyLimits) UnmarshalJSON(data []byte) error {
	aux := make(map[string]string)

	err := json.Unmarshal(data, &aux)

	if err != nil {
		return err
	}

	(*dtl) = make(DailyLimits)

	for k, v := range aux {
		l, err := time.ParseDuration(v)
		if err != nil {
			break
		}
		(*dtl)[strings.ToLower(k)] = l // converts to lower caps
	}
	return err
}

// isValidDailyLimitsFormat checks whether string with daily limits is correct
func isValidDailyLimitsFormat(l DailyLimits) bool {
	for k := range l {
		if k == "*" {
			continue
		}

		words := strings.Fields(k)

		if len(words) == 0 {
			return false
		}

		for _, w := range words {
			valid := false
			for _, d := range weekDays {
				if w == d {
					valid = true
					break
				}
			}
			if !valid {
				return false
			}
		}
	}

	return true
}

// LoadConfig loads ProcessHunder configuration from path
func (ph *ProcessHunter) LoadConfig() error {
	// try to load into this temporary variable first
	var limits []ProcessGroupDailyLimit

	b, err := ioutil.ReadFile(ph.cfgPath)
	if err != nil {
		return err
	}

	err = json.Unmarshal(b, &limits)
	if err != nil {
		return err
	}

	for _, l := range limits {
		if !isValidDailyLimitsFormat(l.DL) {
			return errors.New(fmt.Sprintln("bad days of the week format:", l.DL))
		}
	}

	ph.limitsRWM.Lock()
	ph.limits = limits
	ph.limitsRWM.Unlock()

	file, err := os.Stat(ph.cfgPath)
	if err != nil {
		return err
	}

	ph.cfgTime = file.ModTime()

	return nil
}

// LoadBalance loads the balance from provided path
func (ph *ProcessHunter) LoadBalance() error {
	ph.balance = make(dailyTimeBalance)

	b, err := ioutil.ReadFile(ph.balancePath)

	if err != nil {
		return err
	}

	return json.Unmarshal(b, &ph.balance)
}

// saveBalance saves balance to provided path
func (ph *ProcessHunter) saveBalance() error {
	d, err := json.MarshalIndent(ph.balance, "", "\t")

	if err != nil {
		return err
	}

	return ioutil.WriteFile(ph.balancePath, d, 0644)
}

// SaveBalance saves balance to provided path
func (ph *ProcessHunter) SaveBalance() error {
	ph.balanceRWM.RLock()
	defer ph.balanceRWM.RUnlock()

	return ph.saveBalance()
}
