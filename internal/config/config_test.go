package config

import (
	"os"
	"path/filepath"
	"testing"
)

// ============== LoadConfig 测试 ==============

func TestLoadConfig_NonExistentFile(t *testing.T) {
	cfg, err := LoadConfig("/nonexistent/config.ini")
	if err != nil {
		t.Fatalf("文件不存在时不应返回错误: %v", err)
	}

	if cfg.ColDelimiter != "|" {
		t.Errorf("默认分隔符应为 '|', 实际为 %s", cfg.ColDelimiter)
	}

	if cfg.XDRPaths == nil {
		t.Error("XDRPaths 不应为 nil")
	}

	if len(cfg.XDRPaths) != 0 {
		t.Error("XDRPaths 应为空")
	}
}

func TestLoadConfig_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test.ini")

	content := `[DEFAULT]
col_delimiter = ,

[XDR_PATH]
path1 = /path/to/data1
path2 = /path/to/data2
xdr_template_file = /path/to/template.xlsx
`
	if err := os.WriteFile(configFile, []byte(content), 0644); err != nil {
		t.Fatalf("创建测试配置文件失败: %v", err)
	}

	cfg, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("LoadConfig 失败: %v", err)
	}

	if cfg.ColDelimiter != "," {
		t.Errorf("分隔符应为 ',', 实际为 %s", cfg.ColDelimiter)
	}

	if len(cfg.XDRPaths) != 2 {
		t.Errorf("XDRPaths 应有 2 个条目, 实际为 %d", len(cfg.XDRPaths))
	}

	if cfg.XDRPaths["path1"] != "/path/to/data1" {
		t.Errorf("path1 应为 '/path/to/data1', 实际为 %s", cfg.XDRPaths["path1"])
	}

	if cfg.XDRPaths["path2"] != "/path/to/data2" {
		t.Errorf("path2 应为 '/path/to/data2', 实际为 %s", cfg.XDRPaths["path2"])
	}

	if cfg.TemplateFile != "/path/to/template.xlsx" {
		t.Errorf("TemplateFile 应为 '/path/to/template.xlsx', 实际为 %s", cfg.TemplateFile)
	}
}

func TestLoadConfig_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "empty.ini")

	if err := os.WriteFile(configFile, []byte(""), 0644); err != nil {
		t.Fatalf("创建测试配置文件失败: %v", err)
	}

	cfg, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("LoadConfig 失败: %v", err)
	}

	if cfg.ColDelimiter != "|" {
		t.Errorf("空文件应使用默认分隔符, 实际为 %s", cfg.ColDelimiter)
	}
}

func TestLoadConfig_OnlyDefaultSection(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "default_only.ini")

	content := `[other]
col_delimiter = ;
`
	if err := os.WriteFile(configFile, []byte(content), 0644); err != nil {
		t.Fatalf("创建测试配置文件失败: %v", err)
	}

	cfg, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("LoadConfig 失败: %v", err)
	}

	if cfg.ColDelimiter != "|" {
		t.Errorf("无 DEFAULT 节时应使用默认分隔符, 实际为 %s", cfg.ColDelimiter)
	}
}

func TestLoadConfig_OnlyXDRSection(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "xdr_only.ini")

	content := `[XDR_PATH]
test_path = /test/path
`
	if err := os.WriteFile(configFile, []byte(content), 0644); err != nil {
		t.Fatalf("创建测试配置文件失败: %v", err)
	}

	cfg, err := LoadConfig(configFile)
	if err != nil {
		t.Fatalf("LoadConfig 失败: %v", err)
	}

	if cfg.ColDelimiter != "|" {
		t.Errorf("应使用默认分隔符, 实际为 %s", cfg.ColDelimiter)
	}

	if cfg.XDRPaths["test_path"] != "/test/path" {
		t.Errorf("test_path 应为 '/test/path', 实际为 %s", cfg.XDRPaths["test_path"])
	}
}

// ============== GetXDRPath 测试 ==============

func TestGetXDRPath_Exists(t *testing.T) {
	cfg := &Config{
		XDRPaths: map[string]string{
			"path1": "/path/to/data1",
			"path2": "/path/to/data2",
		},
	}

	path := GetXDRPath(cfg, "path1")
	if path != "/path/to/data1" {
		t.Errorf("应返回 '/path/to/data1', 实际为 %s", path)
	}
}

