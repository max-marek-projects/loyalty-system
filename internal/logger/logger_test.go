package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestInitialize(t *testing.T) {
	err := Initialize("INFO")
	assert.NoError(t, err)
	assert.NotNil(t, Log)

	err = Initialize("invalid-level")
	assert.Error(t, err)

	Log = zap.NewNop()
}
