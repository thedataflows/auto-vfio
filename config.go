package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	supportedConfigFormats = []string{".json", ".yaml", ".yml", ".toml"}
	LogFieldModuleConfig   = []any{"module", "config"}
	LogLevels              = []string{zerolog.LevelTraceValue, zerolog.LevelDebugValue, zerolog.LevelInfoValue, zerolog.LevelWarnValue, zerolog.LevelErrorValue, zerolog.LevelFatalValue, zerolog.LevelPanicValue}
	DefaultLogLevel        = zerolog.InfoLevel
)

type configFile string

func (c configFile) AfterApply(cli *CLI) error {
	ext := filepath.Ext(c.String())
	if !slices.Contains(supportedConfigFormats, ext) {
		return fmt.Errorf("unsupported config file format %q. Supported formats: %s", ext, strings.Join(supportedConfigFormats, ", "))
	}
	return nil
}
func (c configFile) String() string {
	return string(c)
}

type Config struct {
	logger zerolog.Logger
}

type Option func(*Config) error

type Globals struct {
	ConfigFile configFile `short:"c" help:"Config file location. Supported formats: ${supported_formats}" default:"default.yaml" type:"path"`
	LogLevel   string     `short:"l" help:"Logging level. One of: ${log_levels}" default:"${default_log_level}"`

	config *Config
}

type CLI struct {
	Globals
	kong.Plugins
}

// NewConfig creates a new Config
func NewConfig(opts ...Option) (*Config, error) {
	c := &Config{
		logger: log.Output(
			zerolog.ConsoleWriter{
				Out:        os.Stderr,
				TimeFormat: time.TimeOnly,
				NoColor:    false,
			},
		),
	}

	var allErrors error
	for _, opt := range opts {
		if err := opt(c); err != nil {
			allErrors = errors.Join(allErrors, err)
		}
	}

	return c, allErrors
}

// Logger returns the config logger
func (c *Config) Logger() *zerolog.Logger {
	return &c.logger
}

// WithLogLevel sets the log level Option
func WithLogLevel(level string) Option {
	return func(c *Config) error {
		parsedLevel, err := zerolog.ParseLevel(level)
		if err != nil {
			c.Logger().Warn().Fields(LogFieldModuleConfig).
				Msgf("Unknown log level %q. Defaulting to %q", level, DefaultLogLevel)
			parsedLevel = DefaultLogLevel
		}
		if parsedLevel == zerolog.NoLevel {
			parsedLevel = DefaultLogLevel
		}
		c.logger = c.Logger().Level(parsedLevel)
		return nil
	}
}
