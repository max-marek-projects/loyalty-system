package repository

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/max-marek-projects/loyalty-system/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDBStorage_RegisterUser(t *testing.T) {
	type want struct {
		userID int64
		err    error
	}
	tests := []struct {
		name      string
		userData  models.UserData
		mockSetup func(mock sqlmock.Sqlmock)
		expected  want
	}{
		{
			name:     "success",
			userData: models.UserData{Login: "user", PasswordHash: "hash"},
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id"}).AddRow(123)
				mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO users (username, password_hash) VALUES ($1, $2) ON CONFLICT (username) DO NOTHING RETURNING id;`)).
					WithArgs("user", "hash").
					WillReturnRows(rows)
			},
			expected: want{userID: 123, err: nil},
		},
		{
			name:     "conflict (already exists)",
			userData: models.UserData{Login: "existing", PasswordHash: "hash"},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO users (username, password_hash) VALUES ($1, $2) ON CONFLICT (username) DO NOTHING RETURNING id;`)).
					WithArgs("existing", "hash").
					WillReturnError(sql.ErrNoRows)
			},
			expected: want{userID: 0, err: ErrAlreadyInStorage},
		},
		{
			name:     "database error",
			userData: models.UserData{Login: "error", PasswordHash: "hash"},
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO users (username, password_hash) VALUES ($1, $2) ON CONFLICT (username) DO NOTHING RETURNING id;`)).
					WithArgs("error", "hash").
					WillReturnError(errors.New("pq: deadlock"))
			},
			expected: want{userID: 0, err: errors.New("Failed to add user to storage: pq: deadlock")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			dbs := &dbStorage{storage: db}
			tt.mockSetup(mock)

			userID, err := dbs.RegisterUser(context.Background(), tt.userData)
			if tt.expected.err != nil {
				assert.EqualError(t, err, tt.expected.err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected.userID, userID)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestDBStorage_AddOrder(t *testing.T) {
	const (
		userID      = 100
		orderNumber = "9278923470"
		otherUserID = 200
	)

	tests := []struct {
		name      string
		userID    int64
		orderNum  string
		mockSetup func(mock sqlmock.Sqlmock)
		wantErr   error
	}{
		{
			name:     "success new order",
			userID:   userID,
			orderNum: orderNumber,
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id"}).AddRow(1)
				mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO orders (number, user_id, status) VALUES ($1, $2, $3) ON CONFLICT (number) DO NOTHING RETURNING id;`)).
					WithArgs(orderNumber, userID, models.StatusNEW).
					WillReturnRows(rows)
			},
			wantErr: nil,
		},
		{
			name:     "order already exists for same user",
			userID:   userID,
			orderNum: orderNumber,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO orders (number, user_id, status) VALUES ($1, $2, $3) ON CONFLICT (number) DO NOTHING RETURNING id;`)).
					WithArgs(orderNumber, userID, models.StatusNEW).
					WillReturnError(sql.ErrNoRows)
				rows := sqlmock.NewRows([]string{"user_id"}).AddRow(userID)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT user_id FROM orders WHERE number = $1;`)).
					WithArgs(orderNumber).
					WillReturnRows(rows)
			},
			wantErr: ErrAlreadyInStorage,
		},
		{
			name:     "conflict other user",
			userID:   userID,
			orderNum: orderNumber,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO orders (number, user_id, status) VALUES ($1, $2, $3) ON CONFLICT (number) DO NOTHING RETURNING id;`)).
					WithArgs(orderNumber, userID, models.StatusNEW).
					WillReturnError(sql.ErrNoRows)
				rows := sqlmock.NewRows([]string{"user_id"}).AddRow(otherUserID)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT user_id FROM orders WHERE number = $1;`)).
					WithArgs(orderNumber).
					WillReturnRows(rows)
			},
			wantErr: ErrStorageConflict,
		},
		{
			name:     "database error on insert",
			userID:   userID,
			orderNum: orderNumber,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO orders (number, user_id, status) VALUES ($1, $2, $3) ON CONFLICT (number) DO NOTHING RETURNING id;`)).
					WithArgs(orderNumber, userID, models.StatusNEW).
					WillReturnError(errors.New("disk full"))
			},
			wantErr: errors.New("Failed to add order to storage: disk full"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			dbs := &dbStorage{storage: db}
			tt.mockSetup(mock)

			err = dbs.AddOrder(context.Background(), tt.userID, tt.orderNum)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestDBStorage_GetAllOrders(t *testing.T) {
	tests := []struct {
		name      string
		userID    int64
		mockSetup func(mock sqlmock.Sqlmock)
		wantLen   int
		wantErr   error
	}{
		{
			name:   "success with rows",
			userID: 1,
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "number", "status", "accrual", "uploaded_at"}).
					AddRow(1, "111", models.StatusPROCESSED, 100.0, time.Now()).
					AddRow(2, "222", models.StatusNEW, 0.0, time.Now())
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, number, status, accrual, uploaded_at FROM orders WHERE user_id = $1 ORDER BY uploaded_at DESC`)).
					WithArgs(1).
					WillReturnRows(rows)
			},
			wantLen: 2,
			wantErr: nil,
		},
		{
			name:   "empty result",
			userID: 2,
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "number", "status", "accrual", "uploaded_at"})
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, number, status, accrual, uploaded_at FROM orders WHERE user_id = $1 ORDER BY uploaded_at DESC`)).
					WithArgs(2).
					WillReturnRows(rows)
			},
			wantLen: 0,
			wantErr: nil,
		},
		{
			name:   "database error",
			userID: 3,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, number, status, accrual, uploaded_at FROM orders WHERE user_id = $1 ORDER BY uploaded_at DESC`)).
					WithArgs(3).
					WillReturnError(errors.New("connection lost"))
			},
			wantLen: 0,
			wantErr: errors.New("Failed to get all orders from database: connection lost"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			dbs := &dbStorage{storage: db}
			tt.mockSetup(mock)

			orders, err := dbs.GetAllOrders(context.Background(), tt.userID)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Len(t, orders, tt.wantLen)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestDBStorage_CheckUser(t *testing.T) {
	type want struct {
		userID int64
		pass   string
		err    error
	}
	tests := []struct {
		name      string
		username  string
		mockSetup func(mock sqlmock.Sqlmock)
		expected  want
	}{
		{
			name:     "success",
			username: "john",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "password_hash"}).AddRow(123, "hashed_pass")
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, password_hash FROM users WHERE username = $1;`)).
					WithArgs("john").
					WillReturnRows(rows)
			},
			expected: want{userID: 123, pass: "hashed_pass", err: nil},
		},
		{
			name:     "user not found",
			username: "unknown",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, password_hash FROM users WHERE username = $1;`)).
					WithArgs("unknown").
					WillReturnError(sql.ErrNoRows)
			},
			expected: want{userID: 0, pass: "", err: ErrUserNotFound},
		},
		{
			name:     "database error",
			username: "error",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, password_hash FROM users WHERE username = $1;`)).
					WithArgs("error").
					WillReturnError(errors.New("pq: deadlock"))
			},
			expected: want{userID: 0, pass: "", err: errors.New("Failed to get user from storage by id: pq: deadlock")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			dbs := &dbStorage{storage: db}
			tt.mockSetup(mock)

			userID, pass, err := dbs.CheckUser(context.Background(), tt.username)
			if tt.expected.err != nil {
				assert.EqualError(t, err, tt.expected.err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected.userID, userID)
				assert.Equal(t, tt.expected.pass, pass)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestDBStorage_GetBalance(t *testing.T) {
	tests := []struct {
		name      string
		userID    int64
		mockSetup func(mock sqlmock.Sqlmock)
		want      *models.BalanceData
		wantErr   error
	}{
		{
			name:   "success",
			userID: 1,
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"balance", "withdrawn"}).AddRow(500.5, 42.0)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT balance, withdrawn FROM users WHERE id = $1`)).
					WithArgs(1).
					WillReturnRows(rows)
			},
			want:    &models.BalanceData{Current: 500.5, Withdrawn: 42},
			wantErr: nil,
		},
		{
			name:   "user not found",
			userID: 999,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT balance, withdrawn FROM users WHERE id = $1`)).
					WithArgs(999).
					WillReturnError(sql.ErrNoRows)
			},
			want:    nil,
			wantErr: errors.New("Failed to get all orders from database: sql: no rows in result set"),
		},
		{
			name:   "database error",
			userID: 1,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT balance, withdrawn FROM users WHERE id = $1`)).
					WithArgs(1).
					WillReturnError(errors.New("connection lost"))
			},
			want:    nil,
			wantErr: errors.New("Failed to get all orders from database: connection lost"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			dbs := &dbStorage{storage: db}
			tt.mockSetup(mock)

			balance, err := dbs.GetBalance(context.Background(), tt.userID)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, balance)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestDBStorage_GetAllWithdrawals(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name      string
		userID    int64
		mockSetup func(mock sqlmock.Sqlmock)
		wantLen   int
		wantErr   error
	}{
		{
			name:   "success with rows",
			userID: 1,
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"order_number", "sum", "processed_at"}).
					AddRow("111", 100.0, now).
					AddRow("222", 50.5, now)
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT order_number, sum, processed_at FROM withdrawals WHERE user_id = $1 ORDER BY processed_at DESC`)).
					WithArgs(1).
					WillReturnRows(rows)
			},
			wantLen: 2,
			wantErr: nil,
		},
		{
			name:   "empty result",
			userID: 2,
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"order_number", "sum", "processed_at"})
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT order_number, sum, processed_at FROM withdrawals WHERE user_id = $1 ORDER BY processed_at DESC`)).
					WithArgs(2).
					WillReturnRows(rows)
			},
			wantLen: 0,
			wantErr: nil,
		},
		{
			name:   "database error",
			userID: 3,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`SELECT order_number, sum, processed_at FROM withdrawals WHERE user_id = $1 ORDER BY processed_at DESC`)).
					WithArgs(3).
					WillReturnError(errors.New("disk full"))
			},
			wantLen: 0,
			wantErr: errors.New("Failed to get all orders from database: disk full"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			dbs := &dbStorage{storage: db}
			tt.mockSetup(mock)

			withdrawals, err := dbs.GetAllWithdrawals(context.Background(), tt.userID)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Len(t, withdrawals, tt.wantLen)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestDBStorage_MarkAllProcessingAsNew(t *testing.T) {
	tests := []struct {
		name      string
		mockSetup func(mock sqlmock.Sqlmock)
		wantErr   error
	}{
		{
			name: "success",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE orders SET status = $1 WHERE status = $2`)).
					WithArgs(models.StatusNEW, models.StatusPROCESSING).
					WillReturnResult(sqlmock.NewResult(0, 5))
			},
			wantErr: nil,
		},
		{
			name: "database error",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE orders SET status = $1 WHERE status = $2`)).
					WithArgs(models.StatusNEW, models.StatusPROCESSING).
					WillReturnError(errors.New("update failed"))
			},
			wantErr: errors.New("Failed to reset all statuses: update failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			dbs := &dbStorage{storage: db}
			tt.mockSetup(mock)

			err = dbs.MarkAllProcessingAsNew(context.Background())
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestDBStorage_FindAndClaimNewOrders(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name      string
		mockSetup func(mock sqlmock.Sqlmock)
		wantLen   int
		wantErr   error
	}{
		{
			name: "success with rows",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "number", "status", "accrual", "uploaded_at"}).
					AddRow(1, "111", models.StatusPROCESSING, 0.0, now).
					AddRow(2, "222", models.StatusPROCESSING, 0.0, now)
				mock.ExpectQuery(regexp.QuoteMeta(`UPDATE orders SET status = $1 WHERE status = $2 RETURNING id, number, status, accrual, uploaded_at;`)).
					WithArgs(models.StatusPROCESSING, models.StatusNEW).
					WillReturnRows(rows)
			},
			wantLen: 2,
			wantErr: nil,
		},
		{
			name: "empty result",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "number", "status", "accrual", "uploaded_at"})
				mock.ExpectQuery(regexp.QuoteMeta(`UPDATE orders SET status = $1 WHERE status = $2 RETURNING id, number, status, accrual, uploaded_at;`)).
					WithArgs(models.StatusPROCESSING, models.StatusNEW).
					WillReturnRows(rows)
			},
			wantLen: 0,
			wantErr: nil,
		},
		{
			name: "database error",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`UPDATE orders SET status = $1 WHERE status = $2 RETURNING id, number, status, accrual, uploaded_at;`)).
					WithArgs(models.StatusPROCESSING, models.StatusNEW).
					WillReturnError(errors.New("query error"))
			},
			wantLen: 0,
			wantErr: errors.New("Failed to get all orders from database: query error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			dbs := &dbStorage{storage: db}
			tt.mockSetup(mock)

			orders, err := dbs.FindAndClaimNewOrders(context.Background())
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Len(t, orders, tt.wantLen)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestDBStorage_UpdateOrderStatus(t *testing.T) {
	tests := []struct {
		name      string
		orderID   int64
		status    models.OrderStatus
		mockSetup func(mock sqlmock.Sqlmock)
		wantErr   error
	}{
		{
			name:    "success",
			orderID: 10,
			status:  models.StatusPROCESSED,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE orders SET status = $1 WHERE id = $2;`)).
					WithArgs(models.StatusPROCESSED, 10).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: nil,
		},
		{
			name:    "database error",
			orderID: 20,
			status:  models.StatusINVALID,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE orders SET status = $1 WHERE id = $2;`)).
					WithArgs(models.StatusINVALID, 20).
					WillReturnError(errors.New("update failed"))
			},
			wantErr: errors.New("Failed to update status: update failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			dbs := &dbStorage{storage: db}
			tt.mockSetup(mock)

			err = dbs.UpdateOrderStatus(context.Background(), tt.orderID, tt.status)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestDBStorage_ProcessOrderAccrual(t *testing.T) {
	tests := []struct {
		name      string
		orderID   int64
		accrual   float64
		mockSetup func(mock sqlmock.Sqlmock)
		wantErr   error
	}{
		{
			name:    "success",
			orderID: 100,
			accrual: 250.75,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				rows := sqlmock.NewRows([]string{"user_id"}).AddRow(5)
				mock.ExpectQuery(regexp.QuoteMeta(`UPDATE orders SET status = $1, accrual = $2 WHERE id = $3 RETURNING user_id`)).
					WithArgs(models.StatusPROCESSED, 250.75, 100).
					WillReturnRows(rows)
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE users SET balance = balance + $1 WHERE id = $2`)).
					WithArgs(250.75, 5).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
			wantErr: nil,
		},
		{
			name:    "update order fails",
			orderID: 101,
			accrual: 100,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(regexp.QuoteMeta(`UPDATE orders SET status = $1, accrual = $2 WHERE id = $3 RETURNING user_id`)).
					WithArgs(models.StatusPROCESSED, 100.0, 101).
					WillReturnError(errors.New("order not found"))
				mock.ExpectRollback()
			},
			wantErr: errors.New("update order failed: order not found"),
		},
		{
			name:    "update user balance fails",
			orderID: 102,
			accrual: 50,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				rows := sqlmock.NewRows([]string{"user_id"}).AddRow(7)
				mock.ExpectQuery(regexp.QuoteMeta(`UPDATE orders SET status = $1, accrual = $2 WHERE id = $3 RETURNING user_id`)).
					WithArgs(models.StatusPROCESSED, 50.0, 102).
					WillReturnRows(rows)
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE users SET balance = balance + $1 WHERE id = $2`)).
					WithArgs(50.0, 7).
					WillReturnError(errors.New("update users failed"))
				mock.ExpectRollback()
			},
			wantErr: errors.New("update user balance failed: update users failed"),
		},
		{
			name:    "user not found (rows affected 0)",
			orderID: 103,
			accrual: 30,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				rows := sqlmock.NewRows([]string{"user_id"}).AddRow(8)
				mock.ExpectQuery(regexp.QuoteMeta(`UPDATE orders SET status = $1, accrual = $2 WHERE id = $3 RETURNING user_id`)).
					WithArgs(models.StatusPROCESSED, 30.0, 103).
					WillReturnRows(rows)
				mock.ExpectExec(regexp.QuoteMeta(`UPDATE users SET balance = balance + $1 WHERE id = $2`)).
					WithArgs(30.0, 8).
					WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectRollback()
			},
			wantErr: errors.New("user 8 not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			dbs := &dbStorage{storage: db}
			tt.mockSetup(mock)

			err = dbs.ProcessOrderAccrual(context.Background(), tt.orderID, tt.accrual)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
