package config

import (
	"fmt"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/stretchr/testify/assert"
)

func TestParseDuration(t *testing.T) {
	t.Parallel()

	parseDurationTests := []struct {
		input    string
		expected string
	}{
		{input: "29 second", expected: "29s"},
		{input: "29 seconds", expected: "29s"},
		{input: "30 minute", expected: "30m"},
		{input: "30 minutes", expected: "30m"},
		{input: "1 hour", expected: "1h"},
		{input: "2 hours", expected: "2h"},
		{input: "1 day", expected: "24h"},
		{input: "2 days", expected: "48h"},
		{input: "1 week", expected: "168h"},
		{input: "2 weeks", expected: "336h"},
		{input: "1 month", expected: "720h"},
		{input: "2 months", expected: "1440h"},
		{input: "7 days and 12 hour", expected: "180h"},
		{input: "1 month 7 days and 12 hour", expected: "900h"},
		{input: "3 months 1 month 4 days and 12 hour", expected: "2988h"},
	}

	for _, tt := range parseDurationTests {
		t.Run(tt.input, func(t *testing.T) {
			actual, err := parseDuration(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			dt, err := time.ParseDuration(tt.expected)
			assert.NoError(t, err)
			if int(actual.Seconds()) != int(dt.Seconds()) {
				t.Errorf("expected %s, got %d", tt.expected, int(actual.Minutes()))
			}
		})
	}
}

func TestParseDurationError(t *testing.T) {
	t.Parallel()

	dt, err := parseDuration("Hello World")

	assert.ErrorContains(t, err, "invalid duration: Hello World")
	assert.Nil(t, dt)
}

func TestUnmarshalYAML(t *testing.T) {
	t.Parallel()

	test := []struct {
		name     string
		input    string
		err      error
		expected time.Duration
	}{
		{
			name:     "valid duration",
			input:    `5 seconds`,
			expected: 5 * time.Second,
		},
		{
			name:     "empty duration",
			input:    ``,
			expected: time.Duration(0),
		},
		{
			name:  "invalid duration",
			input: `5 lightyears`,
			err:   fmt.Errorf("invalid duration: 5 lightyears"),
		},
		{
			name:  "invalid yaml",
			input: `app:  5 seconds`,
			err:   fmt.Errorf("yaml: unmarshal errors:\n  line 1: cannot unmarshal !!map into string"),
		},
	}

	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			var node yaml.Node
			err := yaml.Unmarshal([]byte(tt.input), &node)
			assert.NoError(t, err)

			d := Duration{}

			err = d.UnmarshalYAML(&node)
			if tt.err != nil {
				assert.ErrorContains(t, tt.err, err.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expected, d.Duration)
		})
	}
}
