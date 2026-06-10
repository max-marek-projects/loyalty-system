package config

import (
	"flag"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"cmd", "-a=:8081", "-l=DEBUG", "-d=postgres://test", "-s=secret", "-p=10", "-r=http://accrual", "-i=5", "-m=true", "-t=10", "-w=10"}
	cfg := LoadConfig()
	assert.NotNil(t, cfg)
	assert.Equal(t, ":8081", cfg.RunAddr)
	assert.Equal(t, "DEBUG", cfg.LoggerLevel)
	assert.Equal(t, "postgres://test", cfg.DatabaseURI)
	assert.Equal(t, "secret", cfg.CookieSecret)
	assert.Equal(t, 10, cfg.MaxParallelWorkers)
	assert.Equal(t, "http://accrual", cfg.AccrualSystemAddress)
	assert.Equal(t, 5, cfg.PollInterval)
	assert.True(t, cfg.MockExternalService)
	assert.Equal(t, 10*time.Second, cfg.ReadTimeout)
	assert.Equal(t, 10*time.Second, cfg.WriteTimeout)

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	os.Args = []string{"cmd"}
	os.Setenv("RUN_ADDRESS", ":9090")
	os.Setenv("LOGGER_LEVEL", "ERROR")
	defer os.Unsetenv("RUN_ADDRESS")
	defer os.Unsetenv("LOGGER_LEVEL")
	cfg2 := LoadConfig()
	assert.Equal(t, ":9090", cfg2.RunAddr)
	assert.Equal(t, "ERROR", cfg2.LoggerLevel)
}
