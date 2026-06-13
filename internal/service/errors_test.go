package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestErrorOrderNotYetProcessed(t *testing.T) {
	innerErr := errors.New("test error")
	retry := time.Duration(5) * time.Second

	errObj := &ErrorOrderNotYetProcessed{
		Err:        innerErr,
		RetryAfter: retry,
	}

	t.Run("Error method", func(t *testing.T) {
		assert.True(t, strings.Contains(errObj.Error(), innerErr.Error()))
	})

	t.Run("Unwrap returns inner error", func(t *testing.T) {
		assert.Equal(t, innerErr, errObj.Unwrap())
	})

	t.Run("errors.Is works", func(t *testing.T) {
		assert.True(t, errors.Is(errObj, innerErr))
		assert.False(t, errors.Is(errObj, errors.New("other")))
	})

	t.Run("errors.As works", func(t *testing.T) {
		var target *ErrorOrderNotYetProcessed
		assert.True(t, errors.As(errObj, &target))
		assert.Equal(t, errObj, target)
	})
}

func TestErrorOrderNotYetProcessed_NilInnerError(t *testing.T) {
	errObj := &ErrorOrderNotYetProcessed{
		Err:        nil,
		RetryAfter: 0,
	}
	assert.Nil(t, errObj.Unwrap())
}
