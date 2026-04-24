package core

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"xdrCheck/internal/config"
	"xdrCheck/internal/parser"
)

// ============== 辅助函数 ==============

// createTestFile 创建临时测试文件并返回路径
func createTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}
	return path
}

// createTestTarGz 创建 .tar.gz 测试文件
func createTestTarGz(t *testing.T, dir, archiveName, innerFileName, content string) string {
	t.Helper()
	archivePath := filepath.Join(dir, archiveName)
	outFile, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("创建tar.gz文件失败: %v", err)
	}
	defer outFile.Close()

	gzWriter := gzip.NewWriter(outFile)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	hdr := &tar.Header{
		Name: innerFileName,
		Mode: 0644,
		Size: int64(len(content)),
	}
	if err := tarWriter.WriteHeader(hdr); err != nil {
		t.Fatalf("写入tar头失败: %v", err)
	}
	if _, err := tarWriter.Write([]byte(content)); err != nil {
		t.Fatalf("写入tar内容失败: %v", err)
	}

	return archivePath
}

// createTestGz 创建 .gz 测试文件
func createTestGz(t *testing.T, dir, archiveName, content string) string {
	t.Helper()
	archivePath := filepath.Join(dir, archiveName)
	outFile, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("创建gz文件失败: %v", err)
	}
	defer outFile.Close()

	gzWriter := gzip.NewWriter(outFile)
	if _, err := gzWriter.Write([]byte(content)); err != nil {
		t.Fatalf("写入gz内容失败: %v", err)
	}
	gzWriter.Close()

	return archivePath
}

// newTestXDRChecker 创建测试用的 XDRChecker
func newTestXDRChecker(delimiter string) *XDRChecker {
	tmpFile, _ := os.CreateTemp("", "result_*.txt")
	return &XDRChecker{
		Config: &config.Config{
			ColDelimiter: delimiter,
			XDRPaths:     make(map[string]string),
		},
		TimeParam:    "20260417",
		WorkerNum:    2,
		ReportFormat: "txt",
		ResultFile:   tmpFile,
	}
}

// ============== NewXDRChecker 测试 ==============

func TestNewXDRChecker_DefaultWorkerNum(t *testing.T) {
	cfg := CheckerConfig{
		Config:       &config.Config{},
		WorkerNum:    0, // 应该使用默认值4
		ReportFormat: "txt",
	}
	checker := NewXDRChecker(cfg)
	if checker.WorkerNum != 4 {
		t.Errorf("默认WorkerNum应为4, 实际为%d", checker.WorkerNum)
	}
}

func TestNewXDRChecker_NegativeWorkerNum(t *testing.T) {
	cfg := CheckerConfig{
		Config:       &config.Config{},
		WorkerNum:    -1,
		ReportFormat: "txt",
	}
	checker := NewXDRChecker(cfg)
	if checker.WorkerNum != 4 {
		t.Errorf("负数WorkerNum应默认为4, 实际为%d", checker.WorkerNum)
	}
}

func TestNewXDRChecker_CustomWorkerNum(t *testing.T) {
	cfg := CheckerConfig{
		Config:       &config.Config{},
		WorkerNum:    8,
		ReportFormat: "txt",
	}
	checker := NewXDRChecker(cfg)
	if checker.WorkerNum != 8 {
		t.Errorf("自定义WorkerNum应为8, 实际为%d", checker.WorkerNum)
	}
}

func TestNewXDRChecker_DefaultReportFormat(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		expected string
	}{
		{"txt格式", "txt", "txt"},
		{"table格式", "table", "table"},
		{"html格式", "html", "html"},
		{"无效格式默认txt", "invalid", "txt"},
		{"空格式默认txt", "", "txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := CheckerConfig{
				Config:       &config.Config{},
				WorkerNum:    4,
				ReportFormat: tt.format,
			}
			checker := NewXDRChecker(cfg)
			if checker.ReportFormat != tt.expected {
				t.Errorf("ReportFormat应为%s, 实际为%s", tt.expected, checker.ReportFormat)
			}
		})
	}
}

func TestNewXDRCheckerLegacy(t *testing.T) {
	cfg := &config.Config{}
	checker := NewXDRCheckerLegacy(cfg, "20260101", 10, false, 4, "table")
	if checker.TimeParam != "20260101" || checker.ScanNum != 10 || checker.WorkerNum != 4 || checker.ReportFormat != "table" {
		t.Errorf("NewXDRCheckerLegacy参数传递不正确")
	}
}

// ============== DefaultTableReportConfig 测试 ==============

func TestDefaultTableReportConfig(t *testing.T) {
	cfg := DefaultTableReportConfig()
	if !cfg.ShowFileName || !cfg.ShowLineNumber || !cfg.ShowFieldValue || !cfg.ShowErrorType || !cfg.ShowCondition {
		t.Error("默认配置应全部显示")
	}
	if cfg.MaxColumnWidth != 30 {
		t.Errorf("MaxColumnWidth应为30, 实际为%d", cfg.MaxColumnWidth)
	}
}

// ============== normalizeSheetName 测试 ==============

