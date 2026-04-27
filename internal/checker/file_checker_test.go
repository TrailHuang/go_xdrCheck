package checker

import (
	"os"
	"path/filepath"
	"testing"
)

// ============== FileCheck 测试 ==============

func TestFileCheck_NoConfig(t *testing.T) {
	fileTypeFlag := FileTypeFlag{}
	result := FileCheck("test.txt", "/path/to/test.txt", fileTypeFlag, "unknown")
	if result != "good" {
		t.Errorf("无配置时应返回 'good', 实际为 %s", result)
	}
}

func TestFileCheck_InvalidPrefix(t *testing.T) {
	fileTypeFlag := FileTypeFlag{
		"test": FileTypeConfig{
			Headers: []string{"valid_prefix_"},
		},
	}

	result := FileCheck("invalid_name.txt", "/path/to/invalid_name.txt", fileTypeFlag, "test")
	if result == "good" {
		t.Error("文件名不符合要求时应返回错误信息")
	}
}

func TestFileCheck_ValidPrefix(t *testing.T) {
	fileTypeFlag := FileTypeFlag{
		"test": FileTypeConfig{
			Headers: []string{"valid_prefix_"},
		},
	}

	result := FileCheck("valid_prefix_test.txt", "/path/to/valid_prefix_test.txt", fileTypeFlag, "test")
	if result != "good" {
		t.Errorf("文件名符合要求时应返回 'good', 实际为 %s", result)
	}
}

func TestFileCheck_InvalidSuffix(t *testing.T) {
	fileTypeFlag := FileTypeFlag{
		"test": FileTypeConfig{
			Headers: []string{"test_"},
			Suffix:  ".txt",
		},
	}

	result := FileCheck("test_file.csv", "/path/to/test_file.csv", fileTypeFlag, "test")
	if result == "good" {
		t.Error("文件后缀不符合要求时应返回错误信息")
	}
}

func TestFileCheck_ValidSuffix(t *testing.T) {
	fileTypeFlag := FileTypeFlag{
		"test": FileTypeConfig{
			Headers: []string{"test_"},
			Suffix:  ".txt",
		},
	}

	result := FileCheck("test_file.txt", "/path/to/test_file.txt", fileTypeFlag, "test")
	if result != "good" {
		t.Errorf("文件后缀符合要求时应返回 'good', 实际为 %s", result)
	}
}

func TestFileCheck_SuffixSkip(t *testing.T) {
	fileTypeFlag := FileTypeFlag{
		"test": FileTypeConfig{
			Headers: []string{"test_"},
			Suffix:  "不校验",
		},
	}

	result := FileCheck("test_file.csv", "/path/to/test_file.csv", fileTypeFlag, "test")
	if result != "good" {
		t.Errorf("后缀配置为'不校验'时应返回 'good', 实际为 %s", result)
	}
}

func TestFileCheck_FileSize(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_file.txt")

	content := make([]byte, 100)
	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	fileTypeFlag := FileTypeFlag{
		"test": FileTypeConfig{
			Headers:   []string{"test_"},
			SizeLimit: "50",
		},
	}

	result := FileCheck("test_file.txt", tmpFile, fileTypeFlag, "test")
	if result != "good" {
		t.Errorf("文件大小符合要求时应返回 'good', 实际为 %s", result)
	}
}

func TestFileCheck_FileSizeTooSmall(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_file.txt")

	content := make([]byte, 10)
	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	fileTypeFlag := FileTypeFlag{
		"test": FileTypeConfig{
			Headers:   []string{"test_"},
			SizeLimit: "50",
		},
	}

	result := FileCheck("test_file.txt", tmpFile, fileTypeFlag, "test")
	if result == "good" {
		t.Error("文件大小不符合要求时应返回错误信息")
	}
}

func TestFileCheck_FileSizeSkip(t *testing.T) {
	fileTypeFlag := FileTypeFlag{
		"test": FileTypeConfig{
			Headers:   []string{"test_"},
			SizeLimit: "不校验",
		},
	}

	result := FileCheck("test_file.txt", "/path/to/test_file.txt", fileTypeFlag, "test")
	if result != "good" {
		t.Errorf("大小配置为'不校验'时应返回 'good', 实际为 %s", result)
	}
}

func TestFileCheck_FileNotExist(t *testing.T) {
	fileTypeFlag := FileTypeFlag{
		"test": FileTypeConfig{
			Headers:   []string{"test_"},
			SizeLimit: "50",
		},
	}

	result := FileCheck("test_file.txt", "/nonexistent/path/test_file.txt", fileTypeFlag, "test")
	if result == "good" {
		t.Error("文件不存在时应返回错误信息")
	}
}

// ============== TraverseDirectory 测试 ==============

func TestTraverseDirectory_NoConfig(t *testing.T) {
	fileTypeFlag := FileTypeFlag{}
	_, _, err := TraverseDirectory("/tmp", fileTypeFlag, "unknown", 0)
	if err == nil {
		t.Error("无配置时应返回错误")
	}
}

func TestTraverseDirectory_NonExistentDir(t *testing.T) {
	fileTypeFlag := FileTypeFlag{
		"test": FileTypeConfig{
			Headers: []string{"test_"},
		},
	}

	files, count, err := TraverseDirectory("/nonexistent/dir", fileTypeFlag, "test", 0)
	if err != nil {
		t.Errorf("目录不存在时不应返回错误: %v", err)
	}
	if len(files) != 0 {
		t.Error("目录不存在时应返回空文件列表")
	}
	if count != 0 {
		t.Error("目录不存在时应返回计数 0")
	}
}

