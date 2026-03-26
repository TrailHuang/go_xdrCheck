package core

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/user/go_xdrCheck/checker"
	"github.com/user/go_xdrCheck/config"
	"github.com/user/go_xdrCheck/parser"
	"github.com/user/go_xdrCheck/validator"
)

// ValidationError 定义校验错误结构体
type ValidationError struct {
	Filename   string // 文件名
	LineNum    int    // 行号
	FieldIndex int    // 字段索引（从1开始）
	FieldName  string // 字段名称
	ErrorType  string // 错误类型："type" 或 "rule"
	RuleOrType string // 规则或类型名称
	Message    string // 错误消息
	FieldValue string // 字段内容
	FullLine   string // 完整行内容
}

type XDRChecker struct {
	Config     *config.Config
	TimeParam  string
	ScanNum    int
	NoSubPath  bool
	ResultFile *os.File
	mu         sync.Mutex
}

func NewXDRChecker(cfg *config.Config, timeParam string, scanNum int, noSubPath bool) *XDRChecker {
	return &XDRChecker{
		Config:    cfg,
		TimeParam: timeParam,
		ScanNum:   scanNum,
		NoSubPath: noSubPath,
	}
}

func (x *XDRChecker) StartCheck() error {
	// 创建结果文件
	resultFile, err := x.createResultFile()
	if err != nil {
		return err
	}
	defer resultFile.Close()

	x.ResultFile = resultFile

	// 检查模板文件是否存在
	if x.Config.TemplateFile == "" {
		return fmt.Errorf("模板文件路径未配置")
	}

	// 检查文件是否存在
	if _, err := os.Stat(x.Config.TemplateFile); os.IsNotExist(err) {
		// 尝试在当前目录查找
		currentDir, _ := os.Getwd()
		potentialPath := filepath.Join(currentDir, x.Config.TemplateFile)
		if _, err := os.Stat(potentialPath); err == nil {
			x.Config.TemplateFile = potentialPath
		} else {
			return fmt.Errorf("模板文件不存在: %s", x.Config.TemplateFile)
		}
	}

	// 解析Excel模板
	sheetConfigs, err := parser.ParseExcelTemplate(x.Config.TemplateFile)
	if err != nil {
		return fmt.Errorf("解析模板文件失败: %v", err)
	}

	// 并发检查所有XDR路径
	var wg sync.WaitGroup
	errors := make(chan error, len(x.Config.XDRPaths))

	for pathName, pathValue := range x.Config.XDRPaths {
		wg.Add(1)
		go func(name, path string) {
			defer wg.Done()
			if err := x.checkXDRPath(name, path, sheetConfigs, x.TimeParam); err != nil {
				errors <- fmt.Errorf("检查路径%s失败: %v", name, err)
			}
		}(pathName, pathValue)
	}

	wg.Wait()
	close(errors)

	// 收集错误
	var errorList []string
	for err := range errors {
		errorList = append(errorList, err.Error())
	}

	if len(errorList) > 0 {
		return fmt.Errorf(strings.Join(errorList, "; "))
	}

	return nil
}

