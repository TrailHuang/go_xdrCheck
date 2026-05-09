package parser

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"xdrCheck/internal/config"

	"github.com/360EntSecGroup-Skylar/excelize"
)

// EnumRange 预解析的枚举范围
type EnumRange struct {
	Min int64
	Max int64
}

// ParsedEnumValue 预解析的枚举值
type ParsedEnumValue struct {
	ExactValues map[string]struct{} // 精确匹配值
	Ranges      []EnumRange         // 范围值
	RawRule     string              // 原始规则字符串（用于错误消息）
}

// ParsedCondition 预解析的条件表达式
type ParsedCondition struct {
	FieldIndex    int                 // 字段索引
	ExpectedExact map[string]struct{} // 等于条件的期望值列表
	ExpectedOrder []string            // 期望值的有序列表，用于条件类型映射（如 if($12==1,2);type=ipv4,ipv6）
	IsEqual       bool                // true: ==, false: !=
}

type FieldRule struct {
	FieldName  string
	Required   string // "必填" or "选填" or "空"
	Type       string // 数据类型 (int, ip, datetime, etc.)
	Rules      []string
	Condition  string // 条件表达式，如 "if($13==5,8)"
	Offset     string // 偏移规则，如 "offset(6,4)"
	Array      string // 数组规则，如 "array(10,11,12)"
	Loop       string // 循环规则，如 "loop(start=,)"
	Jump       string // 跳转规则，如 "jump=1"
	Regex      string // 正则表达式，如 "reg=[^ ]+"
	ConfigItem string // 配置项路径，如 "DEFAULT.manufacture_id"
	// 预解析缓存（在 PreParseRules 中填充）
	ParsedEnums     map[string]*ParsedEnumValue // 枚举规则 -> 预解析结果
	ParsedCondition *ParsedCondition            // 条件表达式预解析结果
}

type FileValidationConfig struct {
	FileHeader   string // 文件头
	FileSuffix   string // 文件后缀
	FileSize     string // 文件大小
	CheckContent string // 文件内容检查
	HeaderCheck  string // 首行校验（来源于sheet名称）
	FieldCount   string // 字段个数（来源于sheet名称）
}

type SheetConfig struct {
	SheetName      string
	FieldRules     []FieldRule
	FileValidation FileValidationConfig
	FieldNumberMap map[string]int // 字段编号到索引的映射（如 "11" -> 索引）
}

func ParseExcelTemplate(filePath string, cfg *config.Config) ([]SheetConfig, error) {
	xlsx, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开Excel文件失败: %v", err)
	}

	var sheetConfigs []SheetConfig

	// 先解析文件校验工作表
	fileConfigs := parseFileValidationSheet(xlsx)

	// 获取所有工作表（按顺序）
	sheets := getOrderedSheetList(xlsx)

	// 然后解析其他工作表
	for _, sheetName := range sheets {
		// 跳过空工作表或文件校验工作表
		if sheetName == "" || sheetName == "文件校验" {
			continue
		}

		sheetConfig, err := parseSheet(xlsx, sheetName, cfg)
		if err != nil {
			continue // 继续处理其他工作表，不中断整个流程
		}

		if len(sheetConfig.FieldRules) > 0 {
			sheetConfigs = append(sheetConfigs, sheetConfig)
		}
	}

	// 合并文件校验配置和字段规则配置
	mergedConfigs := mergeSheetConfigs(fileConfigs, sheetConfigs)

	// 预解析所有规则（枚举、条件等），避免运行时重复Split
	PreParseRules(mergedConfigs)

	return mergedConfigs, nil
}

