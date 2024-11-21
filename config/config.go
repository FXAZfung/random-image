package config

import (
	"gopkg.in/yaml.v3"
	"os"
)

// Config 定义与 YAML 文件对应的结构体
type Config struct {
	App struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
	} `yaml:"app"`
	Server struct {
		Port string `yaml:"port"`
		Host string `yaml:"host"`
	} `yaml:"server"`
	File struct {
		Path string `yaml:"path"`
	} `yaml:"file"`
}

var MainConfig *Config

func InitConfig(filename string) error {
	// 打开 YAML 文件
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// 解析 YAML 文件内容
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&MainConfig); err != nil {
		return err
	}
	return nil
}
