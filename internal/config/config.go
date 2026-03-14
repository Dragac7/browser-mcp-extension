package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config holds configuration for the browser MCP server.
type Config struct {
	WSPort        int
	WSToken       string
	JSScriptsPath string
}

// execDir returns the directory containing the running executable.
func execDir() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(execPath), nil
}

func NewConfig() *Config {
	return &Config{
		WSPort:        9001,
		JSScriptsPath: "./resources/js_scripts",
	}
}

func (c *Config) validate() error {
	var errs []string
	if c.WSPort < 1024 || c.WSPort > 65535 {
		errs = append(errs, "WS_PORT must be between 1024 and 65535")
	}
	if _, err := os.Stat(c.JSScriptsPath); os.IsNotExist(err) {
		errs = append(errs, fmt.Sprintf("JS_SCRIPTS_PATH does not exist: %s", c.JSScriptsPath))
	}
	if len(errs) > 0 {
		return fmt.Errorf("validation error(s): %s", strings.Join(errs, ", "))
	}
	return nil
}

func (c *Config) Load() (*Config, error) {
	if tok := os.Getenv("WS_TOKEN"); tok != "" {
		c.WSToken = tok
	}
	if portStr := os.Getenv("WS_PORT"); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("invalid WS_PORT: %w", err)
		}
		c.WSPort = port
	}
	if scriptsPath := os.Getenv("JS_SCRIPTS_PATH"); scriptsPath != "" {
		absPath, err := filepath.Abs(scriptsPath)
		if err != nil {
			return nil, fmt.Errorf("invalid JS_SCRIPTS_PATH: %w", err)
		}
		c.JSScriptsPath = absPath
	} else {
		base, err := execDir()
		if err != nil {
			absPath, absErr := filepath.Abs(c.JSScriptsPath)
			if absErr != nil {
				return nil, fmt.Errorf("invalid default JS_SCRIPTS_PATH: %w", absErr)
			}
			c.JSScriptsPath = absPath
		} else {
			c.JSScriptsPath = filepath.Join(base, "resources", "js_scripts")
		}
	}
	return c, c.validate()
}
