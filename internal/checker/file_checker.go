package checker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type FileTypeConfig struct {
	Headers      []string
	Suffix       string
	SizeLimit    string
	CheckContent string
	HeaderCheck  string // 首行校验（来源于sheet名称）
	FieldCount   string // 字段个数（来源于sheet名称）
}

type FileTypeFlag map[string]FileTypeConfig

func FileCheck(fileName, filePath string, fileTypeFlag FileTypeFlag, sheetName string) string {
	config, exists := fileTypeFlag[sheetName]
	if !exists {
		return "good"
	}

	// 检查文件名前缀（只使用严格前缀匹配）
	validPrefix := false
	if len(config.Headers) > 0 {
		//for _, header := range config.Headers {
		// 严格前缀匹配
		if strings.HasPrefix(fileName, config.Headers[0]) {
			validPrefix = true
			//break
		}
		//break
	}

	if !validPrefix {
		return fmt.Sprintf("<%s>文件名不符合要求", filePath)
	}

	// 检查文件后缀（如果配置了后缀检查）
	if config.Suffix != "" && config.Suffix != "不校验" {
		// 检查文件后缀是否匹配配置
		if !strings.HasSuffix(fileName, config.Suffix) {
			return fmt.Sprintf("<%s>文件后缀不符合要求", filePath)
		}
	}

	// 检查文件大小
	if config.SizeLimit != "不校验" && config.SizeLimit != "" {
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			return fmt.Sprintf("<%s>无法获取文件信息: %v", filePath, err)
		}

		sizeLimit := 24 // 默认大小限制
		if config.SizeLimit != "不校验" {
			var size int
			_, err := fmt.Sscanf(config.SizeLimit, "%d", &size)
			if err == nil {
				sizeLimit = size
			}
		}

		if fileInfo.Size() <= int64(sizeLimit) {
			return fmt.Sprintf("<%s>文件大小不符合要求", filePath)
		}
	}

	return "good"
}

func TraverseDirectory(path string, fileTypeFlag FileTypeFlag, sheetName string, scanNum int) ([]string, int, error) {
	var filenames []string
	var count int

	config, exists := fileTypeFlag[sheetName]
	if !exists {
		return nil, 0, fmt.Errorf("未找到配置: %s", sheetName)
	}

	// 检查目录是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// 目录不存在，静默跳过，不报错
		return []string{}, 0, nil
	}

	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			// 文件访问错误，静默跳过
			return nil
		}

		if info.IsDir() {
			return nil
		}

		fileName := info.Name()
		result := FileCheck(fileName, filePath, fileTypeFlag, sheetName)

		if result == "good" && config.CheckContent == "校验" {
			// 检查文件头匹配（只使用严格前缀匹配）
			if len(config.Headers) > 0 {
				//for _, header := range config.Headers {
				// 严格前缀匹配
				if strings.HasPrefix(fileName, config.Headers[0]) {
					filenames = append(filenames, filePath)
					//break
				}
				//break
				//}
			}
		}

		count++
		return nil
	})

	if err != nil {
		// 目录遍历错误，静默跳过
		return filenames, count, nil
	}

	// 抽样检查
	if scanNum > 0 {
		if scanNum < len(filenames) {
			filenames = sampleFiles(filenames, scanNum)
			// 显示抽样检查信息
			fmt.Printf("抽样检查%s: %d/%d个文件\n", path, scanNum, len(filenames))
		} else {
			fmt.Printf("全量检查%s: %d个文件\n", path, len(filenames))
		}
	}

	if config.CheckContent == "校验" {
		// 当需要校验内容时，count应该表示需要校验的文件数
		count = len(filenames)
	}

	return filenames, count, nil
}

func sampleFiles(files []string, num int) []string {
	if num >= len(files) {
		return files
	}

	// 简单的抽样实现（Go版本使用固定种子确保可重复性）
	result := make([]string, num)
	for i := 0; i < num; i++ {
		// 均匀抽样
		index := i * len(files) / num
		if index < len(files) {
			result[i] = files[index]
		}
	}
	return result
}
