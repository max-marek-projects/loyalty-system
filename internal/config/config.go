// Package config provides configuration loading from environment variables, .env file, and command-line flags.
package config

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// Config holds all application configuration parameters.
type Config struct {
	RunAddr              string        `env:"RUN_ADDRESS"`            // address and port to run server
	ReadTimeout          time.Duration `env:"READ_TIMEOUT"`           // server read timeout in seconds
	WriteTimeout         time.Duration `env:"WRITE_TIMEOUT"`          // server write timeout in seconds
	LoggerLevel          string        `env:"LOGGER_LEVEL"`           // logger level DEBUG / INFO / WARNING / ERROR / FATAL
	DatabaseURI          string        `env:"DATABASE_URI"`           // database connection url
	CookieSecret         string        `env:"COOKIE_SECRET"`          // secret for cookie signature
	MaxParallelWorkers   int           `env:"MAX_PARALLEL_WORKERS"`   // max amount of parallel workers
	PollInterval         int           `env:"POLL_INTERVAL"`          // external system poll interval
	MockExternalService  bool          `env:"MOCK_EXTERNAL_STORAGE"`  // mock external storage
	AccrualSystemAddress string        `env:"ACCRUAL_SYSTEM_ADDRESS"` // loyalty calculation system address
}

// LoadConfig parses configuration from .env, environment variables, and command-line flags.
// Returns a pointer to the populated Config struct.
func LoadConfig() *Config {
	config := &Config{
		RunAddr:              ":8080",
		LoggerLevel:          "INFO",
		DatabaseURI:          "",
		CookieSecret:         "",
		MaxParallelWorkers:   5,
		AccrualSystemAddress: "",
		PollInterval:         5,
		MockExternalService:  false,
		ReadTimeout:          time.Duration(30) * time.Second,
		WriteTimeout:         time.Duration(30) * time.Second,
	}
	// load env vars
	err := godotenv.Load()
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("No .env file found, using environment variables and flags")
		} else {
			log.Fatalf("Failed to load .env file: %v", err)
		}
	}
	if err := env.Parse(config); err != nil {
		log.Printf("warning: failed to parse env: %v", err)
	}
	// read flags directly to config
	flag.StringVar(&config.RunAddr, "a", config.RunAddr, "address and port to run server")
	flag.StringVar(&config.LoggerLevel, "l", config.LoggerLevel, "logger level")
	flag.StringVar(&config.DatabaseURI, "d", config.DatabaseURI, "database connection url")
	flag.StringVar(&config.CookieSecret, "s", config.CookieSecret, "cookie signing secret")
	flag.IntVar(&config.MaxParallelWorkers, "p", config.MaxParallelWorkers, "maximum amount of parallel workers")
	flag.StringVar(&config.AccrualSystemAddress, "r", config.AccrualSystemAddress, "accrual system base address")
	flag.IntVar(&config.PollInterval, "i", config.PollInterval, "external service poll interval")
	flag.BoolVar(&config.MockExternalService, "m", config.MockExternalService, "mock external storage")
	flag.DurationVar(&config.ReadTimeout, "t", config.ReadTimeout, "server read timeout in seconds")
	flag.DurationVar(&config.WriteTimeout, "w", config.WriteTimeout, "server write timeout in seconds")
	// parse flags
	flag.Parse()
	return config
}
