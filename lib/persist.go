package lib

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

// UnmarshalJSON unmarshales dtl using 12h35m46s duration format
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

	ph.limits = limits

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

// SaveBalance saves balance to provided path
func (ph *ProcessHunter) SaveBalance() error {
	d, err := json.MarshalIndent(ph.balance, "", "\t")

	if err != nil {
		return err
	}

	return ioutil.WriteFile(ph.balancePath, d, 0644)
}
