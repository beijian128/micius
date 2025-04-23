package config

import (
	"os"

	"path/filepath"

	"gopkg.in/yaml.v3"
)

type RedisConfig struct {
	Max      int    `yaml:"max"`
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	MaxIdle  int    `yaml:"max_idle"`
	Db       int    `yaml:"db"`
}

// AppConfig  共用配置项
type AppConfig struct {
	Develop    bool                    `yaml:"develop"`
	AppID      int64                   `yaml:"appid"`
	GameArea   int32                   `yaml:"game_area"`
	DBGame     string                  `yaml:"db_game"`
	RedisNodes map[string]*RedisConfig `yaml:"redis"`
}

// LoadConfig 加载节点配置文件并检查有效性
func LoadConfig(fileName string) (*AppConfig, error) {
	cfg, err := ParseConfigFile(fileName)

	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func parseConfigData(data []byte) (*AppConfig, error) {
	var cfg AppConfig
	if err := yaml.Unmarshal([]byte(data), &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// ParseConfigFile ...
func ParseConfigFile(fileName string) (*AppConfig, error) {
	abs, err := filepath.Abs(fileName)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}

	cfg, err := parseConfigData(data)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
