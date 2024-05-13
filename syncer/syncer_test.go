package syncer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewWorker(t *testing.T) {
	w, err := NewWorker("https://raw.githubusercontent.com/ismdeep/insight-hub-data/main/data/dunwu.meta.json")
	assert.NoError(t, err)
	t.Logf("got w.indexLink         = %v", w.indexLink)
	t.Logf("got w.contentLinkPrefix = %v", w.contentLinkPrefix)
}
