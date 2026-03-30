package core

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"xdrCheck/internal/checker"
	"xdrCheck/internal/config"
	"xdrCheck/internal/parser"
	"xdrCheck/internal/validator"

	"github.com/olekukonko/tablewriter"
)

// TableReportConfig 表格报告配置
type TableReportConfig struct {
	ShowFileName   bool // 显示文件名
	ShowLineNumber bool // 显示行号
	ShowFieldValue bool // 显示字段值
	ShowErrorType  bool // 显示错误类型
	ShowCondition  bool // 显示条件规则
	MaxColumnWidth int  // 最大列宽
}

// DefaultTableReportConfig 默认表格报告配置
func DefaultTableReportConfig() TableReportConfig {
	return TableReportConfig{
		ShowFileName:   true,
		ShowLineNumber: true,
		ShowFieldValue: true,
		ShowErrorType:  true,
		ShowCondition:  true,
		MaxColumnWidth: 30,
	}
}

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

// CheckTask 定义检查任务结构体
type CheckTask struct {
	Filename    string             // 文件名
	PathName    string             // 路径名称
	SheetConfig parser.SheetConfig // 检查规则配置
	IsSpecial   bool               // 是否为特殊路径
}

// CheckResult 定义检查结果结构体
type CheckResult struct {
	Task      CheckTask         // 原始任务
	Errors    []ValidationError // 检查错误
	LineCount int               // 检查行数
	Duration  time.Duration     // 检查耗时
	Success   bool              // 是否成功
	ErrorMsg  string            // 错误消息（如果有）
}

type XDRChecker struct {
	Config       *config.Config
	TimeParam    string
	ScanNum      int
	NoSubPath    bool
	ResultFile   *os.File
	mu           sync.Mutex
	WorkerNum    int    // 协程数，默认4
	ReportFormat string // 报告格式：txt, table, html
}

