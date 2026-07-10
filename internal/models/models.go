// Package models defines data structures used across the loyalty system.
package models

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"
)

// RegisterRequest represents the JSON payload for user registration and login.
type RegisterRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// UserData holds user credentials for internal storage (login and password hash).
type UserData struct {
	Login        string
	PasswordHash string
}

// OrderStatus represents the possible states of an order processing.
type OrderStatus string

const (
	StatusNEW        OrderStatus = "NEW"        // Order just created, not yet sent to accrual system.
	StatusPROCESSING OrderStatus = "PROCESSING" // Order is being processed by accrual system.
	StatusPROCESSED  OrderStatus = "PROCESSED"  // Order finished, accrual added to balance.
	StatusINVALID    OrderStatus = "INVALID"    // Order number is invalid or processing failed.
)

// TimeRFC3339 is a wrapper over time.Time that marshals/unmarshals to RFC3339 JSON.
type TimeRFC3339 struct {
	time.Time
}

// MarshalJSON outputs the time in RFC3339 format, or a zero-value sentinel.
func (ct TimeRFC3339) MarshalJSON() ([]byte, error) {
	if ct.IsZero() {
		return []byte(`"0001-01-01T00:00:00Z"`), nil
	}
	return []byte(`"` + ct.Time.Format(time.RFC3339) + `"`), nil
}

// UnmarshalJSON parses an RFC3339 string into TimeRFC3339.
func (ct *TimeRFC3339) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	if s == "" || s == "null" {
		ct.Time = time.Time{}
		return nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return err
	}
	ct.Time = t
	return nil
}

// Scan implements sql.Scanner for TimeRFC3339.
func (ct *TimeRFC3339) Scan(value interface{}) error {
	if value == nil {
		ct.Time = time.Time{}
		return nil
	}
	switch v := value.(type) {
	case time.Time:
		ct.Time = v
		return nil
	case string:
		if v == "" {
			ct.Time = time.Time{}
			return nil
		}
		parsed, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return err
		}
		ct.Time = parsed
		return nil
	default:
		return fmt.Errorf("cannot scan %T into TimeRFC3339", value)
	}
}

// Value implements driver.Valuer for TimeRFC3339.
func (ct TimeRFC3339) Value() (driver.Value, error) {
	if ct.IsZero() {
		return nil, nil
	}
	return ct.Time, nil
}

// OrderData represents an order stored in the database.
type OrderData struct {
	ID         int64       `json:"-"`           // Internal database ID.
	Number     string      `json:"number"`      // Order number (validated by Luhn).
	Status     OrderStatus `json:"status"`      // Current processing status.
	Accrual    float64     `json:"accrual"`     // Loyalty points earned.
	UploadedAt TimeRFC3339 `json:"uploaded_at"` // When the order was added.
}

// ExternalStatus represents the status of an order in the accrual system.
type ExternalStatus string

const (
	ExternalStatusREGISTERED = "REGISTERED" // Order is known but not yet processed.
	ExternalStatusINVALID    = "INVALID"    // Order number is invalid.
	ExternalStatusPROCESSING = "PROCESSING" // Order is being processed.
	ExternalStatusPROCESSED  = "PROCESSED"  // Processing finished, accrual available.
)

// AccrualResponse is the response from the external accrual system.
type AccrualResponse struct {
	Order   string         `json:"order"`   // Order number.
	Status  ExternalStatus `json:"status"`  // Status from external system.
	Accrual float64        `json:"accrual"` // Calculated loyalty points.
}

// BalanceData represents the user's current balance and total withdrawn amount.
type BalanceData struct {
	Current   float64 `json:"current"`   // Points available for withdrawal.
	Withdrawn float64 `json:"withdrawn"` // Total points already withdrawn.
}

// WithdrawRequest is the JSON payload for a withdrawal request.
type WithdrawRequest struct {
	Order string  `json:"order"` // Order number to withdraw against.
	Sum   float64 `json:"sum"`   // Amount to withdraw.
}

// WithdrawData represents a withdrawal record stored in the database.
type WithdrawData struct {
	Order       string      `json:"order"`        // Order number.
	Sum         float64     `json:"sum"`          // Amount withdrawn.
	ProcessedAt TimeRFC3339 `json:"processed_at"` // Timestamp of withdrawal.
}
