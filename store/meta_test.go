package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetaSave(t *testing.T) {
	assert.NoError(t, MetaSave(&Meta{
		Source:   "test-source",
		Name:     "Test Source2",
		HomePage: "https://example.com/",
	}))
}