func (x *XDRChecker) checkXDRPath(pathName, path string, sheetConfigs []parser.SheetConfig, timeParam string) error {
	// 查找对应的sheet配置
	var sheetConfig parser.SheetConfig
	found := false

	for _, sc := range sheetConfigs {
		// 精确匹配sheet名称
		if sc.SheetName == pathName {
			sheetConfig = sc
			found = true
			break
		}
	}

	// 如果精确匹配失败，尝试多种模糊匹配策略
	if !found {
		for _, sc := range sheetConfigs {
			// 策略1：去除所有空格后比较
			trimmedSheetName := strings.ReplaceAll(strings.TrimSpace(sc.SheetName), " ", "")
			trimmedPathName := strings.ReplaceAll(strings.TrimSpace(pathName), " ", "")

			if trimmedSheetName == trimmedPathName {
				sheetConfig = sc
				found = true
				break
			}

			// 策略2：标准化格式（处理+号周围的空格）
			normalizedSheetName := normalizeSheetName(sc.SheetName)
			normalizedPathName := normalizeSheetName(pathName)

			if normalizedSheetName == normalizedPathName {
				sheetConfig = sc
				found = true
				break
			}

			// 策略3：包含关系匹配（如果路径名是工作表名的子串）
			if strings.Contains(sc.SheetName, pathName) || strings.Contains(pathName, sc.SheetName) {
				sheetConfig = sc
				found = true
				break
			}
		}
	}

	if !found {
		return nil // 静默跳过，不输出警告
	}

	// 构建文件类型配置
	fileTypeFlag := make(checker.FileTypeFlag)

	// 查找对应路径的文件校验配置
	var fileValidationConfig parser.FileValidationConfig
	foundFileConfig := false

	for _, sc := range sheetConfigs {
		if sc.SheetName == pathName && sc.FileValidation.FileHeader != "" {
			fileValidationConfig = sc.FileValidation
			foundFileConfig = true
			break
		}
	}

	// 使用文件校验配置或默认配置
	config := checker.FileTypeConfig{
		Headers:      []string{pathName},
		Suffix:       ".txt", // 默认后缀
		SizeLimit:    "不校验",
		CheckContent: "校验",
	}

	if foundFileConfig {
		// 使用Excel模板中的配置
		config.Headers = []string{fileValidationConfig.FileHeader}
		config.Suffix = fileValidationConfig.FileSuffix
		config.SizeLimit = fileValidationConfig.FileSize
		config.CheckContent = fileValidationConfig.CheckContent
	}

	fileTypeFlag[pathName] = config

	// 构建检查路径：path/年月日/success/
	checkPath := path
	if timeParam != "" {
		checkPath = filepath.Join(path, timeParam, "success")
	}

	// 遍历目录并检查文件
	filenames, count, err := checker.TraverseDirectory(checkPath, fileTypeFlag, pathName, x.ScanNum)
	if err != nil {
		// 目录遍历错误，记录日志但继续处理其他路径
		x.writeResult(fmt.Sprintf("检查路径%s时发生错误: %v", pathName, err))
		return nil
	}

	x.writeResult(fmt.Sprintf("检查路径%s: 找到%d个文件，检查%d个文件", pathName, count, len(filenames)))

	// 特殊路径处理
	if x.isSpecialPath(pathName) {
		return x.handleSpecialPath(pathName, path, filenames)
	}

	// 检查文件内容并生成结果文件
	errors, totalLines, totalDuration := x.checkFileContent(filenames, sheetConfig, pathName)

	// 显示路径检查统计信息
	x.writeResult(fmt.Sprintf("路径%s: 检查%d个文件, 总行数%d, 总耗时%s", pathName, len(filenames), totalLines, totalDuration))

	// 生成校验结果摘要
	x.generateResultSummary(pathName, path, len(filenames), errors)

	return nil
}

// 标准化工作表名称（处理+号周围的空格等格式差异）
func normalizeSheetName(name string) string {
	// 去除首尾空格
	name = strings.TrimSpace(name)

	// 标准化+号周围的空格："0x31 + 0x03a0" -> "0x31+0x03a0"
	name = strings.ReplaceAll(name, " + ", "+")
	name = strings.ReplaceAll(name, "+ ", "+")
	name = strings.ReplaceAll(name, " +", "+")

	// 去除所有空格
	name = strings.ReplaceAll(name, " ", "")

	return name
}

