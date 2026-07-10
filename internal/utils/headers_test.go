package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseRetryAfter(t *testing.T) {
	now := time.Now()
	future := now.Add(30 * time.Second).Format(time.RFC1123)
	past := now.Add(-30 * time.Second).Format(time.RFC1123)

	tests := []struct {
		name   string
		header string
		want   time.Duration
	}{
		{"empty", "", 0},
		{"seconds positive", "60", 60 * time.Second},
		{"seconds negative", "-10", 0},
		{"RFC1123 future", future, time.Second * 30},
		{"RFC1123 past", past, 0},
		{"invalid format", "abc", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseRetryAfter(tt.header)
			if tt.name == "RFC1123 future" {
				assert.Greater(t, got, 29*time.Second)
				assert.Less(t, got, 31*time.Second)
			} else {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