// 解析文件校验工作表
func parseFileValidationSheet(xlsx *excelize.File) []SheetConfig {
	var configs []SheetConfig

	// 检查文件校验工作表是否存在
	sheetIndex := xlsx.GetSheetIndex("文件校验")
	if sheetIndex == -1 {
		return configs
	}

	// 获取文件校验工作表数据
	rows := xlsx.GetRows("文件校验")
	if len(rows) < 2 {
		return configs
	}

	// 解析表头，确定列索引
	headers := rows[0]
	colIndex := make(map[string]int)
	for i, header := range headers {
		colIndex[strings.TrimSpace(header)] = i
	}

	// 解析数据行
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		if len(row) == 0 {
			continue
		}

		// 创建文件校验配置
		config := SheetConfig{
			SheetName: strings.TrimSpace(row[0]), // sheet名称
			FileValidation: FileValidationConfig{
				FileHeader:   strings.TrimSpace(row[1]), // 文件头
				FileSuffix:   strings.TrimSpace(row[2]), // 文件后缀
				FileSize:     strings.TrimSpace(row[3]), // 文件大小
				CheckContent: strings.TrimSpace(row[4]), // 文件内容
				HeaderCheck:  strings.TrimSpace(row[5]), // 首行校验（第6列）
				FieldCount:   strings.TrimSpace(row[6]), // 字段个数（第7列）
			},
		}

		configs = append(configs, config)
	}

	return configs
}

func parseSheet(xlsx *excelize.File, sheetName string, cfg *config.Config) (SheetConfig, error) {
	sheetConfig := SheetConfig{
		SheetName:      sheetName,
		FieldNumberMap: make(map[string]int),
	}

	// 获取工作表的所有行
	rows := xlsx.GetRows(sheetName)
	if rows == nil {
		return sheetConfig, fmt.Errorf("无法获取工作表%s的行数据", sheetName)
	}

	if len(rows) < 2 {
		return sheetConfig, nil
	}

	// 解析表头，确定列索引
	headers := rows[0]
	colIndex := make(map[string]int)
	for i, header := range headers {
		colIndex[strings.TrimSpace(header)] = i
	}

	// 第二行开始是规则
	for rowIndex := 1; rowIndex < len(rows); rowIndex++ {
		row := rows[rowIndex]
		if len(row) == 0 {
			continue
		}

		// 获取字段编号（第一列）
		fieldNumber := ""
		if len(row) > 0 {
			fieldNumber = strings.TrimSpace(row[0])
		}

		// 获取字段名
		fieldName := ""
		if fieldNameIndex, exists := colIndex["字段名"]; exists && fieldNameIndex < len(row) {
			fieldName = strings.TrimSpace(row[fieldNameIndex])
		}
		if fieldName == "" {
			continue
		}

		// 建立字段编号到索引的映射
		if fieldNumber != "" {
			sheetConfig.FieldNumberMap[fieldNumber] = len(sheetConfig.FieldRules)
		}

		fieldRule := FieldRule{
			FieldName: fieldName,
		}

		// 获取属性（支持复杂规则）
		if attrIndex, exists := colIndex["属性"]; exists && attrIndex < len(row) {
			required := strings.TrimSpace(row[attrIndex])

			// 解析复杂规则
			if strings.Contains(required, "|") {
				// 分离基础属性和规则
				parts := strings.Split(required, "|")
				if len(parts) >= 2 {
					fieldRule.Required = strings.TrimSpace(parts[0])

					// 解析规则部分
					rulePart := strings.TrimSpace(parts[1])
					parseComplexRules(&fieldRule, rulePart)
				}
			} else {
				// 简单规则：必填、选填、空
				if required == "必填" || required == "选填" || required == "空" {
					fieldRule.Required = required
				}
			}
		}

		// 获取类型
		if typeIndex, exists := colIndex["类型"]; exists && typeIndex < len(row) {
			fieldType := strings.TrimSpace(row[typeIndex])
			if fieldType != "" && fieldType != "NaN" {
				fieldRule.Type = fieldType
			}
		}

		// 获取校验规则
		if rulesIndex, exists := colIndex["校验规则"]; exists && rulesIndex < len(row) {
			rulesStr := strings.TrimSpace(row[rulesIndex])
			if rulesStr != "" && rulesStr != "NaN" {
				// 解析校验规则
				fieldRule.Rules = parseRules(rulesStr)
			}
		}

		// 获取配置项
		if configItemIndex, exists := colIndex["配置项"]; exists && configItemIndex < len(row) {
			configItem := strings.TrimSpace(row[configItemIndex])
			if configItem != "" && configItem != "NaN" {
				fieldRule.ConfigItem = configItem
			}
		}

		// 处理配置项覆盖：如果配置项存在且不为空，则使用配置项的值替换规则
		if fieldRule.ConfigItem != "" && cfg != nil {
			configValue := config.GetConfigValue(cfg, fieldRule.ConfigItem)
			if configValue != "" {
				originalRules := strings.Join(fieldRule.Rules, ";")
				log.Printf("[配置项替换] Sheet: %s | 字段: %s | 校验规则: %s => 配置内容: %s",
					sheetName, fieldRule.FieldName, originalRules, configValue)
				fieldRule.Rules = parseRules(configValue)
				NewRules := strings.Join(fieldRule.Rules, ";")
				log.Printf("[生效的校验规则] Sheet: %s | 字段: %s | 校验规则: %s",
					sheetName, fieldRule.FieldName, NewRules)
			}
		}

		sheetConfig.FieldRules = append(sheetConfig.FieldRules, fieldRule)
	}

	return sheetConfig, nil
}

