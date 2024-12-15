package initialize

import (
	"github.com/FXAZfung/random-image/internal/config"
	"gopkg.in/yaml.v3"
	"os"
)

func InitConfig(filename string) error {
	// 打开 YAML 文件
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// 解析 YAML 文件内容
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config.MainConfig); err != nil {
		return err
	}
	return nil
}