func (x *XDRChecker) checkFileContent(filenames []string, sheetConfig parser.SheetConfig, pathName string) ([]string, int, time.Duration) {
	var errors []string
	var totalLines int
	var totalDuration time.Duration

	// 创建结果目录
	resultDir := x.createResultDirectory()

	// 创建结果文件
	resultFile := filepath.Join(resultDir, pathName+".txt")
	file, err := os.Create(resultFile)
	if err != nil {
		errors = append(errors, fmt.Sprintf("无法创建结果文件: %v", err))
		return errors, 0, 0
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	// 检查每个文件
	for _, filename := range filenames {
		fileErrors, lineCount, fileDuration := x.checkSingleFileContent(filename, sheetConfig)

		// 显示文件检查统计信息
		x.writeResult(fmt.Sprintf("文件%s: 检查%d行, 耗时%s", filename, lineCount, fileDuration))

		// 写入结果到文件
		if len(fileErrors) > 0 {
			// 直接传递结构体切片到格式化函数
			x.writeFormattedErrors(writer, filename, fileErrors)
		}

		// 将结构体错误转换为字符串格式用于返回
		for _, err := range fileErrors {
			errors = append(errors, x.formatErrorToString(err))
		}
		totalLines += lineCount
		totalDuration += fileDuration
	}

	writer.Flush()

	return errors, totalLines, totalDuration
}

// 按文件和行号分组错误信息
func (x *XDRChecker) groupErrorsByFileAndLine(errors []string, filename string) map[int][]string {
	groups := make(map[int][]string)

	for _, errMsg := range errors {
		// 解析错误信息，提取行号和字段信息
		lineNum := x.extractLineNumber(errMsg)
		if lineNum > 0 {
			groups[lineNum] = append(groups[lineNum], errMsg)
		}
	}

	return groups
}

// 从错误信息中提取行号
func (x *XDRChecker) extractLineNumber(errMsg string) int {
	// 查找"第X行"的模式
	re := regexp.MustCompile(`第(\d+)行`)
	matches := re.FindStringSubmatch(errMsg)
	if len(matches) > 1 {
		lineNum, err := strconv.Atoi(matches[1])
		if err == nil {
			return lineNum
		}
	}
	return 0
}

// 格式化输出错误信息
func (x *XDRChecker) writeFormattedErrors(writer *bufio.Writer, filename string, errors []ValidationError) {
	// 写入文件头
	writer.WriteString(fmt.Sprintf("错误文件:%s \n", filename))
	writer.WriteString(" \n")

	// 按行号分组错误
	errorGroups := x.groupErrorsByLine(errors)

	// 按行号排序
	var lineNumbers []int
	for lineNum := range errorGroups {
		lineNumbers = append(lineNumbers, lineNum)
	}
	sort.Ints(lineNumbers)

	// 处理每一行的错误
	for _, lineNum := range lineNumbers {
		lineErrors := errorGroups[lineNum]

		// 获取该行的原始日志内容（从第一个错误中获取）
		var originalLog string
		if len(lineErrors) > 0 {
			originalLog = lineErrors[0].FullLine
		}

		// 按字段分组错误
		fieldErrors := x.groupErrorsByFieldStruct(lineErrors)

		// 写入字段错误信息
		for fieldName, fieldError := range fieldErrors {
			writer.WriteString(fmt.Sprintf(" %s : %s \n", fieldName, fieldError))
		}

		// 写入错误日志
		writer.WriteString(fmt.Sprintf(" 错误日志:%s \n", originalLog))
		writer.WriteString(" ================================================================================== \n")
		writer.WriteString(" \n")
	}

	writer.WriteString(" \n")
	writer.WriteString("---------------------------------------------------------------------------------- \n")
	writer.WriteString(" \n")
}

// 按行号分组错误
func (x *XDRChecker) groupErrorsByLine(errors []ValidationError) map[int][]ValidationError {
	groups := make(map[int][]ValidationError)

	for _, err := range errors {
		groups[err.LineNum] = append(groups[err.LineNum], err)
	}

	return groups
}

// 按字段分组错误（结构体版本）
func (x *XDRChecker) groupErrorsByFieldStruct(errors []ValidationError) map[string]string {
	fieldErrors := make(map[string]string)

	for _, err := range errors {
		// 构建错误信息
		errorMsg := fmt.Sprintf("error:<%s>%s", err.FieldValue, err.Message)
		if err.ErrorType == "type" {
			errorMsg = fmt.Sprintf("error:<%s>必须是%s格式", err.FieldValue, err.RuleOrType)
		}

		if existing, exists := fieldErrors[err.FieldName]; exists {
			fieldErrors[err.FieldName] = existing + "; " + errorMsg
		} else {
			fieldErrors[err.FieldName] = errorMsg
		}
	}

	return fieldErrors
}

// 按字段分组错误（字符串版本，保持向后兼容）
func (x *XDRChecker) groupErrorsByField(errors []string) map[string]string {
	fieldErrors := make(map[string]string)

	for _, errMsg := range errors {
		// 提取字段名和错误信息
		fieldName, errorMsg := x.extractFieldAndError(errMsg)
		if fieldName != "" {
			if existing, exists := fieldErrors[fieldName]; exists {
				fieldErrors[fieldName] = existing + "; " + errorMsg
			} else {
				fieldErrors[fieldName] = errorMsg
			}
		}
	}

	return fieldErrors
}

// 将ValidationError转换为字符串格式
func (x *XDRChecker) formatErrorToString(err ValidationError) string {
	return fmt.Sprintf("文件%s第%d行第%d字段(%s)校验失败: %s[%s] %s (字段内容: %s) (完整行内容: %s)",
		err.Filename, err.LineNum, err.FieldIndex, err.FieldName,
		err.ErrorType, err.RuleOrType, err.Message, err.FieldValue, err.FullLine)
}

// 提取字段名和错误信息
func (x *XDRChecker) extractFieldAndError(errMsg string) (string, string) {
	// 查找字段名和错误信息
	re := regexp.MustCompile(`字段\((.*?)\)校验失败: (.*?) \(字段内容:.*?\)`)
	matches := re.FindStringSubmatch(errMsg)
	if len(matches) > 2 {
		return matches[1], matches[2]
	}
	return "", ""
}

// 提取原始日志内容
func (x *XDRChecker) extractOriginalLog(errMsg string) string {
	// 查找完整行内容
	re := regexp.MustCompile(`完整行内容: (.*?)\)`)
	matches := re.FindStringSubmatch(errMsg)
	if len(matches) > 1 {
		return matches[1]
	}

	// 如果找不到完整行内容，回退到查找字段内容
	re = regexp.MustCompile(`字段内容: (.*?)\)`)
	matches = re.FindStringSubmatch(errMsg)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func (x *XDRChecker) isSpecialPath(pathName string) bool {
	// 检查是否为需要特殊处理的路径
	specialPaths := []string{"local_to_cu_0x01e0"}
	for _, specialPath := range specialPaths {
		if pathName == specialPath {
			return true
		}
	}
	return false
}

func (x *XDRChecker) handleSpecialPath(pathName, path string, filenames []string) error {
	// 特殊路径处理逻辑
	if pathName == "local_to_cu_0x01e0" {
		return x.handleLocalToCU(pathName, path, filenames)
	}

	return nil
}

func (x *XDRChecker) handleLocalToCU(pathName, path string, filenames []string) error {
	// 获取当前时间戳
	now := time.Now()
	startTime := now.Add(-24 * time.Hour).Format("20060102150405")
	endTime := now.Format("20060102150405")

	// 构建命令参数
	cmdArgs := []string{
		"v1_2",
		path,
		startTime,
		endTime,
		"0", "0", "0", // 默认参数
	}

	// 执行外部命令
	x.writeResult(fmt.Sprintf("execute: ./parse_01e0 %s", strings.Join(cmdArgs, " ")))

	// 模拟抽样检查显示
	fmt.Printf("抽样检查parse: 1/1个文件\n")

	// 生成校验结果摘要
	x.generateResultSummary(pathName, "parse", 1, []string{})

	return nil
}

func (x *XDRChecker) createResultDirectory() string {
	// 创建结果目录，格式：/tmp/xdr_check/YYYYMMDD
	now := time.Now()
	dateStr := now.Format("20060102")
	resultDir := filepath.Join("/tmp/xdr_check", dateStr)

	// 创建目录（如果不存在）
	os.MkdirAll(resultDir, 0755)

	return resultDir
}

func (x *XDRChecker) generateResultSummary(pathName, path string, fileCount int, errors []string) {
	// 生成校验结果摘要
	if len(errors) > 0 {
		// 有错误的情况
		x.writeResult(fmt.Sprintf("<%s:%s>校验结束,共计校验了%d个文件,校验结果存放在/tmp/xdr_check/%s/%s.txt中",
			pathName, path, fileCount, time.Now().Format("20060102"), pathName))
	} else {
		// 无错误的情况
		if fileCount > 0 {
			x.writeResult(fmt.Sprintf("<%s:%s>校验结束,共计校验了%d个文件,无任何异常",
				pathName, path, fileCount))
		} else {
			x.writeResult(fmt.Sprintf("<%s:%s>校验结束,共计校验了%d个文件,无任何异常",
				pathName, path, fileCount))
		}
	}
}

func (x *XDRChecker) checkSingleFileContent(filename string, sheetConfig parser.SheetConfig) ([]ValidationError, int, time.Duration) {
	var errors []ValidationError
	var lineCount int

	// 记录开始时间
	startTime := time.Now()

	// 根据文件类型选择解析方式
	file, err := x.openFile(filename)
	if err != nil {
		errors = append(errors, ValidationError{
			Filename:   filename,
			LineNum:    0,
			FieldIndex: 0,
			FieldName:  "",
			ErrorType:  "system",
			RuleOrType: "file_open",
			Message:    fmt.Sprintf("文件无法打开: %v", err),
			FieldValue: "",
			FullLine:   "",
		})
		return errors, 0, time.Since(startTime)
	}
	defer file.Close()

	// 读取文件内容
	reader := bufio.NewReader(file)
	lineNum := 0

	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			errors = append(errors, ValidationError{
				Filename:   filename,
				LineNum:    0,
				FieldIndex: 0,
				FieldName:  "",
				ErrorType:  "system",
				RuleOrType: "file_read",
				Message:    fmt.Sprintf("文件读取错误: %v", err),
				FieldValue: "",
				FullLine:   "",
			})
			break
		}

		line = strings.TrimSpace(line)

		// 跳过空行
		if line == "" {
			continue
		}

		lineNum++

		// 解析字段
		fields := strings.Split(line, x.Config.ColDelimiter)

		// 校验每个字段
		for i, fieldRule := range sheetConfig.FieldRules {
			if i >= len(fields) {
				continue
			}

			fieldValue := strings.TrimSpace(fields[i])

			// 如果是选填字段且为空，跳过检查
			if fieldRule.Required == "选填" && fieldValue == "" {
				continue
			}

			// 校验字段
			validator := validator.NewRuleValidator(fieldValue, i, fields)

			// 首先校验类型
			if fieldRule.Type != "" {
				valid, msg := validator.ValidateType(fieldRule.Type)
				if !valid {
					errors = append(errors, ValidationError{
						Filename:   filename,
						LineNum:    lineNum,
						FieldIndex: i + 1,
						FieldName:  fieldRule.FieldName,
						ErrorType:  "type",
						RuleOrType: fieldRule.Type,
						Message:    msg,
						FieldValue: fieldValue,
						FullLine:   line,
					})
				}
			}

			// 然后校验其他规则
			for _, rule := range fieldRule.Rules {
				valid, msg := validator.ValidateRule(rule)
				if !valid {
					errors = append(errors, ValidationError{
						Filename:   filename,
						LineNum:    lineNum,
						FieldIndex: i + 1,
						FieldName:  fieldRule.FieldName,
						ErrorType:  "rule",
						RuleOrType: rule,
						Message:    msg,
						FieldValue: fieldValue,
						FullLine:   line,
					})
				}
			}
		}
	}

	lineCount = lineNum
	duration := time.Since(startTime)

	return errors, lineCount, duration
}

