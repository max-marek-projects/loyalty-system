package config

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	RunAddr              string        `env:"RUN_ADDRESS"`            // address and port to run server
	ReadTimeout          time.Duration `env:"READ_TIMEOUT"`           // server read timeout in seconds
	WriteTimeout         time.Duration `env:"WRITE_TIMEOUT"`          // server write timeout in seconds
	LoggerLevel          string        `env:"LOGGER_LEVEL"`           // logger level DEBUG / INFO / WARNING / ERROR / FATAL
	DatabaseUri          string        `env:"DATABASE_URI"`           // database connection url
	CookieSecret         string        `env:"COOKIE_SECRET"`          // secret for cookie signature
	MaxParallelWorkers   int           `env:"MAX_PARALLEL_WORKERS"`   // max amount of parallel workers
	PollInterval         int           `env:"POLL_INTERVAL"`          // external system poll interval
	MockExternalService  bool          `env:"MOCK_EXTERNAL_STORAGE"`  // mock external storage
	AccrualSystemAddress string        `env:"ACCRUAL_SYSTEM_ADDRESS"` // loyalty calculation system address
}

// parse all flags from command line
func LoadConfig() *Config {
	var config Config

	err := godotenv.Load()
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("No .env file found, using environment variables and flags")
		} else {
			log.Fatalf("Failed to load .env file: %v", err)
		}
	}
	// read flags directly to config
	flag.StringVar(&config.RunAddr, "a", ":8080", "address and port to run server")
	flag.StringVar(&config.LoggerLevel, "l", "INFO", "logger level")
	flag.StringVar(&config.DatabaseUri, "d", "", "database connection url")
	flag.StringVar(&config.CookieSecret, "s", "", "cookie signing secret")
	flag.IntVar(&config.MaxParallelWorkers, "p", 5, "maximum amount of parallel workers")
	flag.StringVar(&config.AccrualSystemAddress, "r", "", "maximum concurrent parallel operations")
	flag.IntVar(&config.PollInterval, "i", 30, "external service poll interval")
	flag.BoolVar(&config.MockExternalService, "m", false, "mock external storage")
	// read flags to temp vars
	var readSec, writeSec int
	flag.IntVar(&readSec, "t", 30, "server read timeout in seconds")
	flag.IntVar(&writeSec, "w", 30, "server write timeout in seconds")
	// parse flags
	flag.Parse()
	// parse temp vars to config struct
	config.ReadTimeout = time.Duration(readSec) * time.Second
	config.WriteTimeout = time.Duration(writeSec) * time.Second
	if err := env.Parse(&config); err != nil {
		log.Printf("warning: failed to parse env: %v", err)
	}
	return &config
}
