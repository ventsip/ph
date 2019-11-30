package lib

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
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
func (ph *ProcessHunter) LoadConfig(path string) error {
	ph.limits = nil

	b, err := ioutil.ReadFile(path)

	if err != nil {
		return err
	}

	err = json.Unmarshal(b, &ph.limits)

	if err != nil {
		return err
	}

	for _, l := range ph.limits {
		if !isValidDailyLimitsFormat(l.DL) {
			return errors.New(fmt.Sprintln("bad days of the week format:", l.DL))
		}
	}

	return nil
}

// LoadBalance loads the balance from provided path
func (ph *ProcessHunter) LoadBalance(path string) error {
	ph.balance = make(dailyTimeBalance)

	b, err := ioutil.ReadFile(path)

	if err != nil {
		return err
	}

	return json.Unmarshal(b, &ph.balance)
}

// SaveBalance saves balance to provided path
func (ph *ProcessHunter) SaveBalance(path string) error {
	d, err := json.MarshalIndent(ph.balance, "", "\t")

	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, d, 0644)
}