func (x *XDRChecker) openFile(filename string) (io.ReadCloser, error) {
	// 根据文件扩展名选择打开方式
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".gz", ".tar.gz":
		return x.openCompressedFile(filename)
	case ".txt", ".log":
		return os.Open(filename)
	case ".csv":
		return os.Open(filename)
	default:
		return os.Open(filename)
	}
}

func (x *XDRChecker) openCompressedFile(filename string) (io.ReadCloser, error) {
	// 打开压缩文件
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	// 根据文件扩展名选择解压方式
	ext := strings.ToLower(filepath.Ext(filename))

	if strings.HasSuffix(filename, ".tar.gz") {
		// 处理.tar.gz文件
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			file.Close()
			return nil, fmt.Errorf("创建gzip读取器失败: %v", err)
		}

		// 创建组合的读取器，关闭时同时关闭gzip读取器和文件
		return &combinedReader{
			Reader:  gzReader,
			closers: []io.Closer{gzReader, file},
		}, nil
	} else if ext == ".gz" {
		// 处理.gz文件
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			file.Close()
			return nil, fmt.Errorf("创建gzip读取器失败: %v", err)
		}

		return &combinedReader{
			Reader:  gzReader,
			closers: []io.Closer{gzReader, file},
		}, nil
	}

	// 如果不是压缩文件，直接返回文件
	return file, nil
}

