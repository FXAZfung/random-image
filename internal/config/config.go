package config

// Config 定义与 YAML 文件对应的结构体
type Config struct {
	App struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
	} `yaml:"app"`
	Server struct {
		Port string `yaml:"port"`
		Host string `yaml:"host"`
		Path string `yaml:"path"`
	} `yaml:"server"`
	File struct {
		Path string `yaml:"path"`
	} `yaml:"file"`
	Limit struct {
		Required bool `yaml:"required"`
		Rate     int  `yaml:"rate"`
		Bucket   int  `yaml:"bucket"`
	} `yaml:"limit"`
}

var MainConfig *Config
