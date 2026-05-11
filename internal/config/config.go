package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-ini/ini"
)

type Config struct {
	ColDelimiter string
	XDRPaths     map[string]string
	TemplateFile string
	IniConfig    *ini.File // 保存INI配置引用
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

	// 保存INI配置引用
	cfg.IniConfig = iniCfg

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

func GetConfigValue(cfg *Config, configItem string) string {
	if cfg == nil || cfg.IniConfig == nil || configItem == "" {
		return ""
	}

	// 解析配置项路径，格式为 "section.key"
	parts := strings.SplitN(configItem, ".", 2)
	if len(parts) != 2 {
		return ""
	}

	sectionName := strings.TrimSpace(parts[0])
	keyName := strings.TrimSpace(parts[1])

	// 检查section是否存在
	if !cfg.IniConfig.HasSection(sectionName) {
		return ""
	}

	section := cfg.IniConfig.Section(sectionName)
	if !section.HasKey(keyName) {
		return ""
	}

	value := section.Key(keyName).String()
	return strings.TrimSpace(value)
}
