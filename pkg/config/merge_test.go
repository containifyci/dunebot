package config

import (
	"testing"

	"github.com/containifyci/dunebot/pkg/config/testdata"
	"github.com/stretchr/testify/assert"
)

func TestMerge(t *testing.T) {
	t.Parallel()

	c1, err := testdata.Data.ReadFile("valid_config.yaml")

	assert.NoError(t, err)

	c2, err := testdata.Data.ReadFile("valid_config_merge.yaml")

	assert.NoError(t, err)

	cnt, err := merge(c1, c2)
	assert.NoError(t, err)

	cfg, _, err := Unmarshal(cnt)
	assert.NoError(t, err)

	assert.Equal(t, "1.0.2", cfg.Version)
	assert.Equal(t, "frank", cfg.Approve.Approver)
}

func TestMergeError(t *testing.T) {
	t.Parallel()

	_, err := merge([]byte(" \t \thello: world"), []byte("hello\nworld"))
	assert.Error(t, err)
}

func TestReadError(t *testing.T) {
	t.Parallel()

	_, err := read([]byte(" \t\t hello\nworld"))
	assert.Error(t, err)
}
