package config

import (
	"errors"
	"fmt"
	"net/mail"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	PG    pgConfig
	HTTP  httpConfig
	SMTP  smtpConfig
	River riverConfig
}

type pgConfig struct {
	Name     string
	Port     int
	Host     string
	Schema   string
	Username string
	Password string
}

type httpConfig struct {
	Port            int
	Limiter         limiterConfig
	IdleTimeout     time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

type smtpConfig struct {
	Host     string
	Port     int
	From     mail.Address
	Username string
	Password string
}

type riverConfig struct {
	MaxWorkers      int
	JobTimeout      time.Duration
	SoftStopTimeout time.Duration
}

type limiterConfig struct {
	Burst   int
	RPS     float64
	Enabled bool
}

func defaultConfig() Config {
	return Config{
		PG: pgConfig{
			Port:   5432,
			Schema: "public",
		},
		HTTP: httpConfig{
			Port: 11880,
			Limiter: limiterConfig{
				RPS:     2,
				Burst:   4,
				Enabled: true,
			},
			IdleTimeout:     2 * time.Minute,
			ReadTimeout:     5 * time.Second,
			WriteTimeout:    10 * time.Second,
			ShutdownTimeout: 20 * time.Second,
		},
		SMTP: smtpConfig{
			Port: 587,
			From: mail.Address{
				Name:    "Roy Parsons",
				Address: "roy.parsons@blipsnchitz.com",
			},
		},
		River: riverConfig{
			MaxWorkers:      100,
			JobTimeout:      time.Minute,
			SoftStopTimeout: 10 * time.Second,
		},
	}
}

type envVariable struct {
	req     bool
	reqIf   func(c Config) bool
	mapFunc func(v string, c *Config) error
}

const maxStringLength = 512

var envMap = map[string]envVariable{
	"PG_NAME": {
		req: true,
		mapFunc: func(v string, c *Config) error {
			return confString(v, &c.PG.Name, 1, maxStringLength)
		},
	},

	"PG_PORT": {
		mapFunc: func(v string, c *Config) error {
			return confInt(v, &c.PG.Port, 1, 65535)
		},
	},

	"PG_HOST": {
		req: true,
		mapFunc: func(v string, c *Config) error {
			return confString(v, &c.PG.Host, 1, maxStringLength)
		},
	},

	"PG_SCHEMA": {
		mapFunc: func(v string, c *Config) error {
			return confString(v, &c.PG.Schema, 1, maxStringLength)
		},
	},

	"PG_USERNAME": {
		req: true,
		mapFunc: func(v string, c *Config) error {
			return confString(v, &c.PG.Username, 1, maxStringLength)
		},
	},

	"PG_PASSWORD": {
		req: true,
		mapFunc: func(v string, c *Config) error {
			return confString(v, &c.PG.Password, 1, maxStringLength)
		},
	},

	"HTTP_PORT": {
		mapFunc: func(v string, c *Config) error {
			return confInt(v, &c.HTTP.Port, 1, 65535)
		},
	},

	"HTTP_LIMITER_BURST": {
		mapFunc: func(v string, c *Config) error {
			return confInt(v, &c.HTTP.Limiter.Burst, 1, 10_000)
		},
	},

	"HTTP_LIMITER_RPS": {
		mapFunc: func(v string, c *Config) error {
			return confFloat(v, &c.HTTP.Limiter.RPS, 0.1, 10_000)
		},
	},

	"HTTP_LIMITER_ENABLED": {
		mapFunc: func(v string, c *Config) error {
			return confBool(v, &c.HTTP.Limiter.Enabled)
		},
	},

	"HTTP_IDLE_TIMEOUT": {
		mapFunc: func(v string, c *Config) error {
			return confDuration(v, &c.HTTP.IdleTimeout, 30*time.Second, 10*time.Minute)
		},
	},

	"HTTP_READ_TIMEOUT": {
		mapFunc: func(v string, c *Config) error {
			return confDuration(v, &c.HTTP.ReadTimeout, time.Second, time.Minute)
		},
	},

	"HTTP_WRITE_TIMEOUT": {
		mapFunc: func(v string, c *Config) error {
			return confDuration(v, &c.HTTP.WriteTimeout, time.Second, time.Minute)
		},
	},

	"HTTP_SHUTDOWN_TIMEOUT": {
		mapFunc: func(v string, c *Config) error {
			return confDuration(v, &c.HTTP.ShutdownTimeout, time.Second, 30*time.Second)
		},
	},

	"SMTP_HOST": {
		req: true,
		mapFunc: func(v string, c *Config) error {
			return confString(v, &c.SMTP.Host, 1, maxStringLength)
		},
	},

	"SMTP_PORT": {
		mapFunc: func(v string, c *Config) error {
			return confInt(v, &c.SMTP.Port, 1, 65535)
		},
	},

	"SMTP_FROM": {
		mapFunc: func(v string, c *Config) error {
			return confEmailAddress(v, &c.SMTP.From)
		},
	},

	"SMTP_USERNAME": {
		req: true,
		mapFunc: func(v string, c *Config) error {
			return confString(v, &c.SMTP.Username, 1, maxStringLength)
		},
	},

	"SMTP_PASSWORD": {
		req: true,
		mapFunc: func(v string, c *Config) error {
			return confString(v, &c.SMTP.Password, 1, maxStringLength)
		},
	},

	"RIVER_MAX_WORKERS": {
		mapFunc: func(v string, c *Config) error {
			return confInt(v, &c.River.MaxWorkers, 1, 10_000)
		},
	},

	"RIVER_JOB_TIMEOUT": {
		mapFunc: func(v string, c *Config) error {
			return confDuration(v, &c.River.JobTimeout, 10*time.Second, 5*time.Minute)
		},
	},

	"RIVER_SOFT_STOP_TIMEOUT": {
		mapFunc: func(v string, c *Config) error {
			return confDuration(v, &c.River.SoftStopTimeout, time.Second, 30*time.Second)
		},
	},
}

func LoadFromEnv() (Config, error) {
	var errMap error

	config := defaultConfig()
	exists := make(map[string]bool, len(envMap))

	for key, variable := range envMap {
		value, ok := os.LookupEnv(key)
		if !ok {
			continue
		}

		exists[key] = true

		if err := variable.mapFunc(value, &config); err != nil {
			errMap = errors.Join(errMap, fmt.Errorf("%s: %w", key, err))
		}
	}

	for key, variable := range envMap {
		if exists[key] {
			continue
		}

		if variable.req || (variable.reqIf != nil && variable.reqIf(config)) {
			errMap = errors.Join(errMap, fmt.Errorf("%s: missing required environment variable", key))
		}
	}

	return config, errMap
}

func confInt(v string, out *int, min, max int) error {
	i, err := strconv.Atoi(v)
	if err != nil {
		return fmt.Errorf("parse int: %w", err)
	}

	if i < min || i > max {
		return fmt.Errorf("int %d not in range [%d, %d] (inclusive)", i, min, max)
	}

	*out = i
	return nil
}

func confBool(v string, out *bool) error {
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fmt.Errorf("parse bool: %w", err)
	}

	*out = b
	return nil
}

func confFloat(v string, out *float64, min, max float64) error {
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fmt.Errorf("parse float: %w", err)
	}

	if f < min || f > max {
		return fmt.Errorf("float %g not in range [%g, %g] (inclusive)", f, min, max)
	}

	*out = f
	return nil
}

func confDuration(v string, out *time.Duration, min, max time.Duration) error {
	d, err := time.ParseDuration(v)
	if err != nil {
		return fmt.Errorf("parse duration: %w", err)
	}

	if d < min || d > max {
		return fmt.Errorf("duration %s not in range [%s, %s] (inclusive)", d, min, max)
	}

	*out = d
	return nil
}

func confEmailAddress(v string, out *mail.Address) error {
	email, err := mail.ParseAddress(v)
	if err != nil {
		return fmt.Errorf("parse email address: %w", err)
	}

	*out = *email
	return nil
}

func confString(v string, out *string, minLen, maxLen int) error {
	trimmed := strings.TrimSpace(v)

	if len(trimmed) < minLen || len(trimmed) > maxLen {
		return fmt.Errorf("string length %d not in range [%d, %d] (inclusive)", len(trimmed), minLen, maxLen)
	}

	*out = trimmed
	return nil
}
