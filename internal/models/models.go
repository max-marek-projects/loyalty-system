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
	Id         int64       `json:"-"`
	Number     string      `json:"number"`
	Status     OrderStatus `json:"status"`
	Accrual    int64       `json:"accrual"`
	UploadedAt TimeRFC3339 `json:"uploaded_at"`
}
