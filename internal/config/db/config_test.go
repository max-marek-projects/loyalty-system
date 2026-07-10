package db

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewDBConf(t *testing.T) {
	cfg := NewDBConf("postgres://test")
	assert.Equal(t, "postgres://test", cfg.URL)
	assert.Equal(t, 10, cfg.MaxOpenConns)
	assert.Equal(t, 5, cfg.MaxIdleConns)
	assert.Equal(t, 5*time.Minute, cfg.ConnMaxLifetime)
	assert.Equal(t, "./migrations", cfg.MigrationsPath)
}