func TestGetXDRPath_NotExists(t *testing.T) {
	cfg := &Config{
		XDRPaths: map[string]string{
			"path1": "/path/to/data1",
		},
	}

	path := GetXDRPath(cfg, "nonexistent")
	if path != "" {
		t.Errorf("不存在的键应返回空字符串, 实际为 %s", path)
	}
}

func TestGetXDRPath_EmptyConfig(t *testing.T) {
	cfg := &Config{
		XDRPaths: make(map[string]string),
	}

	path := GetXDRPath(cfg, "any")
	if path != "" {
		t.Errorf("空配置应返回空字符串, 实际为 %s", path)
	}
}

// ============== GetConfigFile 测试 ==============

func TestGetConfigFile_NoFileExists(t *testing.T) {
	tmpDir := t.TempDir()

	originalDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("切换目录失败: %v", err)
	}
	defer os.Chdir(originalDir)

	file := GetConfigFile()
	if file != "xdr_check.ini" {
		t.Errorf("无配置文件时应返回第一个配置文件, 实际为 %s", file)
	}
}

func TestGetConfigFile_FirstFileExists(t *testing.T) {
	tmpDir := t.TempDir()

	configFile := filepath.Join(tmpDir, "xdr_check.ini")
	if err := os.WriteFile(configFile, []byte(""), 0644); err != nil {
		t.Fatalf("创建测试配置文件失败: %v", err)
	}

	originalDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("切换目录失败: %v", err)
	}
	defer os.Chdir(originalDir)

	file := GetConfigFile()
	if file != "xdr_check.ini" {
		t.Errorf("应返回 'xdr_check.ini', 实际为 %s", file)
	}
}

func TestGetConfigFile_SecondFileExists(t *testing.T) {
	tmpDir := t.TempDir()

	configFile := filepath.Join(tmpDir, "xdr_check-AV.ini")
	if err := os.WriteFile(configFile, []byte(""), 0644); err != nil {
		t.Fatalf("创建测试配置文件失败: %v", err)
	}

	originalDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("切换目录失败: %v", err)
	}
	defer os.Chdir(originalDir)

	file := GetConfigFile()
	if file != "xdr_check-AV.ini" {
		t.Errorf("应返回 'xdr_check-AV.ini', 实际为 %s", file)
	}
}

func TestGetConfigFile_ThirdFileExists(t *testing.T) {
	tmpDir := t.TempDir()

	configFile := filepath.Join(tmpDir, "xdr_check-IOT.ini")
	if err := os.WriteFile(configFile, []byte(""), 0644); err != nil {
		t.Fatalf("创建测试配置文件失败: %v", err)
	}

	originalDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("切换目录失败: %v", err)
	}
	defer os.Chdir(originalDir)

	file := GetConfigFile()
	if file != "xdr_check-IOT.ini" {
		t.Errorf("应返回 'xdr_check-IOT.ini', 实际为 %s", file)
	}
}

func TestGetConfigFile_FourthFileExists(t *testing.T) {
	tmpDir := t.TempDir()

	configFile := filepath.Join(tmpDir, "xdr_check-IDC.ini")
	if err := os.WriteFile(configFile, []byte(""), 0644); err != nil {
		t.Fatalf("创建测试配置文件失败: %v", err)
	}

	originalDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("切换目录失败: %v", err)
	}
	defer os.Chdir(originalDir)

	file := GetConfigFile()
	if file != "xdr_check-IDC.ini" {
		t.Errorf("应返回 'xdr_check-IDC.ini', 实际为 %s", file)
	}
}

func TestGetConfigFile_Priority(t *testing.T) {
	tmpDir := t.TempDir()

	firstFile := filepath.Join(tmpDir, "xdr_check.ini")
	secondFile := filepath.Join(tmpDir, "xdr_check-AV.ini")

	if err := os.WriteFile(firstFile, []byte(""), 0644); err != nil {
		t.Fatalf("创建测试配置文件失败: %v", err)
	}
	if err := os.WriteFile(secondFile, []byte(""), 0644); err != nil {
		t.Fatalf("创建测试配置文件失败: %v", err)
	}

	originalDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("切换目录失败: %v", err)
	}
	defer os.Chdir(originalDir)

	file := GetConfigFile()
	if file != "xdr_check.ini" {
		t.Errorf("应优先返回第一个配置文件, 实际为 %s", file)
	}
}
