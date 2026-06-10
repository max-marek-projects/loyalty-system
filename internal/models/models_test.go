package models

import (
	"database/sql/driver"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimeRFC3339_MarshalJSON(t *testing.T) {
	fixedTime := time.Date(2024, 1, 15, 14, 30, 45, 123456789, time.FixedZone("UTC+3", 3*3600))
	tests := []struct {
		name     string
		timeVal  TimeRFC3339
		expected string
		wantErr  bool
	}{
		{
			name:     "non-zero time",
			timeVal:  TimeRFC3339{Time: fixedTime},
			expected: `"2024-01-15T14:30:45+03:00"`,
			wantErr:  false,
		},
		{
			name:     "zero time",
			timeVal:  TimeRFC3339{Time: time.Time{}},
			expected: `"0001-01-01T00:00:00Z"`,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.timeVal)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.name == "zero time" {
					actual := string(data)
					if actual != `"0001-01-01T00:00:00Z"` && actual != `"0001-01-01T00:00:00+00:00"` {
						assert.Fail(t, "unexpected zero time format: %s", actual)
					}
				} else {
					assert.Equal(t, tt.expected, string(data))
				}
			}
		})
	}
}

func TestTimeRFC3339_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		jsonString  string
		expected    time.Time
		expectError bool
	}{
		{
			name:        "valid RFC3339 with timezone",
			jsonString:  `"2024-01-15T14:30:45+03:00"`,
			expected:    time.Date(2024, 1, 15, 14, 30, 45, 0, time.FixedZone("UTC+3", 3*3600)),
			expectError: false,
		},
		{
			name:        "valid RFC3339 Zulu",
			jsonString:  `"2024-01-15T14:30:45Z"`,
			expected:    time.Date(2024, 1, 15, 14, 30, 45, 0, time.UTC),
			expectError: false,
		},
		{
			name:        "empty string",
			jsonString:  `""`,
			expected:    time.Time{},
			expectError: false,
		},
		{
			name:        "null",
			jsonString:  `null`,
			expected:    time.Time{},
			expectError: false,
		},
		{
			name:        "invalid format",
			jsonString:  `"2024/01/15"`,
			expected:    time.Time{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tm TimeRFC3339
			err := json.Unmarshal([]byte(tt.jsonString), &tm)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.expected.IsZero() {
					assert.True(t, tm.Time.IsZero())
				} else {
					assert.True(t, tm.Time.Equal(tt.expected), "expected %v, got %v", tt.expected, tm.Time)
				}
			}
		})
	}
}

func TestTimeRFC3339_Scan(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	tests := []struct {
		name        string
		value       interface{}
		expected    time.Time
		expectError bool
	}{
		{
			name:        "scan time.Time",
			value:       now,
			expected:    now,
			expectError: false,
		},
		{
			name:        "scan string RFC3339",
			value:       now.Format(time.RFC3339),
			expected:    now,
			expectError: false,
		},
		{
			name:        "scan nil",
			value:       nil,
			expected:    time.Time{},
			expectError: false,
		},
		{
			name:        "scan empty string",
			value:       "",
			expected:    time.Time{},
			expectError: false,
		},
		{
			name:        "scan invalid string",
			value:       "not a time",
			expected:    time.Time{},
			expectError: true,
		},
		{
			name:        "scan unsupported type",
			value:       12345,
			expected:    time.Time{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tm TimeRFC3339
			err := tm.Scan(tt.value)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if !tt.expected.IsZero() {
					if tt.name == "scan string RFC3339" {
						assert.True(t, tm.Time.Truncate(time.Second).Equal(tt.expected), "expected %v, got %v", tt.expected, tm.Time)
					} else {
						assert.Equal(t, tt.expected.UTC(), tm.Time.UTC())
					}
				} else {
					assert.True(t, tm.Time.IsZero())
				}
			}
		})
	}
}

func TestTimeRFC3339_Value(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		timeVal  TimeRFC3339
		expected driver.Value
	}{
		{
			name:     "non-zero time",
			timeVal:  TimeRFC3339{Time: now},
			expected: now,
		},
		{
			name:     "zero time",
			timeVal:  TimeRFC3339{Time: time.Time{}},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := tt.timeVal.Value()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, val)
		})
	}
}