func NewXDRChecker(cfg *config.Config, timeParam string, scanNum int, noSubPath bool, workerNum int, reportFormat string) *XDRChecker {
	// 如果workerNum为0或负数，使用默认值4
	if workerNum <= 0 {
		workerNum = 4
	}

	// 验证报告格式
	if reportFormat != "txt" && reportFormat != "table" && reportFormat != "html" {
		reportFormat = "txt" // 默认格式
	}

	return &XDRChecker{
		Config:       cfg,
		TimeParam:    timeParam,
		ScanNum:      scanNum,
		NoSubPath:    noSubPath,
		WorkerNum:    workerNum,
		ReportFormat: reportFormat,
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

	// 设置协程数，默认4
	if x.WorkerNum <= 0 {
		x.WorkerNum = 4
	}

	// 第一阶段：扫描所有文件，构建任务列表
	var allTasks []CheckTask
	var totalFiles int

	for pathName, pathValue := range x.Config.XDRPaths {
		// 查找对应的sheet配置
		sheetConfig, found := x.findSheetConfig(sheetConfigs, pathName)
		if !found {
			continue // 静默跳过，不输出警告
		}

		// 构建检查路径
		checkPath := pathValue
		if x.TimeParam != "" {
			checkPath = filepath.Join(pathValue, x.TimeParam, "success")
		}

		// 扫描该路径下的所有文件
		filenames, count, err := x.scanFilesForPath(checkPath, pathName, sheetConfig)
		if err != nil {
			x.writeResult(fmt.Sprintf("扫描路径%s失败: %v", pathName, err))
			continue
		}

		// 特殊路径也创建任务，在worker中统一处理
		isSpecial := x.isSpecialPath(pathName)
		// 为每个文件创建检查任务
		for _, filename := range filenames {
			allTasks = append(allTasks, CheckTask{
				Filename:    filename,
				PathName:    pathName,
				SheetConfig: sheetConfig,
				IsSpecial:   isSpecial,
			})
		}
		totalFiles += count
	}

	x.writeResult(fmt.Sprintf("扫描完成，共发现%d个文件，准备使用%d个协程进行处理", totalFiles, x.WorkerNum))

	// 第二阶段：使用协程池处理所有任务
	return x.processTasksWithWorkerPool(allTasks, x.WorkerNum)
}

// findSheetConfig 查找对应的sheet配置
func (x *XDRChecker) findSheetConfig(sheetConfigs []parser.SheetConfig, pathName string) (parser.SheetConfig, bool) {
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

	return sheetConfig, found
}

// scanFilesForPath 扫描指定路径下的文件
func (x *XDRChecker) scanFilesForPath(checkPath, pathName string, sheetConfig parser.SheetConfig) ([]string, int, error) {
	// 构建文件类型配置
	fileTypeFlag := make(checker.FileTypeFlag)

	// 查找对应路径的文件校验配置
	var fileValidationConfig parser.FileValidationConfig
	foundFileConfig := false

	if sheetConfig.FileValidation.FileHeader != "" {
		fileValidationConfig = sheetConfig.FileValidation
		foundFileConfig = true
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

	// 遍历目录并获取文件列表
	filenames, count, err := checker.TraverseDirectory(checkPath, fileTypeFlag, pathName, x.ScanNum)
	if err != nil {
		return nil, 0, fmt.Errorf("目录遍历错误: %v", err)
	}

	return filenames, count, nil
}

// processTasksWithWorkerPool 使用协程池处理任务，单协程写入文件
func (x *XDRChecker) processTasksWithWorkerPool(tasks []CheckTask, workerNum int) error {
	if len(tasks) == 0 {
		x.writeResult("没有发现需要检查的文件")
		return nil
	}

	// 创建任务通道和结果通道
	taskChan := make(chan CheckTask, len(tasks))
	resultChan := make(chan CheckResult, len(tasks))

	// 启动worker协程
	var wg sync.WaitGroup
	for i := 0; i < workerNum; i++ {
		wg.Add(1)
		go x.worker(i, taskChan, resultChan, &wg)
	}

	// 启动文件写入协程
	var fileWg sync.WaitGroup
	fileWg.Add(1)
	go x.fileWriter(resultChan, &fileWg)

	// 发送任务到通道
	for _, task := range tasks {
		taskChan <- task
	}
	close(taskChan)

	// 等待所有worker完成
	wg.Wait()
	close(resultChan)

	// 等待文件写入完成
	fileWg.Wait()

	x.writeResult("所有文件检查完成")
	return nil
}

// worker 协程处理函数
func (x *XDRChecker) worker(id int, taskChan <-chan CheckTask, resultChan chan<- CheckResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for task := range taskChan {
		// 特殊路径处理
		if task.IsSpecial {
			x.writeResult(fmt.Sprintf("[Worker %d] 特殊路径%s: 执行特殊处理", id, task.PathName))

			// 执行特殊路径处理
			// 注意：特殊路径处理需要正确的路径，这里使用PathName作为路径标识
			if err := x.handleSpecialPath(task.PathName, task.PathName, []string{task.Filename}); err != nil {
				x.writeResult(fmt.Sprintf("[Worker %d] 特殊路径%s处理失败: %v", id, task.PathName, err))
			}

			// 特殊路径不进行文件检查，直接返回成功结果
			result := CheckResult{
				Task:      task,
				Errors:    []ValidationError{},
				LineCount: 0,
				Duration:  0,
				Success:   true,
				ErrorMsg:  fmt.Sprintf("特殊路径%s处理完成", task.PathName),
			}
			resultChan <- result
			continue
		}

		// 普通文件处理
		// 处理单个文件检查
		errors, lineCount, duration := x.checkSingleFileContent(task.Filename, task.SheetConfig)

		// 显示文件检查统计信息
		x.writeResult(fmt.Sprintf("[Worker %d] 文件%s: 检查%d行, 耗时%s, 发现%d个错误",
			id, task.Filename, lineCount, duration, len(errors)))

		// 发送处理结果到文件写入协程
		result := CheckResult{
			Task:      task,
			Errors:    errors,
			LineCount: lineCount,
			Duration:  duration,
			Success:   len(errors) == 0,
		}

		if len(errors) > 0 {
			result.ErrorMsg = fmt.Sprintf("文件%s检查发现%d个错误", task.Filename, len(errors))
		}

		resultChan <- result
	}
}

// fileWriter 文件写入协程，单协程负责所有文件写入
func (x *XDRChecker) fileWriter(resultChan <-chan CheckResult, wg *sync.WaitGroup) {
	defer wg.Done()

	// 创建结果目录
	resultDir := x.createResultDirectory()

	// 用于跟踪每个路径的结果文件
	resultFiles := make(map[string]*os.File)
	writers := make(map[string]*bufio.Writer)

	// 统计信息
	startTime := time.Now()
	stats := make(map[string]struct {
		FileCount   int
		ErrorCount  int
		TotalLines  int
		TotalErrors int
		ResultFile  string
	})

	// 处理所有结果
	for result := range resultChan {
		pathName := result.Task.PathName

		// 更新统计信息
		if stat, exists := stats[pathName]; exists {
			stat.FileCount++
			stat.TotalLines += result.LineCount
			stat.TotalErrors += len(result.Errors)
			if len(result.Errors) > 0 {
				stat.ErrorCount++
			}
			stats[pathName] = stat
		} else {
			stats[pathName] = struct {
				FileCount   int
				ErrorCount  int
				TotalLines  int
				TotalErrors int
				ResultFile  string
			}{
				FileCount:   1,
				ErrorCount:  ternary(len(result.Errors) > 0, 1, 0),
				TotalLines:  result.LineCount,
				TotalErrors: len(result.Errors),
				ResultFile:  "", // 初始化为空，稍后在有错误时设置
			}
		}

		// 如果有错误，写入结果文件
		if len(result.Errors) > 0 {
			resultFile := filepath.Join(resultDir, pathName+".txt")

			// 更新统计信息中的结果文件名
			if stat, exists := stats[pathName]; exists {
				stat.ResultFile = resultFile
				stats[pathName] = stat
			}

			// 如果文件还未打开，则打开文件
			if _, exists := resultFiles[resultFile]; !exists {
				file, err := os.OpenFile(resultFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					x.writeResult(fmt.Sprintf("无法打开结果文件%s: %v", resultFile, err))
					continue
				}

				resultFiles[resultFile] = file
				writers[resultFile] = bufio.NewWriter(file)
				x.writeResult(fmt.Sprintf("创建结果文件: %s", resultFile))
			}

			// 写入错误信息
			writer := writers[resultFile]
			x.writeFormattedErrors(writer, result.Task.Filename, result.Errors)
			x.writeResult(fmt.Sprintf("写入错误信息到文件: %s (源文件: %s, 错误数: %d)",
				resultFile, result.Task.Filename, len(result.Errors)))
		}
	}

	// 刷新并关闭所有文件
	for resultFile, writer := range writers {
		writer.Flush()
		if file, exists := resultFiles[resultFile]; exists {
			file.Close()
		}
	}

	// 打印统计报告
	x.printStatisticsReport(stats, time.Since(startTime))
}

// ternary 三目运算符函数
func ternary(condition bool, trueVal, falseVal int) int {
	if condition {
		return trueVal
	}
	return falseVal
}

// printStatisticsReport 打印统计报告
func (x *XDRChecker) printStatisticsReport(stats map[string]struct {
	FileCount   int
	ErrorCount  int
	TotalLines  int
	TotalErrors int
	ResultFile  string
}, duration time.Duration) {
	x.writeResult("\n=== 文件检查统计报告 ===")
	x.writeResult(fmt.Sprintf("总处理时间: %s", duration))
	x.writeResult("-")

	totalFiles := 0
	totalErrorFiles := 0
	totalLines := 0
	totalErrors := 0

	// 按路径名称排序输出
	var paths []string
	for path := range stats {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, path := range paths {
		stat := stats[path]
		x.writeResult(fmt.Sprintf("路径: %s", path))
		x.writeResult(fmt.Sprintf("  文件总数: %d", stat.FileCount))
		x.writeResult(fmt.Sprintf("  错误文件数: %d", stat.ErrorCount))
		x.writeResult(fmt.Sprintf("  总行数: %d", stat.TotalLines))
		x.writeResult(fmt.Sprintf("  总错误数: %d", stat.TotalErrors))

		// 如果有错误，显示结果文件名
		if stat.TotalErrors > 0 && stat.ResultFile != "" {
			x.writeResult(fmt.Sprintf("  结果文件: %s", stat.ResultFile))
		}

		x.writeResult("-")

		totalFiles += stat.FileCount
		totalErrorFiles += stat.ErrorCount
		totalLines += stat.TotalLines
		totalErrors += stat.TotalErrors
	}

	x.writeResult("=== 汇总统计 ===")
	x.writeResult(fmt.Sprintf("总文件数: %d", totalFiles))
	x.writeResult(fmt.Sprintf("总错误文件数: %d", totalErrorFiles))
	x.writeResult(fmt.Sprintf("总检查行数: %d", totalLines))
	x.writeResult(fmt.Sprintf("总错误数: %d", totalErrors))
	x.writeResult(fmt.Sprintf("错误率: %.2f%%", float64(totalErrors)/float64(totalLines)*100))
	x.writeResult(fmt.Sprintf("文件错误率: %.2f%%", float64(totalErrorFiles)/float64(totalFiles)*100))
	x.writeResult("================")
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

// 格式化输出错误信息
func (x *XDRChecker) writeFormattedErrors(writer *bufio.Writer, filename string, errors []ValidationError) {
	// 根据报告格式选择输出方式
	switch x.ReportFormat {
	case "table":
		x.writeTableFormatErrors(writer, filename, errors)
	case "html":
		x.writeHTMLFormatErrors(writer, filename, errors)
	default: // txt 格式
		x.writeTextFormatErrors(writer, filename, errors)
	}
}

// 文本格式输出错误信息
func (x *XDRChecker) writeTextFormatErrors(writer *bufio.Writer, filename string, errors []ValidationError) {
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

// 表格格式输出错误信息
func (x *XDRChecker) writeTableFormatErrors(writer *bufio.Writer, filename string, errors []ValidationError) {
	if len(errors) == 0 {
		writer.WriteString(fmt.Sprintf("✅ 文件 %s 验证通过，无错误信息\n", filename))
		return
	}

	// 创建表格
	table := tablewriter.NewWriter(writer)
	table.Header("字段名", "错误类型", "错误信息", "字段值", "行号")

	// 添加数据行
	for _, err := range errors {
		table.Append(err.FieldName, err.ErrorType, err.Message, err.FieldValue, strconv.Itoa(err.LineNum))
	}

	// 写入文件标题
	writer.WriteString(fmt.Sprintf("文件: %s\n", filename))
	writer.WriteString(fmt.Sprintf("错误数量: %d\n", len(errors)))
	writer.WriteString("\n")

	// 渲染表格
	table.Render()
	writer.WriteString("\n")
}

// HTML格式输出错误信息（基础版本）
func (x *XDRChecker) writeHTMLFormatErrors(writer *bufio.Writer, filename string, errors []ValidationError) {
	writer.WriteString("<!DOCTYPE html>\n")
	writer.WriteString("<html>\n")
	writer.WriteString("<head>\n")
	writer.WriteString("<meta charset=\"UTF-8\">\n")
	writer.WriteString("<title>XDR检查报告 - " + filename + "</title>\n")
	writer.WriteString("<style>\n")
	writer.WriteString("body { font-family: Arial, sans-serif; margin: 20px; }\n")
	writer.WriteString("h1 { color: #333; }\n")
	writer.WriteString("table { border-collapse: collapse; width: 100%; margin: 20px 0; }\n")
	writer.WriteString("th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }\n")
	writer.WriteString("th { background-color: #f2f2f2; }\n")
	writer.WriteString("tr:nth-child(even) { background-color: #f9f9f9; }\n")
	writer.WriteString(".error { color: red; }\n")
	writer.WriteString(".success { color: green; }\n")
	writer.WriteString("</style>\n")
	writer.WriteString("</head>\n")
	writer.WriteString("<body>\n")

	if len(errors) == 0 {
		writer.WriteString("<h3 class=\"success\">✅ 文件 " + filename + " 验证通过</h3>\n")
	} else {
		writer.WriteString("<h3 class=\"error\">❌ 文件 " + filename + " 错误报告</h3>\n")
		writer.WriteString("<p>错误数量: " + fmt.Sprintf("%d", len(errors)) + "</p>\n")

		// 生成表格
		writer.WriteString("<table>\n")
		writer.WriteString("<tr><th>字段名</th><th>错误信息</th><th>行号</th></tr>\n")

		for _, err := range errors {
			writer.WriteString(fmt.Sprintf("<tr><td>%s</td><td class=\"error\">%s</td><td>%d</td></tr>\n",
				err.FieldName, err.Message, err.LineNum))
		}

		writer.WriteString("</table>\n")
	}

	writer.WriteString("</body>\n")
	writer.WriteString("</html>\n")
}

// GenerateTableReport 生成表格化详细报告
func (x *XDRChecker) GenerateTableReport(errors []ValidationError) (string, error) {
	config := DefaultTableReportConfig()
	return x.GenerateTableReportWithConfig(errors, config)
}

// GenerateTableReportWithConfig 使用自定义配置生成表格化详细报告
func (x *XDRChecker) GenerateTableReportWithConfig(errors []ValidationError, config TableReportConfig) (string, error) {
	if len(errors) == 0 {
		return "✅ 所有文件验证通过，无错误信息", nil
	}

	var report strings.Builder

	// 生成报告头部
	report.WriteString(x.generateTableHeader(config))
	report.WriteString("\n")

	// 按文件名分组错误
	errorGroups := x.groupErrorsByFile(errors)

	// 生成表格内容
	for filename, fileErrors := range errorGroups {
		// 文件名标题
		report.WriteString(fmt.Sprintf("📄 文件: %s\n", filepath.Base(filename)))
		report.WriteString(x.generateTableSeparator(config))
		report.WriteString("\n")

		// 表格内容
		for _, err := range fileErrors {
			report.WriteString(x.generateTableRow(err, config))
			report.WriteString("\n")
		}

		report.WriteString("\n")
	}

	// 生成统计信息
	report.WriteString(x.generateStatistics(errors))

	return report.String(), nil
}

// generateTableHeader 生成表格头部
func (x *XDRChecker) generateTableHeader(config TableReportConfig) string {
	var header strings.Builder
	var columns []string

	// 构建列标题
	if config.ShowFileName {
		columns = append(columns, "文件名")
	}
	if config.ShowLineNumber {
		columns = append(columns, "行号")
	}
	columns = append(columns, "字段名")
	if config.ShowFieldValue {
		columns = append(columns, "字段值")
	}
	if config.ShowErrorType {
		columns = append(columns, "错误类型")
	}
	columns = append(columns, "错误描述")
	if config.ShowCondition {
		columns = append(columns, "条件规则")
	}

	// 生成表头
	for i, col := range columns {
		width := x.calculateColumnWidth(col, config)
		if i == 0 {
			header.WriteString("│ ")
		} else {
			header.WriteString(" │ ")
		}
		header.WriteString(fmt.Sprintf("%-*s", width, col))
	}
	header.WriteString(" │")

	return header.String()
}

// generateTableSeparator 生成表格分隔线
func (x *XDRChecker) generateTableSeparator(config TableReportConfig) string {
	var separator strings.Builder
	var columns []string

	// 构建列标题（用于计算宽度）
	if config.ShowFileName {
		columns = append(columns, "文件名")
	}
	if config.ShowLineNumber {
		columns = append(columns, "行号")
	}
	columns = append(columns, "字段名")
	if config.ShowFieldValue {
		columns = append(columns, "字段值")
	}
	if config.ShowErrorType {
		columns = append(columns, "错误类型")
	}
	columns = append(columns, "错误描述")
	if config.ShowCondition {
		columns = append(columns, "条件规则")
	}

	// 生成分隔线
	for i, col := range columns {
		width := x.calculateColumnWidth(col, config)
		if i == 0 {
			separator.WriteString("├─")
		} else {
			separator.WriteString("─┼─")
		}
		separator.WriteString(strings.Repeat("─", width))
	}
	separator.WriteString("─┤")

	return separator.String()
}

// generateTableRow 生成表格行
func (x *XDRChecker) generateTableRow(err ValidationError, config TableReportConfig) string {
	var row strings.Builder

	// 文件名
	if config.ShowFileName {
		filename := filepath.Base(err.Filename)
		width := x.calculateColumnWidth("文件名", config)
		row.WriteString(fmt.Sprintf("│ %-*s", width, x.truncateString(filename, width)))
	}

	// 行号
	if config.ShowLineNumber {
		lineNum := fmt.Sprintf("%d", err.LineNum)
		width := x.calculateColumnWidth("行号", config)
		if config.ShowFileName {
			row.WriteString(" │ ")
		} else {
			row.WriteString("│ ")
		}
		row.WriteString(fmt.Sprintf("%-*s", width, lineNum))
	}

	// 字段名
	fieldName := err.FieldName
	width := x.calculateColumnWidth("字段名", config)
	if config.ShowFileName || config.ShowLineNumber {
		row.WriteString(" │ ")
	} else {
		row.WriteString("│ ")
	}
	row.WriteString(fmt.Sprintf("%-*s", width, fieldName))

	// 字段值
	if config.ShowFieldValue {
		fieldValue := err.FieldValue
		if fieldValue == "" {
			fieldValue = "<空>"
		}
		width := x.calculateColumnWidth("字段值", config)
		row.WriteString(fmt.Sprintf(" │ %-*s", width, x.truncateString(fieldValue, width)))
	}

	// 错误类型
	if config.ShowErrorType {
		errorType := x.translateErrorType(err.ErrorType)
		width := x.calculateColumnWidth("错误类型", config)
		row.WriteString(fmt.Sprintf(" │ %-*s", width, errorType))
	}

	// 错误描述
	errorDesc := err.Message
	errorDescWidth := x.calculateColumnWidth("错误描述", config)
	row.WriteString(fmt.Sprintf(" │ %-*s", errorDescWidth, x.truncateString(errorDesc, errorDescWidth)))

	// 条件规则
	if config.ShowCondition && err.RuleOrType != "" {
		rule := err.RuleOrType
		ruleWidth := x.calculateColumnWidth("条件规则", config)
		row.WriteString(fmt.Sprintf(" │ %-*s", ruleWidth, x.truncateString(rule, ruleWidth)))
	}

	row.WriteString(" │")

	return row.String()
}

// generateStatistics 生成统计信息
func (x *XDRChecker) generateStatistics(errors []ValidationError) string {
	var stats strings.Builder
	stats.WriteString("📊 错误统计信息\n")
	stats.WriteString("─────────────────────────────────────────────\n")

	// 按错误类型统计
	typeCount := make(map[string]int)
	fileCount := make(map[string]bool)
	fieldCount := make(map[string]bool)

	for _, err := range errors {
		typeCount[err.ErrorType]++
		fileCount[err.Filename] = true
		fieldCount[err.FieldName] = true
	}

	stats.WriteString(fmt.Sprintf("总错误数: %d\n", len(errors)))
	stats.WriteString(fmt.Sprintf("涉及文件: %d\n", len(fileCount)))
	stats.WriteString(fmt.Sprintf("涉及字段: %d\n", len(fieldCount)))
	stats.WriteString("\n错误类型分布:\n")

	for errType, count := range typeCount {
		stats.WriteString(fmt.Sprintf("  • %s: %d\n", x.translateErrorType(errType), count))
	}

	return stats.String()
}

// groupErrorsByFile 按文件名分组错误
func (x *XDRChecker) groupErrorsByFile(errors []ValidationError) map[string][]ValidationError {
	groups := make(map[string][]ValidationError)
	for _, err := range errors {
		groups[err.Filename] = append(groups[err.Filename], err)
	}
	return groups
}

// calculateColumnWidth 计算列宽
func (x *XDRChecker) calculateColumnWidth(columnName string, config TableReportConfig) int {
	baseWidths := map[string]int{
		"文件名":  20,
		"行号":   6,
		"字段名":  12,
		"字段值":  15,
		"错误类型": 8,
		"错误描述": 25,
		"条件规则": 20,
	}

	width, exists := baseWidths[columnName]
	if !exists {
		width = 15
	}

	if width > config.MaxColumnWidth {
		return config.MaxColumnWidth
	}
	return width
}

// truncateString 截断字符串
func (x *XDRChecker) truncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength-3] + "..."
}

// translateErrorType 翻译错误类型
func (x *XDRChecker) translateErrorType(errorType string) string {
	translations := map[string]string{
		"condition": "条件错误",
		"type":      "类型错误",
		"rule":      "规则错误",
	}

	if translated, exists := translations[errorType]; exists {
		return translated
	}
	return errorType
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
			// 检查是否已经存在相同的错误信息，避免重复添加
			if !strings.Contains(existing, errorMsg) {
				fieldErrors[err.FieldName] = existing + "; " + errorMsg
			}
		} else {
			fieldErrors[err.FieldName] = errorMsg
		}
	}

	return fieldErrors
}

// 提取原始日志内容

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

	// 检查目录是否存在
	if _, err := os.Stat(resultDir); err == nil {
		// 目录存在，先清空目录
		if err := os.RemoveAll(resultDir); err != nil {
			// 清空失败，记录错误但继续创建目录
			fmt.Printf("警告: 清空目录 %s 失败: %v\n", resultDir, err)
		}
	}

	// 创建目录
	if err := os.MkdirAll(resultDir, 0755); err != nil {
		fmt.Printf("错误: 创建结果目录 %s 失败: %v\n", resultDir, err)
		return ""
	}

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

			// 如果是空字段，跳过检查
			if fieldRule.Required == "空" {
				continue
			}

			// 校验字段
			validator := validator.NewRuleValidator(fieldValue, i, fields, sheetConfig.FieldNumberMap)

			// 首先校验条件（如果有）
			if fieldRule.Condition != "" {
				// 对于选填字段且为空的情况，需要特殊处理
				if fieldRule.Required == "选填" && fieldValue == "" {
					// 选填字段为空时，必须验证条件
					valid, msg := validator.ValidateCondition(fieldRule.Condition)
					if !valid {
						errors = append(errors, ValidationError{
							Filename:   filename,
							LineNum:    lineNum,
							FieldIndex: i + 1,
							FieldName:  fieldRule.FieldName,
							ErrorType:  "condition",
							RuleOrType: fieldRule.Condition,
							Message:    msg,
							FieldValue: fieldValue,
							FullLine:   line,
						})
					}
				} else {
					// 其他情况（必填字段或选填字段有值）
					valid, msg := validator.ValidateCondition(fieldRule.Condition)
					if !valid {
						errors = append(errors, ValidationError{
							Filename:   filename,
							LineNum:    lineNum,
							FieldIndex: i + 1,
							FieldName:  fieldRule.FieldName,
							ErrorType:  "condition",
							RuleOrType: fieldRule.Condition,
							Message:    msg,
							FieldValue: fieldValue,
							FullLine:   line,
						})
					}
				}
			}

			// 然后校验类型
			if fieldRule.Type != "" {
				// 对于选填字段且为空的情况，跳过类型校验
				if fieldRule.Required == "选填" && fieldValue == "" && fieldRule.Condition == "" {
					// 选填字段为空且无条件规则时，跳过类型校验
				} else {
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
			}

			// 然后校验其他规则
			for _, rule := range fieldRule.Rules {
				// 对于选填字段且为空的情况，跳过规则校验
				if fieldRule.Required == "选填" && fieldValue == "" && fieldRule.Condition == "" {
					// 选填字段为空且无条件规则时，跳过规则校验
				} else {
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
		// 处理.tar.gz文件 - 先解压gzip，再解析tar格式
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			file.Close()
			return nil, fmt.Errorf("创建gzip读取器失败: %v", err)
		}

		// 解析tar格式，提取第一个文件的内容
		tarReader := tar.NewReader(gzReader)

		// 读取tar文件头，找到第一个普通文件
		for {
			header, err := tarReader.Next()
			if err == io.EOF {
				break // tar文件结束
			}
			if err != nil {
				gzReader.Close()
				file.Close()
				return nil, fmt.Errorf("读取tar文件头失败: %v", err)
			}

			// 如果是普通文件，返回其内容
			if header.Typeflag == tar.TypeReg {
				// 创建组合的读取器，关闭时同时关闭所有资源
				return &combinedReader{
					Reader:  tarReader,
					closers: []io.Closer{gzReader, file},
				}, nil
			}
		}

		gzReader.Close()
		file.Close()
		return nil, fmt.Errorf("tar文件中未找到普通文件")
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

	// 根据报告格式确定文件后缀
	fileExt := ".txt"
	if x.ReportFormat == "html" {
		fileExt = ".html"
	}

	resultFile := filepath.Join(resultDir, "check_result"+fileExt)
	file, err := os.Create(resultFile)
	if err != nil {
		return nil, err
	}

	// 写入文件头
	writer := bufio.NewWriter(file)

	// 根据格式写入不同的文件头
	switch x.ReportFormat {
	case "html":
		writer.WriteString("<!DOCTYPE html>\n")
		writer.WriteString("<html>\n")
		writer.WriteString("<head>\n")
		writer.WriteString("<meta charset=\"UTF-8\">\n")
		writer.WriteString("<title>XDR检查结果报告</title>\n")
		writer.WriteString("<style>\n")
		writer.WriteString("body { font-family: Arial, sans-serif; margin: 20px; }\n")
		writer.WriteString("h1 { color: #333; }\n")
		writer.WriteString("table { border-collapse: collapse; width: 100%; margin: 20px 0; }\n")
		writer.WriteString("th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }\n")
		writer.WriteString("th { background-color: #f2f2f2; }\n")
		writer.WriteString("tr:nth-child(even) { background-color: #f9f9f9; }\n")
		writer.WriteString(".error { color: red; }\n")
		writer.WriteString(".success { color: green; }\n")
		writer.WriteString("</style>\n")
		writer.WriteString("</head>\n")
		writer.WriteString("<body>\n")
		writer.WriteString("<h1>XDR检查结果报告</h1>\n")
		writer.WriteString("<p>生成时间: " + time.Now().Format("2006-01-02 15:04:05") + "</p>\n")
		writer.WriteString("<p>检查时间: " + x.TimeParam + "</p>\n")
		writer.WriteString("<hr>\n")
	default:
		writer.WriteString("XDR检查结果报告\n")
		writer.WriteString("生成时间: " + time.Now().Format("2006-01-02 15:04:05") + "\n")
		writer.WriteString("检查时间: " + x.TimeParam + "\n")
		writer.WriteString("\n")
	}

	writer.Flush()

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
