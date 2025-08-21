package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadYAML 从文件加载 YAML，并解析到 out（结构体指针）
// 使用示例：
//
//	var cfg AppConfig
//	err := LoadYAML("config.yaml", &cfg)
func LoadYAML(path string, out interface{}) error {
	if out == nil {
		return fmt.Errorf("out must not be nil")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}
	return ParseYAML(data, out)
}

// ParseYAML 从字节解析 YAML 到 out（结构体指针）
func ParseYAML(data []byte, out interface{}) error {
	if out == nil {
		return fmt.Errorf("out must not be nil")
	}
	return yaml.Unmarshal(data, out)
}