func parseRules(ruleStr string) []string {
	var rules []string

	// 分割多个规则
	rules = strings.Split(ruleStr, ";")

	// 清理规则
	for i := range rules {
		rules[i] = strings.TrimSpace(rules[i])
		// 移除空规则
		if rules[i] == "" {
			rules = append(rules[:i], rules[i+1:]...)
		}
	}

	return rules
}

// 合并文件校验配置和字段规则配置
func mergeSheetConfigs(fileConfigs, sheetConfigs []SheetConfig) []SheetConfig {
	var mergedConfigs []SheetConfig

	// 创建映射表，按sheet名称索引配置
	fileConfigMap := make(map[string]SheetConfig)
	sheetConfigMap := make(map[string]SheetConfig)

	// 填充文件校验配置映射
	for _, config := range fileConfigs {
		fileConfigMap[config.SheetName] = config
	}

	// 填充字段规则配置映射
	for _, config := range sheetConfigs {
		sheetConfigMap[config.SheetName] = config
	}

	// 合并所有sheet名称
	allSheetNames := make(map[string]bool)
	for name := range fileConfigMap {
		allSheetNames[name] = true
	}
	for name := range sheetConfigMap {
		allSheetNames[name] = true
	}

	// 为每个sheet创建合并后的配置
	for sheetName := range allSheetNames {
		mergedConfig := SheetConfig{
			SheetName: sheetName,
		}

		// 合并文件校验配置
		if fileConfig, exists := fileConfigMap[sheetName]; exists {
			mergedConfig.FileValidation = fileConfig.FileValidation
		}

		// 合并字段规则配置
		if sheetConfig, exists := sheetConfigMap[sheetName]; exists {
			mergedConfig.FieldRules = sheetConfig.FieldRules
			mergedConfig.FieldNumberMap = sheetConfig.FieldNumberMap
		}

		mergedConfigs = append(mergedConfigs, mergedConfig)
	}

	return mergedConfigs
}

// 获取有序的工作表列表
func getOrderedSheetList(xlsx *excelize.File) []string {
	var orderedSheets []string

	// 获取工作表映射
	sheetMap := xlsx.GetSheetMap()

	// 由于GetSheetMap返回的是无序映射，我们需要按索引顺序获取工作表
	// 创建一个索引到工作表名称的映射
	indexToName := make(map[int]string)
	maxIndex := 0

	for index, name := range sheetMap {
		indexToName[index] = name
		if index > maxIndex {
			maxIndex = index
		}
	}

	// 按索引顺序获取工作表名称
	for i := 1; i <= maxIndex; i++ {
		if name, exists := indexToName[i]; exists {
			orderedSheets = append(orderedSheets, name)
		}
	}

	return orderedSheets
}

