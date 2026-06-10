package models

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"
)

type RegisterRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type UserData struct {
	Login        string
	PasswordHash string
}

type OrderStatus string

const (
	StatusNEW        OrderStatus = "NEW"
	StatusPROCESSING OrderStatus = "PROCESSING"
	StatusPROCESSED  OrderStatus = "PROCESSED"
	StatusINVALID    OrderStatus = "INVALID"
)

type TimeRFC3339 struct {
	time.Time
}

func (ct TimeRFC3339) MarshalJSON() ([]byte, error) {
	if ct.IsZero() {
		return []byte(`"0001-01-01T00:00:00Z"`), nil
	}
	return []byte(`"` + ct.Time.Format(time.RFC3339) + `"`), nil
}

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

// driver.Valuer to
func (ct TimeRFC3339) Value() (driver.Value, error) {
	if ct.IsZero() {
		return nil, nil
	}
	return ct.Time, nil
}

type OrderData struct {
	ID         int64       `json:"-"`
	Number     string      `json:"number"`
	Status     OrderStatus `json:"status"`
	Accrual    float64     `json:"accrual"`
	UploadedAt TimeRFC3339 `json:"uploaded_at"`
}

type ExternalStatus string

const (
	ExternalStatusREGISTERED = "REGISTERED"
	ExternalStatusINVALID    = "INVALID"
	ExternalStatusPROCESSING = "PROCESSING"
	ExternalStatusPROCESSED  = "PROCESSED"
)

type AccrualResponse struct {
	Order   string         `json:"order"`
	Status  ExternalStatus `json:"status"`
	Accrual float64        `json:"accrual"`
}

type BalanceData struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

type WithdrawRequest struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

type WithdrawData struct {
	Order       string      `json:"order"`
	Sum         float64     `json:"sum"`
	ProcessedAt TimeRFC3339 `json:"processed_at"`
}