func TestNormalizeSheetName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"去除前后空格", "  abc  ", "abc"},
		{"标准化+号周围空格", "0x31 + 0x03a0", "0x31+0x03a0"},
		{"+号前有空格", "0x31 +0x03a0", "0x31+0x03a0"},
		{"+号后有空格", "0x31+ 0x03a0", "0x31+0x03a0"},
		{"去除所有空格", "a b c", "abc"},
		{"无空格不变", "abc", "abc"},
		{"空字符串", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeSheetName(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeSheetName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// ============== ternary 测试 ==============

func TestTernary(t *testing.T) {
	if ternary(true, 1, 2) != 1 {
		t.Error("ternary(true, 1, 2) 应返回1")
	}
	if ternary(false, 1, 2) != 2 {
		t.Error("ternary(false, 1, 2) 应返回2")
	}
}

// ============== isSpecialPath 测试 ==============

func TestIsSpecialPath(t *testing.T) {
	checker := newTestXDRChecker("|")

	if !checker.isSpecialPath("local_to_cu_0x01e0") {
		t.Error("local_to_cu_0x01e0 应为特殊路径")
	}

	if checker.isSpecialPath("normal_path") {
		t.Error("normal_path 不应为特殊路径")
	}

	if checker.isSpecialPath("") {
		t.Error("空路径不应为特殊路径")
	}
}

// ============== findSheetConfig 测试 ==============

func TestFindSheetConfig_ExactMatch(t *testing.T) {
	checker := newTestXDRChecker("|")
	sheetConfigs := []parser.SheetConfig{
		{SheetName: "path_a"},
		{SheetName: "path_b"},
		{SheetName: "path_c"},
	}

	sc, found := checker.findSheetConfig(sheetConfigs, "path_b")
	if !found || sc.SheetName != "path_b" {
		t.Error("精确匹配应找到 path_b")
	}
}

func TestFindSheetConfig_NotFound(t *testing.T) {
	checker := newTestXDRChecker("|")
	sheetConfigs := []parser.SheetConfig{
		{SheetName: "path_a"},
	}

	_, found := checker.findSheetConfig(sheetConfigs, "path_z")
	if found {
		t.Error("不存在的路径不应匹配")
	}
}

func TestFindSheetConfig_FuzzyMatch_TrimSpaces(t *testing.T) {
	checker := newTestXDRChecker("|")
	sheetConfigs := []parser.SheetConfig{
		{SheetName: "path a"},
	}

	sc, found := checker.findSheetConfig(sheetConfigs, "patha")
	if !found || sc.SheetName != "path a" {
		t.Error("去除空格后应匹配")
	}
}

func TestFindSheetConfig_FuzzyMatch_Contains(t *testing.T) {
	checker := newTestXDRChecker("|")
	sheetConfigs := []parser.SheetConfig{
		{SheetName: "0x31+0x03a0"},
	}

	sc, found := checker.findSheetConfig(sheetConfigs, "0x31")
	if !found {
		t.Errorf("包含关系应匹配: %v", sc.SheetName)
	}
}

func TestFindSheetConfig_EmptyConfigs(t *testing.T) {
	checker := newTestXDRChecker("|")
	_, found := checker.findSheetConfig([]parser.SheetConfig{}, "anything")
	if found {
		t.Error("空配置不应匹配")
	}
}

// ============== decodeBase64 测试 ==============

func TestDecodeBase64(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"有效base64", "aGVsbG8=", "hello", false},
		{"空字符串", "", "", false}, // 空字符串解码后仍为空
		{"无效base64", "!!!", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeBase64(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeBase64(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("decodeBase64(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ============== combinedReader 测试 ==============

func TestCombinedReader_Close(t *testing.T) {
	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()

	cr := &combinedReader{
		Reader:  r1,
		closers: []io.Closer{r1, r2},
	}

	// 关闭应该不返回错误
	if err := cr.Close(); err != nil {
		t.Errorf("combinedReader.Close() 不应返回错误: %v", err)
	}

	// 写入端应该收到关闭信号
	w1.Close()
	w2.Close()
}

func TestCombinedReader_CloseError(t *testing.T) {
	cr := &combinedReader{
		Reader: strings.NewReader("test"),
		closers: []io.Closer{},
	}

	// 无 closer 时不应该出错
	if err := cr.Close(); err != nil {
		t.Errorf("无closer时Close不应返回错误: %v", err)
	}
}

func TestCombinedReader_ReadAndClose(t *testing.T) {
	content := "hello world"
	cr := &combinedReader{
		Reader:  strings.NewReader(content),
		closers: []io.Closer{},
	}

	data, err := io.ReadAll(cr)
	if err != nil {
		t.Fatalf("读取失败: %v", err)
	}
	if string(data) != content {
		t.Errorf("读取内容不匹配: got %q, want %q", string(data), content)
	}
}

// ============== openFile 测试 ==============

func TestOpenFile_TxtFile(t *testing.T) {
	tmpDir := t.TempDir()
	content := "line1\nline2\n"
	path := createTestFile(t, tmpDir, "test.txt", content)

	checker := newTestXDRChecker("|")
	file, err := checker.openFile(path)
	if err != nil {
		t.Fatalf("打开txt文件失败: %v", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("读取文件失败: %v", err)
	}
	if string(data) != content {
		t.Errorf("文件内容不匹配: got %q, want %q", string(data), content)
	}
}

func TestOpenFile_LogFile(t *testing.T) {
	tmpDir := t.TempDir()
	content := "log line\n"
	path := createTestFile(t, tmpDir, "test.log", content)

	checker := newTestXDRChecker("|")
	file, err := checker.openFile(path)
	if err != nil {
		t.Fatalf("打开log文件失败: %v", err)
	}
	defer file.Close()
}

func TestOpenFile_CsvFile(t *testing.T) {
	tmpDir := t.TempDir()
	content := "a,b,c\n"
	path := createTestFile(t, tmpDir, "test.csv", content)

	checker := newTestXDRChecker("|")
	file, err := checker.openFile(path)
	if err != nil {
		t.Fatalf("打开csv文件失败: %v", err)
	}
	defer file.Close()
}

func TestOpenFile_GzFile(t *testing.T) {
	tmpDir := t.TempDir()
	content := "gz content line\n"
	createTestGz(t, tmpDir, "test.gz", content)
	gzPath := filepath.Join(tmpDir, "test.gz")

	checker := newTestXDRChecker("|")
	file, err := checker.openFile(gzPath)
	if err != nil {
		t.Fatalf("打开gz文件失败: %v", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("读取gz文件失败: %v", err)
	}
	if string(data) != content {
		t.Errorf("gz文件内容不匹配: got %q, want %q", string(data), content)
	}
}

func TestOpenFile_TarGzFile(t *testing.T) {
	tmpDir := t.TempDir()
	content := "tar.gz content\n"
	createTestTarGz(t, tmpDir, "test.tar.gz", "inner.txt", content)
	tarGzPath := filepath.Join(tmpDir, "test.tar.gz")

	checker := newTestXDRChecker("|")
	file, err := checker.openFile(tarGzPath)
	if err != nil {
		t.Fatalf("打开tar.gz文件失败: %v", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("读取tar.gz文件失败: %v", err)
	}
	if string(data) != content {
		t.Errorf("tar.gz文件内容不匹配: got %q, want %q", string(data), content)
	}
}

func TestOpenFile_TarFile(t *testing.T) {
	tmpDir := t.TempDir()
	content := "tar content\n"
	archivePath := filepath.Join(tmpDir, "test.tar")
	outFile, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("创建tar文件失败: %v", err)
	}

	tarWriter := tar.NewWriter(outFile)
	hdr := &tar.Header{
		Name: "inner.txt",
		Mode: 0644,
		Size: int64(len(content)),
	}
	tarWriter.WriteHeader(hdr)
	tarWriter.Write([]byte(content))
	tarWriter.Close()
	outFile.Close()

	checker := newTestXDRChecker("|")
	file, err := checker.openFile(archivePath)
	if err != nil {
		t.Fatalf("打开tar文件失败: %v", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("读取tar文件失败: %v", err)
	}
	if string(data) != content {
		t.Errorf("tar文件内容不匹配: got %q, want %q", string(data), content)
	}
}

func TestOpenCompressedFile_TarGzNoRegularFile(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "empty.tar.gz")
	outFile, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("创建tar.gz文件失败: %v", err)
	}
	gzWriter := gzip.NewWriter(outFile)
	tarWriter := tar.NewWriter(gzWriter)
	// 只写一个目录头，不写普通文件
	hdr := &tar.Header{
		Name:     "dir",
		Mode:     0755,
		Typeflag: tar.TypeDir,
	}
	tarWriter.WriteHeader(hdr)
	tarWriter.Close()
	gzWriter.Close()
	outFile.Close()

	checker := newTestXDRChecker("|")
	_, err = checker.openFile(archivePath)
	if err == nil {
		t.Error("tar.gz中无普通文件时应返回错误")
	}
}

func TestOpenCompressedFile_TarNoRegularFile(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "empty.tar")
	outFile, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("创建tar文件失败: %v", err)
	}
	tarWriter := tar.NewWriter(outFile)
	hdr := &tar.Header{
		Name:     "dir",
		Mode:     0755,
		Typeflag: tar.TypeDir,
	}
	tarWriter.WriteHeader(hdr)
	tarWriter.Close()
	outFile.Close()

	checker := newTestXDRChecker("|")
	_, err = checker.openFile(archivePath)
	if err == nil {
		t.Error("tar中无普通文件时应返回错误")
	}
}

func TestOpenCompressedFile_InvalidGz(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "invalid.gz")
	// 写入无效的gzip数据
	os.WriteFile(archivePath, []byte("not a gzip file"), 0644)

	checker := newTestXDRChecker("|")
	_, err := checker.openFile(archivePath)
	if err == nil {
		t.Error("无效gz文件应返回错误")
	}
}

func TestOpenFile_NonExistentFile(t *testing.T) {
	checker := newTestXDRChecker("|")
	_, err := checker.openFile("/nonexistent/path/file.txt")
	if err == nil {
		t.Error("打开不存在的文件应返回错误")
	}
}

// ============== checkSingleFileContent 测试 ==============

func TestCheckSingleFileContent_BasicValidation(t *testing.T) {
	tmpDir := t.TempDir()
	// 构造3个字段的测试数据，分隔符为|
	content := "field1|field2|field3\n"
	path := createTestFile(t, tmpDir, "test.txt", content)

	checker := newTestXDRChecker("|")

	sheetConfig := parser.SheetConfig{
		SheetName:      "test",
		FieldNumberMap: map[string]int{"1": 0, "2": 1, "3": 2},
		FieldRules: []parser.FieldRule{
			{FieldName: "F1", Required: "必填", Type: ""},
			{FieldName: "F2", Required: "选填", Type: ""},
			{FieldName: "F3", Required: "必填", Type: "int"},
		},
	}

	errors, lineCount, _ := checker.checkSingleFileContent(path, sheetConfig)
	if lineCount != 1 {
		t.Errorf("行数应为1, 实际为%d", lineCount)
	}
	// field3的值"field3"不是int，应该有类型错误
	if len(errors) == 0 {
		t.Error("field3为非int值，应报告类型错误")
	}
}

func TestCheckSingleFileContent_RequiredFieldEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	content := "|field2|field3\n"
	path := createTestFile(t, tmpDir, "test.txt", content)

	checker := newTestXDRChecker("|")

	sheetConfig := parser.SheetConfig{
		SheetName:      "test",
		FieldNumberMap: map[string]int{"1": 0, "2": 1, "3": 2},
		FieldRules: []parser.FieldRule{
			{FieldName: "F1", Required: "必填"},
			{FieldName: "F2", Required: "选填"},
			{FieldName: "F3", Required: "必填"},
		},
	}

	errors, _, _ := checker.checkSingleFileContent(path, sheetConfig)
	hasRequiredError := false
	for _, e := range errors {
		if e.ErrorType == "required" {
			hasRequiredError = true
		}
	}
	if !hasRequiredError {
		t.Error("必填字段为空时应报告required错误")
	}
}

func TestCheckSingleFileContent_SkipEmptyField(t *testing.T) {
	tmpDir := t.TempDir()
	content := "v1|v2|v3\n"
	path := createTestFile(t, tmpDir, "test.txt", content)

	checker := newTestXDRChecker("|")

	sheetConfig := parser.SheetConfig{
		SheetName:      "test",
		FieldNumberMap: map[string]int{"1": 0, "2": 1, "3": 2},
		FieldRules: []parser.FieldRule{
			{FieldName: "F1", Required: "空"}, // 空 = 跳过检查
			{FieldName: "F2", Required: "选填"},
			{FieldName: "F3", Required: "必填"},
		},
	}

	errors, _, _ := checker.checkSingleFileContent(path, sheetConfig)
	for _, e := range errors {
		if e.FieldName == "F1" {
			t.Error("Required为空的字段不应报错")
		}
	}
}

func TestCheckSingleFileContent_SkipHeaderLine(t *testing.T) {
	tmpDir := t.TempDir()
	// HeaderCheck = "不校验" 跳过首行
	content := "header1|header2|header3\n1|2|3\n"
	path := createTestFile(t, tmpDir, "test.txt", content)

	checker := newTestXDRChecker("|")

	sheetConfig := parser.SheetConfig{
		SheetName: "test",
		FileValidation: parser.FileValidationConfig{
			HeaderCheck: "不校验",
		},
		FieldNumberMap: map[string]int{"1": 0, "2": 1, "3": 2},
		FieldRules: []parser.FieldRule{
			{FieldName: "F1", Required: "必填", Type: "int"},
			{FieldName: "F2", Required: "必填", Type: "int"},
			{FieldName: "F3", Required: "必填", Type: "int"},
		},
	}

	errors, lineCount, _ := checker.checkSingleFileContent(path, sheetConfig)
	if lineCount != 2 {
		t.Errorf("行数应为2(首行跳过校验但仍计行), 实际为%d", lineCount)
	}
	// 首行是"header1|header2|header3"，虽然HeaderCheck="不校验"跳过了字段校验，
	// 但第二行的"1|2|3"都是有效int，不应有类型错误
	for _, e := range errors {
		if e.ErrorType == "type" {
			t.Errorf("有效int值不应报类型错误: %s", e.Message)
		}
	}
}

func TestCheckSingleFileContent_FieldCountMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	// 只有2个字段，但规则期望3个
	content := "v1|v2\n"
	path := createTestFile(t, tmpDir, "test.txt", content)

	checker := newTestXDRChecker("|")

	sheetConfig := parser.SheetConfig{
		SheetName: "test",
		FileValidation: parser.FileValidationConfig{
			FieldCount: "校验",
		},
		FieldRules: []parser.FieldRule{
			{FieldName: "F1", Required: "必填"},
			{FieldName: "F2", Required: "必填"},
			{FieldName: "F3", Required: "必填"},
		},
	}

	errors, _, _ := checker.checkSingleFileContent(path, sheetConfig)
	hasFieldCountError := false
	for _, e := range errors {
		if e.ErrorType == "field_count" {
			hasFieldCountError = true
		}
	}
	if !hasFieldCountError {
		t.Error("字段数量不匹配时应报field_count错误")
	}
}

func TestCheckSingleFileContent_EmptyLines(t *testing.T) {
	tmpDir := t.TempDir()
	content := "1|2|3\n\n\n4|5|6\n"
	path := createTestFile(t, tmpDir, "test.txt", content)

	checker := newTestXDRChecker("|")

	sheetConfig := parser.SheetConfig{
		SheetName:      "test",
		FieldNumberMap: map[string]int{"1": 0, "2": 1, "3": 2},
		FieldRules: []parser.FieldRule{
			{FieldName: "F1", Required: "必填", Type: "int"},
			{FieldName: "F2", Required: "选填", Type: "int"},
			{FieldName: "F3", Required: "必填", Type: "int"},
		},
	}

	errors, lineCount, _ := checker.checkSingleFileContent(path, sheetConfig)
	if lineCount != 2 {
		t.Errorf("空行跳过后行数应为2, 实际为%d", lineCount)
	}
	if len(errors) > 0 {
		t.Errorf("有效int数据不应有错误, 但有%d个", len(errors))
	}
}

func TestCheckSingleFileContent_FileNotFound(t *testing.T) {
	checker := newTestXDRChecker("|")

	sheetConfig := parser.SheetConfig{
		SheetName:  "test",
		FieldRules: []parser.FieldRule{},
	}

	errors, lineCount, _ := checker.checkSingleFileContent("/nonexistent/file.txt", sheetConfig)
	if lineCount != 0 {
		t.Errorf("文件不存在时行数应为0, 实际为%d", lineCount)
	}
	if len(errors) == 0 {
		t.Error("文件不存在时应报错")
	}
	if errors[0].ErrorType != "system" {
		t.Errorf("错误类型应为system, 实际为%s", errors[0].ErrorType)
	}
}

func TestCheckSingleFileContent_ConditionValidation(t *testing.T) {
	tmpDir := t.TempDir()
	// 条件: if($2==1) -> 字段1变必填; $2=1, 字段0为空
	content := "|1|v3\n"
	path := createTestFile(t, tmpDir, "test.txt", content)

	checker := newTestXDRChecker("|")

	sheetConfig := parser.SheetConfig{
		SheetName:      "test",
		FieldNumberMap: map[string]int{"1": 0, "2": 1, "3": 2},
		FieldRules: []parser.FieldRule{
			{FieldName: "F1", Required: "选填", Condition: "if($2==1)",
				ParsedCondition: &parser.ParsedCondition{
					FieldIndex:    1,
					ExpectedExact: map[string]struct{}{"1": {}},
					IsEqual:       true,
				}},
			{FieldName: "F2", Required: "必填"},
			{FieldName: "F3", Required: "必填"},
		},
	}

	errors, _, _ := checker.checkSingleFileContent(path, sheetConfig)
	hasRequiredError := false
	for _, e := range errors {
		if e.ErrorType == "required" && e.FieldName == "F1" {
			hasRequiredError = true
		}
	}
	if !hasRequiredError {
		t.Error("条件满足时选填变为必填，空值应报required错误")
	}
}

func TestCheckSingleFileContent_ConditionNotSatisfied(t *testing.T) {
	tmpDir := t.TempDir()
	// 条件: if($2==1) -> 字段0变必填; $2=2, 条件不满足，字段0选填
	content := "|2|v3\n"
	path := createTestFile(t, tmpDir, "test.txt", content)

	checker := newTestXDRChecker("|")

	sheetConfig := parser.SheetConfig{
		SheetName:      "test",
		FieldNumberMap: map[string]int{"1": 0, "2": 1, "3": 2},
		FieldRules: []parser.FieldRule{
			{FieldName: "F1", Required: "选填", Condition: "if($2==1)",
				ParsedCondition: &parser.ParsedCondition{
					FieldIndex:    1,
					ExpectedExact: map[string]struct{}{"1": {}},
					IsEqual:       true,
				}},
			{FieldName: "F2", Required: "必填"},
			{FieldName: "F3", Required: "必填"},
		},
	}

	errors, _, _ := checker.checkSingleFileContent(path, sheetConfig)
	for _, e := range errors {
		if e.FieldName == "F1" && e.ErrorType == "required" {
			t.Error("条件不满足时选填字段为空不应报required错误")
		}
	}
}

func TestCheckSingleFileContent_EnumValidation(t *testing.T) {
	tmpDir := t.TempDir()
	content := "5|abc\n"
	path := createTestFile(t, tmpDir, "test.txt", content)

	checker := newTestXDRChecker("|")

	sheetConfig := parser.SheetConfig{
		SheetName:      "test",
		FieldNumberMap: map[string]int{"1": 0, "2": 1},
		FieldRules: []parser.FieldRule{
			{
				FieldName: "F1", Required: "必填",
				Rules: []string{"[1-10]"},
				ParsedEnums: map[string]*parser.ParsedEnumValue{
					"[1-10]": {
						Ranges:  []parser.EnumRange{{Min: 1, Max: 10}},
						RawRule: "1-10",
					},
				},
			},
			{FieldName: "F2", Required: "必填"},
		},
	}

	errors, _, _ := checker.checkSingleFileContent(path, sheetConfig)
	for _, e := range errors {
		if e.FieldName == "F1" {
			t.Errorf("5在[1-10]范围内，不应报错: %s", e.Message)
		}
	}
}

func TestCheckSingleFileContent_IPValidation(t *testing.T) {
	tmpDir := t.TempDir()
	content := "192.168.1.1|invalid_ip\n"
	path := createTestFile(t, tmpDir, "test.txt", content)

	checker := newTestXDRChecker("|")

	sheetConfig := parser.SheetConfig{
		SheetName:      "test",
		FieldNumberMap: map[string]int{"1": 0, "2": 1},
		FieldRules: []parser.FieldRule{
			{FieldName: "F1", Required: "必填", Type: "ipv4"},
			{FieldName: "F2", Required: "必填", Type: "ipv4"},
		},
	}

	errors, _, _ := checker.checkSingleFileContent(path, sheetConfig)
	validIPHasError := false
	invalidIPHasError := false
	for _, e := range errors {
		if e.FieldName == "F1" && e.ErrorType == "type" {
			validIPHasError = true
		}
		if e.FieldName == "F2" && e.ErrorType == "type" {
			invalidIPHasError = true
		}
	}
	if validIPHasError {
		t.Error("有效IPv4不应报类型错误")
	}
	if !invalidIPHasError {
		t.Error("无效IPv4应报类型错误")
	}
}

func TestCheckSingleFileContent_OptionalEmptyField(t *testing.T) {
	tmpDir := t.TempDir()
	content := "|v2\n"
	path := createTestFile(t, tmpDir, "test.txt", content)

	checker := newTestXDRChecker("|")

	sheetConfig := parser.SheetConfig{
		SheetName:      "test",
		FieldNumberMap: map[string]int{"1": 0, "2": 1},
		FieldRules: []parser.FieldRule{
			{FieldName: "F1", Required: "选填", Type: "int"}, // 选填且为空，跳过类型校验
			{FieldName: "F2", Required: "必填"},
		},
	}

	errors, _, _ := checker.checkSingleFileContent(path, sheetConfig)
	for _, e := range errors {
		if e.FieldName == "F1" {
			t.Errorf("选填字段为空时不应报错: %s", e.Message)
		}
	}
}

// ============== processTasksWithWorkerPool 测试 ==============

func TestProcessTasksWithWorkerPool_EmptyTasks(t *testing.T) {
	tmpDir := t.TempDir()
	resultFile, err := os.Create(filepath.Join(tmpDir, "result.txt"))
	if err != nil {
		t.Fatalf("创建结果文件失败: %v", err)
	}

	checker := &XDRChecker{
		Config:       &config.Config{ColDelimiter: "|"},
		ResultFile:   resultFile,
		WorkerNum:    2,
		ReportFormat: "txt",
	}

	err = checker.processTasksWithWorkerPool([]CheckTask{}, 2)
	if err != nil {
		t.Errorf("空任务列表不应返回错误: %v", err)
	}
}

func TestProcessTasksWithWorkerPool_SingleTask(t *testing.T) {
	tmpDir := t.TempDir()
	content := "1|2|3\n"
	path := createTestFile(t, tmpDir, "test.txt", content)

	resultFile, err := os.Create(filepath.Join(tmpDir, "result.txt"))
	if err != nil {
		t.Fatalf("创建结果文件失败: %v", err)
	}

	checker := &XDRChecker{
		Config:       &config.Config{ColDelimiter: "|"},
		ResultFile:   resultFile,
		WorkerNum:    1,
		ReportFormat: "txt",
	}

	sheetConfig := parser.SheetConfig{
		SheetName: "test",
		FieldRules: []parser.FieldRule{
			{FieldName: "F1", Required: "必填", Type: "int"},
			{FieldName: "F2", Required: "必填", Type: "int"},
			{FieldName: "F3", Required: "必填", Type: "int"},
		},
	}

	tasks := []CheckTask{
		{Filename: path, PathName: "test", SheetConfig: sheetConfig},
	}

	err = checker.processTasksWithWorkerPool(tasks, 1)
	if err != nil {
		t.Errorf("处理任务不应返回错误: %v", err)
	}
}

// ============== worker 测试 ==============

func TestWorker_NormalTask(t *testing.T) {
	tmpDir := t.TempDir()
	content := "1|2|3\n"
	path := createTestFile(t, tmpDir, "test.txt", content)

	checker := newTestXDRChecker("|")

	sheetConfig := parser.SheetConfig{
		SheetName:      "test",
		FieldNumberMap: map[string]int{"1": 0, "2": 1, "3": 2},
		FieldRules: []parser.FieldRule{
			{FieldName: "F1", Required: "必填", Type: "int"},
			{FieldName: "F2", Required: "必填", Type: "int"},
			{FieldName: "F3", Required: "必填", Type: "int"},
		},
	}

	taskChan := make(chan CheckTask, 1)
	resultChan := make(chan CheckResult, 1)
	var wg sync.WaitGroup
	wg.Add(1)

	taskChan <- CheckTask{
		Filename:    path,
		PathName:    "test",
		SheetConfig: sheetConfig,
		IsSpecial:   false,
	}
	close(taskChan)

	go checker.worker(0, taskChan, resultChan, &wg)
	wg.Wait()
	close(resultChan)

	result := <-resultChan
	if result.LineCount != 1 {
		t.Errorf("行数应为1, 实际为%d", result.LineCount)
	}
	if len(result.Errors) > 0 {
		t.Errorf("有效int数据不应有错误, 有%d个", len(result.Errors))
	}
}

func TestWorker_SpecialTask(t *testing.T) {
	checker := newTestXDRChecker("|")

	sheetConfig := parser.SheetConfig{
		SheetName:  "test",
		FieldRules: []parser.FieldRule{},
	}

	taskChan := make(chan CheckTask, 1)
	resultChan := make(chan CheckResult, 1)
	var wg sync.WaitGroup
	wg.Add(1)

	taskChan <- CheckTask{
		Filename:    "/nonexistent/file.dat",
		PathName:    "local_to_cu_0x01e0",
		SheetConfig: sheetConfig,
		IsSpecial:   true,
	}
	close(taskChan)

	go checker.worker(0, taskChan, resultChan, &wg)
	wg.Wait()
	close(resultChan)

	result := <-resultChan
	if result.Success {
		t.Error("特殊路径处理不存在的文件应失败")
	}
}

// ============== ClearOldTmpDirs 测试 ==============

func TestClearOldTmpDirs(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建旧目录
	oldDir := filepath.Join(tmpDir, "20260101")
	if err := os.MkdirAll(oldDir, 0755); err != nil {
		t.Fatalf("创建旧目录失败: %v", err)
	}
	// 修改修改时间为31天前
	oldTime := time.Now().AddDate(0, 0, -31)
	if err := os.Chtimes(oldDir, oldTime, oldTime); err != nil {
		t.Fatalf("修改目录时间失败: %v", err)
	}

	// 创建新目录
	newDir := filepath.Join(tmpDir, "20260417")
	if err := os.MkdirAll(newDir, 0755); err != nil {
		t.Fatalf("创建新目录失败: %v", err)
	}

	err := ClearOldTmpDirs(tmpDir, 30)
	if err != nil {
		t.Fatalf("ClearOldTmpDirs返回错误: %v", err)
	}

	// 旧目录应被删除
	if _, err := os.Stat(oldDir); !os.IsNotExist(err) {
		t.Error("旧目录应被删除")
	}

	// 新目录应保留
	if _, err := os.Stat(newDir); os.IsNotExist(err) {
		t.Error("新目录应保留")
	}
}

func TestClearOldTmpDirs_NonExistentBase(t *testing.T) {
	err := ClearOldTmpDirs("/nonexistent_base_dir_12345", 30)
	if err == nil {
		t.Error("不存在的基目录应返回错误")
	}
}

// ============== generateResultSummary 测试 ==============

func TestGenerateResultSummary_WithErrors(t *testing.T) {
	tmpDir := t.TempDir()
	resultFile, _ := os.Create(filepath.Join(tmpDir, "result.txt"))

	checker := &XDRChecker{
		ResultFile: resultFile,
	}

	// 不 panic 即为通过
	checker.generateResultSummary("path1", "/some/path", 5, []string{"error1"})
}

func TestGenerateResultSummary_NoErrors(t *testing.T) {
	tmpDir := t.TempDir()
	resultFile, _ := os.Create(filepath.Join(tmpDir, "result.txt"))

	checker := &XDRChecker{
		ResultFile: resultFile,
	}

	checker.generateResultSummary("path1", "/some/path", 5, []string{})
}

// ============== writeResult 测试 ==============

func TestWriteResult(t *testing.T) {
	tmpDir := t.TempDir()
	resultFile, _ := os.Create(filepath.Join(tmpDir, "result.txt"))

	checker := &XDRChecker{
		ResultFile: resultFile,
	}

	checker.writeResult("测试消息")

	// 验证文件中有内容写入
	resultFile.Seek(0, 0)
	data, _ := io.ReadAll(resultFile)
	if !strings.Contains(string(data), "测试消息") {
		t.Error("writeResult应写入消息到结果文件")
	}
}

func TestWriteResult_NilFile(t *testing.T) {
	checker := &XDRChecker{
		ResultFile: nil,
	}

	// 不应panic
	checker.writeResult("测试消息")
}

// ============== fileWriter 测试 ==============

func TestFileWriter_WithErrors(t *testing.T) {
	tmpDir := t.TempDir()
	resultFile, _ := os.Create(filepath.Join(tmpDir, "result.txt"))

	checker := &XDRChecker{
		Config:       &config.Config{ColDelimiter: "|"},
		ResultFile:   resultFile,
		WorkerNum:    2,
		ReportFormat: "txt",
	}

	sheetConfig := parser.SheetConfig{
		SheetName:      "test_path",
		FieldNumberMap: map[string]int{"1": 0, "2": 1},
		FieldRules: []parser.FieldRule{
			{FieldName: "F1", Required: "必填"},
			{FieldName: "F2", Required: "必填"},
		},
	}

	resultChan := make(chan CheckResult, 2)
	var wg sync.WaitGroup
	wg.Add(1)

	// 发送有错误的结果
	resultChan <- CheckResult{
		Task: CheckTask{
			Filename:    "test.txt",
			PathName:    "test_path",
			SheetConfig: sheetConfig,
		},
		Errors: []ValidationError{
			{
				Filename:   "test.txt",
				LineNum:    1,
				FieldIndex: 1,
				FieldName:  "F1",
				ErrorType:  "type",
				RuleOrType: "int",
				Message:    "不是有效的整数",
				FieldValue: "abc",
				FullLine:   "abc|def",
			},
		},
		LineCount: 1,
		Success:   false,
		ErrorMsg:  "文件test.txt检查发现1个错误",
	}

	// 发送无错误的结果
	resultChan <- CheckResult{
		Task: CheckTask{
			Filename:    "test2.txt",
			PathName:    "test_path",
			SheetConfig: sheetConfig,
		},
		Errors:    []ValidationError{},
		LineCount: 5,
		Success:   true,
	}

	close(resultChan)
	go checker.fileWriter(resultChan, &wg)
	wg.Wait()
}

func TestFileWriter_NoErrors(t *testing.T) {
	tmpDir := t.TempDir()
	resultFile, _ := os.Create(filepath.Join(tmpDir, "result.txt"))

	checker := &XDRChecker{
		Config:       &config.Config{ColDelimiter: "|"},
		ResultFile:   resultFile,
		WorkerNum:    2,
		ReportFormat: "txt",
	}

	sheetConfig := parser.SheetConfig{
		SheetName:  "test_path",
		FieldRules: []parser.FieldRule{},
	}

	resultChan := make(chan CheckResult, 1)
	var wg sync.WaitGroup
	wg.Add(1)

	resultChan <- CheckResult{
		Task: CheckTask{
			Filename:    "test.txt",
			PathName:    "test_path",
			SheetConfig: sheetConfig,
		},
		Errors:    []ValidationError{},
		LineCount: 10,
		Success:   true,
	}

	close(resultChan)
	go checker.fileWriter(resultChan, &wg)
	wg.Wait()
}

// ============== createResultDirectory 测试 ==============

func TestCreateResultDirectory(t *testing.T) {
	tmpBase := filepath.Join(t.TempDir(), "xdr_check_test")
	// 使用固定日期
	os.MkdirAll(tmpBase, 0755)

	checker := &XDRChecker{
		Config:       &config.Config{ColDelimiter: "|"},
		WorkerNum:    2,
		ReportFormat: "txt",
	}

	// 通过覆盖时间来测试，但由于 createResultDirectory 使用 time.Now()，
	// 我们只验证它能创建目录
	resultDir := checker.createResultDirectory()
	if resultDir != "" {
		// 验证目录存在
		if _, err := os.Stat(resultDir); os.IsNotExist(err) {
			t.Error("结果目录应该已创建")
		}
	}
}

// ============== 表格格式输出测试 ==============

func TestWriteFormattedErrors_TxtFormat(t *testing.T) {
	tmpDir := t.TempDir()
	resultFile, _ := os.Create(filepath.Join(tmpDir, "result.txt"))

	checker := &XDRChecker{
		ReportFormat: "txt",
		ResultFile:   resultFile,
	}

	writer := bufio.NewWriter(resultFile)
	errors := []ValidationError{
		{
			Filename:   "test.txt",
			LineNum:    1,
			FieldIndex: 1,
			FieldName:  "F1",
			ErrorType:  "type",
			RuleOrType: "int",
			Message:    "不是有效的整数",
			FieldValue: "abc",
			FullLine:   "abc|def",
		},
	}

	checker.writeFormattedErrors(writer, "test.txt", errors)
	writer.Flush()
}

func TestWriteFormattedErrors_TableFormat(t *testing.T) {
	tmpDir := t.TempDir()
	resultFile, _ := os.Create(filepath.Join(tmpDir, "result.txt"))

	checker := &XDRChecker{
		ReportFormat: "table",
		ResultFile:   resultFile,
	}

	writer := bufio.NewWriter(resultFile)
	errors := []ValidationError{
		{
			Filename:   "test.txt",
			LineNum:    1,
			FieldIndex: 1,
			FieldName:  "F1",
			ErrorType:  "type",
			RuleOrType: "int",
			Message:    "不是有效的整数",
			FieldValue: "abc",
			FullLine:   "abc|def",
		},
	}

	checker.writeFormattedErrors(writer, "test.txt", errors)
	writer.Flush()
}

func TestWriteFormattedErrors_HtmlFormat(t *testing.T) {
	tmpDir := t.TempDir()
	resultFile, _ := os.Create(filepath.Join(tmpDir, "result.html"))

	checker := &XDRChecker{
		ReportFormat: "html",
		ResultFile:   resultFile,
	}

	writer := bufio.NewWriter(resultFile)
	errors := []ValidationError{
		{
			Filename:   "test.txt",
			LineNum:    1,
			FieldIndex: 1,
			FieldName:  "F1",
			ErrorType:  "type",
			RuleOrType: "int",
			Message:    "不是有效的整数",
			FieldValue: "abc",
			FullLine:   "abc|def",
		},
	}

	checker.writeFormattedErrors(writer, "test.txt", errors)
	writer.Flush()
}

// ============== groupErrorsByLine / groupErrorsByFieldStruct 测试 ==============

func TestGroupErrorsByLine(t *testing.T) {
	checker := newTestXDRChecker("|")
	errors := []ValidationError{
		{LineNum: 1, FieldName: "F1", ErrorType: "type", Message: "错误1"},
		{LineNum: 1, FieldName: "F2", ErrorType: "rule", Message: "错误2"},
		{LineNum: 2, FieldName: "F1", ErrorType: "type", Message: "错误3"},
	}

	groups := checker.groupErrorsByLine(errors)
	if len(groups[1]) != 2 {
		t.Errorf("第1行应有2个错误, 实际为%d", len(groups[1]))
	}
	if len(groups[2]) != 1 {
		t.Errorf("第2行应有1个错误, 实际为%d", len(groups[2]))
	}
}

func TestGroupErrorsByFieldStruct(t *testing.T) {
	checker := newTestXDRChecker("|")
	errors := []ValidationError{
		{FieldName: "F1", ErrorType: "type", RuleOrType: "int", FieldValue: "abc"},
		{FieldName: "F1", ErrorType: "rule", RuleOrType: "len=5", FieldValue: "abc", Message: "长度应为5"},
		{FieldName: "F2", ErrorType: "type", RuleOrType: "ip", FieldValue: "invalid"},
	}

	groups := checker.groupErrorsByFieldStruct(errors)
	if _, ok := groups["F1"]; !ok {
		t.Error("应有F1的错误分组")
	}
	if _, ok := groups["F2"]; !ok {
		t.Error("应有F2的错误分组")
	}
}

// ============== 基准测试 ==============

func BenchmarkCheckSingleFileContent(b *testing.B) {
	tmpDir := b.TempDir()
	// 1000行 x 5字段
	var lines []string
	for i := 0; i < 1000; i++ {
		lines = append(lines, "123|192.168.1.1|hello|5|2023-01-01 12:00:00")
	}
	content := strings.Join(lines, "\n")
	path := filepath.Join(tmpDir, "bench.txt")
	os.WriteFile(path, []byte(content), 0644)

	checker := &XDRChecker{
		Config:       &config.Config{ColDelimiter: "|"},
		WorkerNum:    4,
		ReportFormat: "txt",
	}

	sheetConfig := parser.SheetConfig{
		SheetName:      "bench",
		FieldNumberMap: map[string]int{"1": 0, "2": 1, "3": 2, "4": 3, "5": 4},
		FieldRules: []parser.FieldRule{
			{FieldName: "F1", Required: "必填", Type: "int", Rules: []string{"[1-999]"},
				ParsedEnums: map[string]*parser.ParsedEnumValue{
					"[1-999]": {Ranges: []parser.EnumRange{{Min: 1, Max: 999}}, RawRule: "1-999"},
				},
			},
			{FieldName: "F2", Required: "必填", Type: "ipv4"},
			{FieldName: "F3", Required: "必填", Rules: []string{"len>=1"}},
			{FieldName: "F4", Required: "必填", Type: "int", Rules: []string{"[1-100]"},
				ParsedEnums: map[string]*parser.ParsedEnumValue{
					"[1-100]": {Ranges: []parser.EnumRange{{Min: 1, Max: 100}}, RawRule: "1-100"},
				},
			},
			{FieldName: "F5", Required: "必填", Type: "datetime"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker.checkSingleFileContent(path, sheetConfig)
	}
}