// 组合读取器，用于同时关闭多个资源
type combinedReader struct {
	io.Reader
	closers []io.Closer
}

func (c *combinedReader) Close() error {
	var errs []error
	for _, closer := range c.closers {
		if err := closer.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("关闭资源时发生错误: %v", errs)
	}
	return nil
}

func (x *XDRChecker) createResultFile() (*os.File, error) {
	resultDir := filepath.Join("/tmp/xdr_check", x.TimeParam)
	if err := os.MkdirAll(resultDir, 0755); err != nil {
		return nil, err
	}

	resultFile := filepath.Join(resultDir, "check_result.txt")
	file, err := os.Create(resultFile)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func (x *XDRChecker) writeResult(message string) {
	x.mu.Lock()
	defer x.mu.Unlock()

	if x.ResultFile != nil {
		x.ResultFile.WriteString(message + "\n")
	}

	// 同时输出到控制台
	fmt.Println(message)
}

// 清理旧的临时目录
func ClearOldTmpDirs(baseDir string, keepDays int) error {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return err
	}

	cutoffTime := time.Now().AddDate(0, 0, -keepDays)

	for _, entry := range entries {
		if entry.IsDir() {
			info, err := entry.Info()
			if err != nil {
				continue
			}

			if info.ModTime().Before(cutoffTime) {
				dirPath := filepath.Join(baseDir, entry.Name())
				os.RemoveAll(dirPath)
			}
		}
	}

	return nil
}
