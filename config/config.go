package config

import (
	"encoding/json"
	"os"
)

// Config 配置结构
type Config struct {
	Server ServerConfig `json:"server"`
	Douyin DouyinConfig `json:"douyin"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host string `json:"host"`
	Port string `json:"port"`
}

// DouyinConfig 抖音配置
type DouyinConfig struct {
	UserAgent string `json:"user_agent"`
	Origin    string `json:"origin"`
	Timeout   int    `json:"timeout"`
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: "12000",
		},
		Douyin: DouyinConfig{
			UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
			Origin:    "https://live.douyin.com",
			Timeout:   30,
		},
	}
}

// LoadConfig 加载配置文件
func LoadConfig(filename string) (*Config, error) {
	config := DefaultConfig()

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return config, nil
	}

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(config); err != nil {
		return nil, err
	}

	return config, nil
}

// SaveConfig 保存配置文件
func (c *Config) SaveConfig(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(c)
}