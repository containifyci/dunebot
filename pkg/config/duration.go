package config

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Duration struct {
	time.Duration
}

var durationRegex = regexp.MustCompile(`(?i)(\d+)\s*(second|minute|hour|day|week|month)s?`)

func parseDuration(input string) (*time.Duration, error) {
	matches := durationRegex.FindAllStringSubmatch(input, -1)
	if matches == nil {
		return nil, fmt.Errorf("invalid duration: %s", input)
	}

	var totalDuration time.Duration
	for _, match := range matches {
		value, err := strconv.Atoi(match[1])
		if err != nil {
			return nil, fmt.Errorf("invalid number in duration: %s", match[1])
		}

		unit := strings.ToLower(match[2])
		var duration time.Duration
		switch unit {
		case "second":
			duration = time.Duration(value) * time.Second
		case "minute":
			duration = time.Duration(value) * time.Minute
		case "hour":
			duration = time.Duration(value) * time.Hour
		case "day":
			duration = time.Duration(value) * 24 * time.Hour
		case "week":
			duration = time.Duration(value) * 7 * 24 * time.Hour
		case "month":
			// Approximation: 30 days per month
			duration = time.Duration(value) * 30 * 24 * time.Hour
		default:
			return nil, fmt.Errorf("unknown time unit: %s", unit)
		}

		totalDuration += duration
	}

	return &totalDuration, nil
}

func (p *Duration) UnmarshalYAML(value *yaml.Node) error {
	age := new(string)

	if err := value.Decode(&age); err != nil {
		return err
	}

	if age == nil {
		return nil
	}

	dt, err := parseDuration(*age)
	if err != nil {
		return err
	}

	p.Duration = *dt

	return nil
}