func TestTraverseDirectory_WithFiles(t *testing.T) {
	tmpDir := t.TempDir()

	validFiles := []string{"test_1.txt", "test_2.txt", "test_3.txt"}
	for _, name := range validFiles {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte("test content"), 0644); err != nil {
			t.Fatalf("创建测试文件失败: %v", err)
		}
	}

	invalidFiles := []string{"invalid_1.txt", "other_2.txt"}
	for _, name := range invalidFiles {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte("test content"), 0644); err != nil {
			t.Fatalf("创建测试文件失败: %v", err)
		}
	}

	fileTypeFlag := FileTypeFlag{
		"test": FileTypeConfig{
			Headers:      []string{"test_"},
			CheckContent: "校验",
		},
	}

	files, count, err := TraverseDirectory(tmpDir, fileTypeFlag, "test", 0)
	if err != nil {
		t.Fatalf("TraverseDirectory 失败: %v", err)
	}

	if len(files) != len(validFiles) {
		t.Errorf("应找到 %d 个文件, 实际为 %d", len(validFiles), len(files))
	}

	if count != len(validFiles) {
		t.Errorf("计数应为 %d, 实际为 %d", len(validFiles), count)
	}
}

func TestTraverseDirectory_NoContentCheck(t *testing.T) {
	tmpDir := t.TempDir()

	files := []string{"test_1.txt", "test_2.txt"}
	for _, name := range files {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte("test content"), 0644); err != nil {
			t.Fatalf("创建测试文件失败: %v", err)
		}
	}

	fileTypeFlag := FileTypeFlag{
		"test": FileTypeConfig{
			Headers:      []string{"test_"},
			CheckContent: "",
		},
	}

	result, count, err := TraverseDirectory(tmpDir, fileTypeFlag, "test", 0)
	if err != nil {
		t.Fatalf("TraverseDirectory 失败: %v", err)
	}

	if len(result) != 0 {
		t.Error("不校验内容时应返回空文件列表")
	}

	if count != len(files) {
		t.Errorf("计数应为 %d, 实际为 %d", len(files), count)
	}
}

func TestTraverseDirectory_Sampling(t *testing.T) {
	tmpDir := t.TempDir()

	for i := 0; i < 10; i++ {
		name := "test_" + string(rune('0'+i)) + ".txt"
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte("test content"), 0644); err != nil {
			t.Fatalf("创建测试文件失败: %v", err)
		}
	}

	fileTypeFlag := FileTypeFlag{
		"test": FileTypeConfig{
			Headers:      []string{"test_"},
			CheckContent: "校验",
		},
	}

	files, _, err := TraverseDirectory(tmpDir, fileTypeFlag, "test", 5)
	if err != nil {
		t.Fatalf("TraverseDirectory 失败: %v", err)
	}

	if len(files) != 5 {
		t.Errorf("抽样应返回 5 个文件, 实际为 %d", len(files))
	}
}

func TestTraverseDirectory_FullScan(t *testing.T) {
	tmpDir := t.TempDir()

	for i := 0; i < 3; i++ {
		name := "test_" + string(rune('0'+i)) + ".txt"
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte("test content"), 0644); err != nil {
			t.Fatalf("创建测试文件失败: %v", err)
		}
	}

	fileTypeFlag := FileTypeFlag{
		"test": FileTypeConfig{
			Headers:      []string{"test_"},
			CheckContent: "校验",
		},
	}

	files, _, err := TraverseDirectory(tmpDir, fileTypeFlag, "test", 10)
	if err != nil {
		t.Fatalf("TraverseDirectory 失败: %v", err)
	}

	if len(files) != 3 {
		t.Errorf("文件数小于抽样数时应全量扫描, 应返回 3 个文件, 实际为 %d", len(files))
	}
}

// ============== sampleFiles 测试 ==============

func TestSampleFiles_LessThanNum(t *testing.T) {
	files := []string{"file1.txt", "file2.txt", "file3.txt"}
	result := sampleFiles(files, 5)

	if len(result) != len(files) {
		t.Errorf("抽样数大于文件数时应返回所有文件, 实际为 %d", len(result))
	}
}

func TestSampleFiles_EqualNum(t *testing.T) {
	files := []string{"file1.txt", "file2.txt", "file3.txt"}
	result := sampleFiles(files, 3)

	if len(result) != len(files) {
		t.Errorf("抽样数等于文件数时应返回所有文件, 实际为 %d", len(result))
	}
}

func TestSampleFiles_MoreThanNum(t *testing.T) {
	files := []string{"file1.txt", "file2.txt", "file3.txt", "file4.txt", "file5.txt"}
	result := sampleFiles(files, 2)

	if len(result) != 2 {
		t.Errorf("应返回 2 个文件, 实际为 %d", len(result))
	}
}

func TestSampleFiles_Empty(t *testing.T) {
	files := []string{}
	result := sampleFiles(files, 2)

	if len(result) != 0 {
		t.Errorf("空文件列表应返回空结果, 实际为 %d", len(result))
	}
}

func TestSampleFiles_Distribution(t *testing.T) {
	files := []string{"file1.txt", "file2.txt", "file3.txt", "file4.txt", "file5.txt", "file6.txt", "file7.txt", "file8.txt", "file9.txt", "file10.txt"}
	result := sampleFiles(files, 3)

	if len(result) != 3 {
		t.Errorf("应返回 3 个文件, 实际为 %d", len(result))
	}

	if result[0] != files[0] {
		t.Errorf("第一个抽样应为 file1.txt, 实际为 %s", result[0])
	}
}