// 解析文件类型配置
func ParseFileTypeConfig(xlsx *excelize.File, sheetName string) ([]string, string, string, string, error) {
	rows := xlsx.GetRows(sheetName)
	if rows == nil {
		return nil, "", "", "", fmt.Errorf("无法获取工作表%s的行数据", sheetName)
	}

	if len(rows) < 2 {
		return nil, "", "", "", fmt.Errorf("工作表%s配置不完整", sheetName)
	}

	// 假设配置在特定的行和列
	var headers []string
	var suffix, sizeLimit, checkContent string

	// 这里需要根据实际的Excel结构来解析
	// 简化实现，实际应该根据具体的列位置来解析
	for _, row := range rows {
		if len(row) >= 4 {
			if strings.Contains(row[0], "文件头") || strings.Contains(row[0], "header") {
				headers = strings.Split(strings.TrimSpace(row[1]), ";")
			}
			if strings.Contains(row[0], "后缀") || strings.Contains(row[0], "suffix") {
				suffix = strings.TrimSpace(row[1])
			}
			if strings.Contains(row[0], "大小") || strings.Contains(row[0], "size") {
				sizeLimit = strings.TrimSpace(row[1])
			}
			if strings.Contains(row[0], "内容") || strings.Contains(row[0], "content") {
				checkContent = strings.TrimSpace(row[1])
			}
		}
	}

	return headers, suffix, sizeLimit, checkContent, nil
}

// PreParseRules 预解析所有规则，避免在运行时重复 Split
func PreParseRules(configs []SheetConfig) {
	for i := range configs {
		for j := range configs[i].FieldRules {
			preParseFieldRule(&configs[i].FieldRules[j], configs[i].FieldNumberMap)
		}
	}
}

// preParseFieldRule 预解析单个字段规则
func preParseFieldRule(fr *FieldRule, fieldNumberMap map[string]int) {
	// 预解析枚举规则
	for _, rule := range fr.Rules {
		if strings.HasPrefix(rule, "[") && strings.HasSuffix(rule, "]") {
			if fr.ParsedEnums == nil {
				fr.ParsedEnums = make(map[string]*ParsedEnumValue)
			}
			fr.ParsedEnums[rule] = parseEnumRule(rule)
		}
		// 处理复合规则中的枚举
		if strings.Contains(rule, ";") {
			subRules := strings.Split(rule, ";")
			for _, sr := range subRules {
				sr = strings.TrimSpace(sr)
				if strings.HasPrefix(sr, "[") && strings.HasSuffix(sr, "]") {
					if fr.ParsedEnums == nil {
						fr.ParsedEnums = make(map[string]*ParsedEnumValue)
					}
					if _, exists := fr.ParsedEnums[sr]; !exists {
						fr.ParsedEnums[sr] = parseEnumRule(sr)
					}
				}
			}
		}
	}

	// 预解析条件表达式
	if fr.Condition != "" {
		fr.ParsedCondition = parseConditionRule(fr.Condition, fieldNumberMap)
	}
}

// parseEnumRule 预解析枚举规则
func parseEnumRule(rule string) *ParsedEnumValue {
	inner := strings.Trim(rule, "[]")
	parts := strings.Split(inner, ",")

	result := &ParsedEnumValue{
		ExactValues: make(map[string]struct{}),
		RawRule:     inner,
	}

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// 检查是否为范围格式
		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) == 2 {
				min, err1 := parseInt64(strings.TrimSpace(rangeParts[0]))
				max, err2 := parseInt64(strings.TrimSpace(rangeParts[1]))
				if err1 == nil && err2 == nil {
					result.Ranges = append(result.Ranges, EnumRange{Min: min, Max: max})
					continue
				}
			}
		}

		// 精确值
		result.ExactValues[part] = struct{}{}
	}

	return result
}

