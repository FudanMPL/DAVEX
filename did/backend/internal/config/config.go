package config

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server  ServerConfig  `yaml:"server"`
	Chain   ChainConfig   `yaml:"chain"`
	Storage StorageConfig `yaml:"storage"`
	Actors  []ActorConfig `yaml:"actors"`
}

type ServerConfig struct {
	Address                string   `yaml:"address"`
	ReadTimeoutSeconds     int      `yaml:"read_timeout_seconds"`
	WriteTimeoutSeconds    int      `yaml:"write_timeout_seconds"`
	ShutdownTimeoutSeconds int      `yaml:"shutdown_timeout_seconds"`
	AllowedOrigins         []string `yaml:"allowed_origins"`
}

type ChainConfig struct {
	ContractName string `yaml:"contract_name"`
	Timeout      int64  `yaml:"timeout"`
}

type StorageConfig struct {
	StatePath string `yaml:"state_path"`
	AuditPath string `yaml:"audit_path"`
}

type ActorConfig struct {
	Alias         string `yaml:"alias"`
	DID           string `yaml:"did"`
	Admin         bool   `yaml:"admin"`
	SDKConfigPath string `yaml:"sdk_config_path"`
	SDKWorkingDir string `yaml:"sdk_working_dir"`
	TokenSHA256   string `yaml:"token_sha256"`
}

func Load(path string) (Config, error) {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return Config{}, fmt.Errorf("resolve backend configuration path: %w", err)
	}
	path = absolutePath
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read backend configuration: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("decode backend configuration: %w", err)
	}
	base := filepath.Dir(path)
	if !filepath.IsAbs(cfg.Storage.StatePath) {
		cfg.Storage.StatePath = filepath.Clean(filepath.Join(base, cfg.Storage.StatePath))
	}
	if !filepath.IsAbs(cfg.Storage.AuditPath) {
		cfg.Storage.AuditPath = filepath.Clean(filepath.Join(base, cfg.Storage.AuditPath))
	}
	for index := range cfg.Actors {
		if !filepath.IsAbs(cfg.Actors[index].SDKConfigPath) {
			cfg.Actors[index].SDKConfigPath = filepath.Clean(filepath.Join(base, cfg.Actors[index].SDKConfigPath))
		}
		if cfg.Actors[index].SDKWorkingDir != "" && !filepath.IsAbs(cfg.Actors[index].SDKWorkingDir) {
			cfg.Actors[index].SDKWorkingDir = filepath.Clean(filepath.Join(base, cfg.Actors[index].SDKWorkingDir))
		}
	}
	if address := strings.TrimSpace(os.Getenv("DID_BACKEND_ADDRESS")); address != "" {
		cfg.Server.Address = address
	}
	if contract := strings.TrimSpace(os.Getenv("DID_CONTRACT_NAME")); contract != "" {
		cfg.Chain.ContractName = contract
	}
	if statePath := strings.TrimSpace(os.Getenv("DID_BACKEND_STATE")); statePath != "" {
		cfg.Storage.StatePath = statePath
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c *Config) Validate() error {
	if strings.TrimSpace(c.Server.Address) == "" {
		c.Server.Address = ":8081"
	}
	if c.Server.ReadTimeoutSeconds <= 0 {
		c.Server.ReadTimeoutSeconds = 15
	}
	if c.Server.WriteTimeoutSeconds <= 0 {
		c.Server.WriteTimeoutSeconds = 30
	}
	if c.Server.ShutdownTimeoutSeconds <= 0 {
		c.Server.ShutdownTimeoutSeconds = 10
	}
	if strings.TrimSpace(c.Chain.ContractName) == "" {
		return errors.New("chain.contract_name is required")
	}
	if strings.TrimSpace(c.Storage.StatePath) == "" || strings.TrimSpace(c.Storage.AuditPath) == "" {
		return errors.New("storage.state_path and storage.audit_path are required")
	}
	if len(c.Actors) == 0 {
		return errors.New("at least one actor is required")
	}
	seenAliases := make(map[string]struct{}, len(c.Actors))
	seenTokens := make(map[string]struct{}, len(c.Actors))
	for index := range c.Actors {
		actor := &c.Actors[index]
		actor.Alias = strings.TrimSpace(actor.Alias)
		actor.DID = strings.TrimSpace(actor.DID)
		actor.TokenSHA256 = strings.ToLower(strings.TrimSpace(actor.TokenSHA256))
		if actor.Alias == "" || actor.SDKConfigPath == "" || len(actor.TokenSHA256) != 64 {
			return fmt.Errorf("actor %d requires alias, SDK path and a 64-character token_sha256", index)
		}
		if decoded, err := hex.DecodeString(actor.TokenSHA256); err != nil || len(decoded) != sha256Size {
			return fmt.Errorf("actor %s token_sha256 is not a valid SHA-256 hexadecimal value", actor.Alias)
		}
		if _, exists := seenAliases[actor.Alias]; exists {
			return fmt.Errorf("duplicate actor alias %s", actor.Alias)
		}
		if _, exists := seenTokens[actor.TokenSHA256]; exists {
			return fmt.Errorf("duplicate actor token hash for %s", actor.Alias)
		}
		seenAliases[actor.Alias] = struct{}{}
		seenTokens[actor.TokenSHA256] = struct{}{}
	}
	return nil
}

const sha256Size = 32

func (c Config) ReadTimeout() time.Duration {
	return time.Duration(c.Server.ReadTimeoutSeconds) * time.Second
}
func (c Config) WriteTimeout() time.Duration {
	return time.Duration(c.Server.WriteTimeoutSeconds) * time.Second
}
func (c Config) ShutdownTimeout() time.Duration {
	return time.Duration(c.Server.ShutdownTimeoutSeconds) * time.Second
}
