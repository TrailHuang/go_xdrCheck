package config

import (
	"fmt"
	"os"

	"github.com/go-ini/ini"
)

type Config struct {
	ColDelimiter string
	XDRPaths     map[string]string
	TemplateFile string
}

func LoadConfig(file string) (*Config, error) {
	cfg := &Config{
		ColDelimiter: "|",
		XDRPaths:     make(map[string]string),
	}

	// 如果文件不存在，使用默认配置
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return cfg, nil
	}

	iniCfg, err := ini.Load(file)
	if err != nil {
		return nil, fmt.Errorf("加载配置文件失败: %v", err)
	}

	// 读取默认分隔符
	if iniCfg.HasSection("DEFAULT") {
		defaultSection := iniCfg.Section("DEFAULT")
		if defaultSection.HasKey("col_delimiter") {
			cfg.ColDelimiter = defaultSection.Key("col_delimiter").String()
		}
	}

	// 读取XDR路径配置
	if iniCfg.HasSection("XDR_PATH") {
		xdrSection := iniCfg.Section("XDR_PATH")
		for _, key := range xdrSection.Keys() {
			if key.Name() == "xdr_template_file" {
				cfg.TemplateFile = key.String()
			} else {
				cfg.XDRPaths[key.Name()] = key.String()
			}
		}
	}

	return cfg, nil
}

func GetXDRPath(config *Config, pathName string) string {
	if path, exists := config.XDRPaths[pathName]; exists {
		return path
	}
	return ""
}

func GetConfigFile() string {
	// 按优先级查找配置文件
	configFiles := []string{
		"xdr_check.ini",
		"xdr_check-AV.ini",
		"xdr_check-IOT.ini",
		"xdr_check-IDC.ini",
	}

	for _, file := range configFiles {
		if _, err := os.Stat(file); err == nil {
			return file
		}
	}

	// 默认返回第一个配置文件
	return configFiles[0]
}