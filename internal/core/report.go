package core

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
)

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
