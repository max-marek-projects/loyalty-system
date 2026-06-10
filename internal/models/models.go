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
	return []byte(`"` + ct.Time.Format("2006-01-02T15:04:05-07:00") + `"`), nil
}

func (ct *TimeRFC3339) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	t, err := time.Parse("2006-01-02T15:04:05-07:00", s)
	if err != nil {
		return err
	}
	ct.Time = t
	return nil
}

func (t *TimeRFC3339) Scan(value interface{}) error {
	if value == nil {
		t.Time = time.Time{}
		return nil
	}
	switch v := value.(type) {
	case time.Time:
		t.Time = v
		return nil
	case string:
		parsed, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return err
		}
		t.Time = parsed
		return nil
	default:
		return fmt.Errorf("cannot scan %T into TimeRFC3339", value)
	}
}

// driver.Valuer to
func (t TimeRFC3339) Value() (driver.Value, error) {
	if t.IsZero() {
		return nil, nil
	}
	return t.Time, nil
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