// parseConditionRule 预解析条件表达式
func parseConditionRule(condition string, fieldNumberMap map[string]int) *ParsedCondition {
	if !strings.HasPrefix(condition, "if(") || !strings.HasSuffix(condition, ")") {
		return nil
	}

	content := strings.TrimPrefix(condition, "if(")
	content = strings.TrimSuffix(content, ")")

	isEqual := true
	var parts []string

	if strings.Contains(content, "!=") {
		isEqual = false
		parts = strings.Split(content, "!=")
	} else if strings.Contains(content, "==") {
		isEqual = true
		parts = strings.Split(content, "==")
	} else {
		return nil
	}

	if len(parts) != 2 {
		return nil
	}

	fieldRef := strings.TrimSpace(parts[0])
	expectedStr := strings.TrimSpace(parts[1])

	// 解析字段索引
	fieldIndex := -1
	if strings.HasPrefix(fieldRef, "$") {
		fieldNumberStr := strings.TrimPrefix(fieldRef, "$")
		if fieldNumberMap != nil {
			if idx, exists := fieldNumberMap[fieldNumberStr]; exists {
				fieldIndex = idx
			}
		}
		if fieldIndex == -1 {
			if num, err := strconv.Atoi(fieldNumberStr); err == nil {
				fieldIndex = num - 1
			}
		}
	}

	// 解析期望值列表
	expectedValues := make(map[string]struct{})
	var expectedOrder []string
	for _, v := range strings.Split(expectedStr, ",") {
		v = strings.TrimSpace(v)
		if strings.HasPrefix(v, "\"") && strings.HasSuffix(v, "\"") {
			v = strings.Trim(v, "\"")
		}
		expectedValues[v] = struct{}{}
		expectedOrder = append(expectedOrder, v)
	}

	return &ParsedCondition{
		FieldIndex:    fieldIndex,
		ExpectedExact: expectedValues,
		ExpectedOrder: expectedOrder,
		IsEqual:       isEqual,
	}
}

func parseInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

// parseComplexRules 解析复杂的条件规则
func parseComplexRules(fieldRule *FieldRule, rulePart string) {
	// 使用分号分隔多个规则
	rules := strings.Split(rulePart, ";")

	for _, rule := range rules {
		rule = strings.TrimSpace(rule)
		if rule == "" {
			continue
		}

		// 解析条件规则
		if strings.HasPrefix(rule, "if(") && strings.HasSuffix(rule, ")") {
			fieldRule.Condition = rule
		}

		// 解析偏移规则
		if strings.HasPrefix(rule, "offset(") && strings.HasSuffix(rule, ")") {
			fieldRule.Offset = rule
		}

		// 解析数组规则
		if strings.HasPrefix(rule, "array(") && strings.HasSuffix(rule, ")") {
			fieldRule.Array = rule
		}

		// 解析循环规则
		if strings.HasPrefix(rule, "loop(") && strings.HasSuffix(rule, ")") {
			fieldRule.Loop = rule
		}

		// 解析跳转规则
		if strings.HasPrefix(rule, "jump=") {
			fieldRule.Jump = rule
		}

		// 解析正则规则
		if strings.HasPrefix(rule, "reg=") {
			fieldRule.Regex = strings.TrimPrefix(rule, "reg=")
		}

		// 解析类型规则
		if strings.HasPrefix(rule, "type=") {
			// 如果已经有类型，则合并
			if fieldRule.Type != "" {
				fieldRule.Type += "," + strings.TrimPrefix(rule, "type=")
			} else {
				fieldRule.Type = strings.TrimPrefix(rule, "type=")
			}
		}
	}
}
