package validator

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"xdrCheck/internal/parser"
)

type RuleValidator struct {
	FieldValue     string
	FieldIndex     int
	AllFields      []string
	FieldNumberMap map[string]int // 字段编号到索引的映射（如 "11" -> 索引）
	ParsedEnum     *parser.ParsedEnumValue     // 预解析的枚举规则缓存
	ParsedCond     *parser.ParsedCondition     // 预解析的条件规则缓存

	// 数值解析缓存
	parsedIntValue  int64
	parsedIntValid  bool
	parsedIntDone   bool
}

func NewRuleValidator(fieldValue string, fieldIndex int, allFields []string, fieldNumberMap map[string]int) *RuleValidator {
	return &RuleValidator{
		FieldValue:     fieldValue,
		FieldIndex:     fieldIndex,
		AllFields:      allFields,
		FieldNumberMap: fieldNumberMap,
	}
}

// NewRuleValidatorWithCache 创建带预解析缓存的验证器
func NewRuleValidatorWithCache(fieldValue string, fieldIndex int, allFields []string, fieldNumberMap map[string]int, parsedEnum *parser.ParsedEnumValue, parsedCond *parser.ParsedCondition) *RuleValidator {
	return &RuleValidator{
		FieldValue:     fieldValue,
		FieldIndex:     fieldIndex,
		AllFields:      allFields,
		FieldNumberMap: fieldNumberMap,
		ParsedEnum:     parsedEnum,
		ParsedCond:     parsedCond,
	}
}

// Reset 重置验证器状态，复用已分配的对象避免 GC 压力
// allFields 和 fieldNumberMap 通常在同一行内不变，可以保留
func (rv *RuleValidator) Reset(fieldValue string, fieldIndex int, parsedEnum *parser.ParsedEnumValue, parsedCond *parser.ParsedCondition) {
	rv.FieldValue = fieldValue
	rv.FieldIndex = fieldIndex
	rv.ParsedEnum = parsedEnum
	rv.ParsedCond = parsedCond
	// 重置数值缓存
	rv.parsedIntDone = false
	rv.parsedIntValid = false
	rv.parsedIntValue = 0
}

// getCachedIntValue 获取缓存中字段值的整数解析结果
func (rv *RuleValidator) getCachedIntValue() (int64, bool) {
	if !rv.parsedIntDone {
		var err error
		rv.parsedIntValue, err = strconv.ParseInt(rv.FieldValue, 10, 64)
		rv.parsedIntValid = (err == nil)
		rv.parsedIntDone = true
	}
	return rv.parsedIntValue, rv.parsedIntValid
}

// 校验类型主函数
func (rv *RuleValidator) ValidateType(dataType string) (bool, string) {
	dataType = strings.TrimSpace(dataType)

	// 处理逗号分隔的复合类型（如 "ipv4,ipv6"）
	if strings.Contains(dataType, ",") {
		types := strings.Split(dataType, ",")
		for _, t := range types {
			t = strings.TrimSpace(t)
			if valid, _ := rv.ValidateType(t); valid {
				return true, ""
			}
		}
		return false, fmt.Sprintf("不符合%s格式", dataType)
	}

	switch dataType {
	case "int":
		return rv.validateInteger()
	case "ip":
		// ip类型同时支持IPv4和IPv6
		if IsIPv4(rv.FieldValue) || IsIPv6(rv.FieldValue) {
			return true, ""
		}
		return false, "不是有效的IP地址（IPv4或IPv6）"
	case "ipv4":
		return rv.validateIPv4()
	case "ipv6":
		return rv.validateIPv6()
	case "ip_compressed":
		if IsIPv4(rv.FieldValue) || IsIPv6Compressed(rv.FieldValue) {
			return true, ""
		}
		return false, "不是有效的ip_compressed格式(IPv4或IPv6压缩格式)"
	case "ip_exploded":
		if IsIPv4(rv.FieldValue) || IsIPv6Exploded(rv.FieldValue) {
			return true, ""
		}
		return false, "不是有效的ip_exploded格式(IPv4或IPv6展开格式)"
	case "datetime":
		return rv.validateDateTime()
	case "base64":
		return rv.validateBase64()
	case "base64_json":
		return rv.validateBase64JSON()
	case "json":
		return rv.validateJSON()
	default:
		return true, "" // 未知类型，跳过校验
	}
}

// 校验规则主函数
func (rv *RuleValidator) ValidateRule(rule string) (bool, string) {
	rule = strings.TrimSpace(rule)

	// 处理复合规则
	if strings.Contains(rule, ";") {
		rules := strings.Split(rule, ";")
		for _, r := range rules {
			if r == "" {
				continue
			}
			valid, msg := rv.validateSingleRule(strings.TrimSpace(r))
			if !valid {
				return false, msg
			}
		}
		return true, ""
	}

	return rv.validateSingleRule(rule)
}

// 整数类型校验
func (rv *RuleValidator) validateInteger() (bool, string) {
	if rv.FieldValue == "" {
		return true, "" // 空值跳过校验
	}

	_, valid := rv.getCachedIntValue()
	if !valid {
		return false, "不是有效的整数"
	}
	return true, ""
}

// IPv4地址校验
func (rv *RuleValidator) validateIPv4() (bool, string) {
	if rv.FieldValue == "" {
		return true, "" // 空值跳过校验
	}

	if !IsIPv4(rv.FieldValue) {
		return false, "不是有效的IPv4地址"
	}
	return true, ""
}

// IPv6地址校验
func (rv *RuleValidator) validateIPv6() (bool, string) {
	if rv.FieldValue == "" {
		return true, "" // 空值跳过校验
	}

	if !IsIPv6(rv.FieldValue) {
		return false, "不是有效的IPv6地址"
	}
	return true, ""
}

// 日期时间校验 (yyyy-MM-dd HH:mm:ss)
func (rv *RuleValidator) validateDateTime() (bool, string) {
	if rv.FieldValue == "" {
		return true, "" // 空值跳过校验
	}

	_, err := time.Parse("2006-01-02 15:04:05", rv.FieldValue)
	if err != nil {
		return false, "不是有效的日期时间格式 (yyyy-MM-dd HH:mm:ss)"
	}
	return true, ""
}

// Base64编码校验
// 支持逗号分隔的多个base64字符串
func (rv *RuleValidator) validateBase64() (bool, string) {
	if rv.FieldValue == "" {
		return true, "" // 空值跳过校验
	}

	// 检查是否包含逗号（多个base64值）
	if strings.Contains(rv.FieldValue, ",") {
		parts := strings.Split(rv.FieldValue, ",")
		for i, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if _, err := base64.StdEncoding.DecodeString(part); err != nil {
				return false, fmt.Sprintf("第%d个base64值不是有效的Base64编码", i+1)
			}
		}
		return true, ""
	}

	// 单个base64值
	_, err := base64.StdEncoding.DecodeString(rv.FieldValue)
	if err != nil {
		return false, "不是有效的Base64编码"
	}
	return true, ""
}

// Base64编码的JSON校验
func (rv *RuleValidator) validateBase64JSON() (bool, string) {
	if rv.FieldValue == "" {
		return true, "" // 空值跳过校验
	}

	decoded, err := base64.StdEncoding.DecodeString(rv.FieldValue)
	if err != nil {
		return false, "不是有效的Base64编码"
	}

	var jsonData interface{}
	if err := json.Unmarshal(decoded, &jsonData); err != nil {
		return false, "Base64解码后不是有效的JSON格式"
	}
	return true, ""
}

// JSON格式校验
func (rv *RuleValidator) validateJSON() (bool, string) {
	if rv.FieldValue == "" {
		return true, "" // 空值跳过校验
	}

	var jsonData interface{}
	if err := json.Unmarshal([]byte(rv.FieldValue), &jsonData); err != nil {
		return false, "不是有效的JSON格式"
	}
	return true, ""
}

// 校验单个规则
func (rv *RuleValidator) validateSingleRule(rule string) (bool, string) {
	// 标准化规则（去除多余空格）
	rule = strings.ReplaceAll(rule, " ", "")

	switch {
	case strings.HasPrefix(rule, "len>="):
		return rv.validateLengthGreaterEqual(rule)
	case strings.HasPrefix(rule, "len<="):
		return rv.validateLengthLessEqual(rule)
	case strings.HasPrefix(rule, "len="):
		return rv.validateLengthEqual(rule)
	case strings.HasPrefix(rule, "len>"):
		return rv.validateLengthGreater(rule)
	case strings.HasPrefix(rule, "len<"):
		return rv.validateLengthLess(rule)
	case strings.HasPrefix(rule, "size>="):
		return rv.validateSizeGreaterEqual(rule)
	case strings.HasPrefix(rule, "size<="):
		return rv.validateSizeLessEqual(rule)
	case strings.HasPrefix(rule, "size="):
		return rv.validateSizeEqual(rule)
	case strings.HasPrefix(rule, "size>"):
		return rv.validateSizeGreater(rule)
	case strings.HasPrefix(rule, "size<"):
		return rv.validateSizeLess(rule)
	case strings.HasPrefix(rule, "reg="):
		return rv.validateRegex(rule)
	case strings.HasPrefix(rule, "json_field="):
		return rv.validateJSONField(rule)
	case strings.HasPrefix(rule, "[") && strings.HasSuffix(rule, "]"):
		return rv.validateEnum(rule)
	case strings.Contains(rule, "-"):
		return rv.validateRange(rule)
	default:
		return rv.validateBasicRule(rule)
	}
}

// 长度校验
func (rv *RuleValidator) validateLengthEqual(rule string) (bool, string) {
	expectedLen, err := strconv.Atoi(strings.TrimPrefix(rule, "len="))
	if err != nil {
		return false, "长度规则格式错误"
	}

	if len(rv.FieldValue) != expectedLen {
		return false, fmt.Sprintf("长度应为%d，实际为%d", expectedLen, len(rv.FieldValue))
	}
	return true, ""
}

func (rv *RuleValidator) validateLengthGreater(rule string) (bool, string) {
	minLen, err := strconv.Atoi(strings.TrimPrefix(rule, "len>"))
	if err != nil {
		return false, "长度规则格式错误"
	}

	if len(rv.FieldValue) <= minLen {
		return false, fmt.Sprintf("长度应大于%d，实际为%d", minLen, len(rv.FieldValue))
	}
	return true, ""
}

func (rv *RuleValidator) validateLengthLess(rule string) (bool, string) {
	maxLen, err := strconv.Atoi(strings.TrimPrefix(rule, "len<"))
	if err != nil {
		return false, "长度规则格式错误"
	}

	if len(rv.FieldValue) >= maxLen {
		return false, fmt.Sprintf("长度应小于%d，实际为%d", maxLen, len(rv.FieldValue))
	}
	return true, ""
}

func (rv *RuleValidator) validateLengthGreaterEqual(rule string) (bool, string) {
	minLen, err := strconv.Atoi(strings.TrimPrefix(rule, "len>="))
	if err != nil {
		return false, "长度规则格式错误"
	}

	if len(rv.FieldValue) < minLen {
		return false, fmt.Sprintf("长度应大于等于%d，实际为%d", minLen, len(rv.FieldValue))
	}
	return true, ""
}

func (rv *RuleValidator) validateLengthLessEqual(rule string) (bool, string) {
	maxLen, err := strconv.Atoi(strings.TrimPrefix(rule, "len<="))
	if err != nil {
		return false, "长度规则格式错误"
	}

	if len(rv.FieldValue) > maxLen {
		return false, fmt.Sprintf("长度应小于等于%d，实际为%d", maxLen, len(rv.FieldValue))
	}
	return true, ""
}

// 大小校验（用于数字）
func (rv *RuleValidator) validateSizeEqual(rule string) (bool, string) {
	expectedSize, err := strconv.ParseInt(strings.TrimPrefix(rule, "size="), 10, 64)
	if err != nil {
		return false, "大小规则格式错误"
	}

	actualSize, valid := rv.getCachedIntValue()
	if !valid {
		return false, "字段值不是有效数字"
	}

	if actualSize != expectedSize {
		return false, fmt.Sprintf("大小应为%d，实际为%d", expectedSize, actualSize)
	}
	return true, ""
}

func (rv *RuleValidator) validateSizeGreater(rule string) (bool, string) {
	minSize, err := strconv.ParseInt(strings.TrimPrefix(rule, "size>"), 10, 64)
	if err != nil {
		return false, "大小规则格式错误"
	}

	actualSize, valid := rv.getCachedIntValue()
	if !valid {
		return false, "字段值不是有效数字"
	}

	if actualSize <= minSize {
		return false, fmt.Sprintf("大小应大于%d，实际为%d", minSize, actualSize)
	}
	return true, ""
}

func (rv *RuleValidator) validateSizeLess(rule string) (bool, string) {
	maxSize, err := strconv.ParseInt(strings.TrimPrefix(rule, "size<"), 10, 64)
	if err != nil {
		return false, "大小规则格式错误"
	}

	actualSize, valid := rv.getCachedIntValue()
	if !valid {
		return false, "字段值不是有效数字"
	}

	if actualSize >= maxSize {
		return false, fmt.Sprintf("大小应小于%d，实际为%d", maxSize, actualSize)
	}
	return true, ""
}

func (rv *RuleValidator) validateSizeGreaterEqual(rule string) (bool, string) {
	minSize, err := strconv.ParseInt(strings.TrimPrefix(rule, "size>="), 10, 64)
	if err != nil {
		return false, "大小规则格式错误"
	}

	actualSize, valid := rv.getCachedIntValue()
	if !valid {
		return false, "字段值不是有效数字"
	}

	if actualSize < minSize {
		return false, fmt.Sprintf("大小应大于等于%d，实际为%d", minSize, actualSize)
	}
	return true, ""
}

func (rv *RuleValidator) validateSizeLessEqual(rule string) (bool, string) {
	maxSize, err := strconv.ParseInt(strings.TrimPrefix(rule, "size<="), 10, 64)
	if err != nil {
		return false, "大小规则格式错误"
	}

	actualSize, valid := rv.getCachedIntValue()
	if !valid {
		return false, "字段值不是有效数字"
	}

	if actualSize > maxSize {
		return false, fmt.Sprintf("大小应小于等于%d，实际为%d", maxSize, actualSize)
	}
	return true, ""
}

// 正则表达式校验
func (rv *RuleValidator) validateRegex(rule string) (bool, string) {
	pattern := strings.TrimPrefix(rule, "reg=")
	re, err := GetRegex(pattern)
	if err != nil {
		return false, "正则表达式格式错误"
	}

	if !re.MatchString(rv.FieldValue) {
		return false, "字段值不符合正则表达式规则"
	}
	return true, ""
}

// JSON字段校验
func (rv *RuleValidator) validateJSONField(rule string) (bool, string) {
	// 简化实现，实际应该解析JSON
	if rv.FieldValue == "" {
		return true, ""
	}

	// 检查是否为有效JSON
	if !strings.Contains(rv.FieldValue, "{") && !strings.Contains(rv.FieldValue, "}") {
		return false, "字段值不是有效的JSON格式"
	}

	return true, ""
}

// 范围校验
func (rv *RuleValidator) validateRange(rule string) (bool, string) {
	// 检查是否为枚举格式 [a,b,c]
	if strings.HasPrefix(rule, "[") && strings.HasSuffix(rule, "]") {
		return rv.validateEnum(rule)
	}

	// 检查是否为数值范围格式 min-max
	parts := strings.Split(rule, "-")
	if len(parts) != 2 {
		return false, "范围规则格式错误"
	}

	min, err1 := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
	max, err2 := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)

	if err1 != nil || err2 != nil {
		return false, "范围规则格式错误"
	}

	value, valid := rv.getCachedIntValue()
	if !valid {
		return false, "字段值不是有效数字"
	}

	if value < min || value > max {
		return false, fmt.Sprintf("值应在%d-%d范围内，实际为%d", min, max, value)
	}

	return true, ""
}

// 枚举校验
func (rv *RuleValidator) validateEnum(rule string) (bool, string) {
	// 优先使用预解析缓存
	if rv.ParsedEnum != nil {
		return rv.validateEnumParsed(rv.ParsedEnum)
	}

	// 回退到原始逻辑（兼容旧调用方式）
	rule = strings.Trim(rule, "[]")
	validValues := strings.Split(rule, ",")

	for _, validValue := range validValues {
		validValue = strings.TrimSpace(validValue)

		// 检查是否为范围格式 (如 "0-5")
		if strings.Contains(validValue, "-") {
			parts := strings.Split(validValue, "-")
			if len(parts) == 2 {
				min, err1 := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
				max, err2 := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)

				if err1 == nil && err2 == nil {
					fieldValue, err := strconv.ParseInt(rv.FieldValue, 10, 64)
					if err == nil {
						if fieldValue >= min && fieldValue <= max {
							return true, ""
						}
					}
				}
			}
		} else {
			// 简单值匹配
			if validValue == rv.FieldValue {
				return true, ""
			}
		}
	}

	return false, fmt.Sprintf("字段值不在允许的范围内: %s", rule)
}

// validateEnumParsed 使用预解析结果进行枚举校验（零Split开销）
func (rv *RuleValidator) validateEnumParsed(pe *parser.ParsedEnumValue) (bool, string) {
	// 先检查精确匹配
	if _, ok := pe.ExactValues[rv.FieldValue]; ok {
		return true, ""
	}

	// 再检查范围匹配
	if len(pe.Ranges) > 0 {
		fieldValue, err := strconv.ParseInt(rv.FieldValue, 10, 64)
		if err == nil {
			for _, r := range pe.Ranges {
				if fieldValue >= r.Min && fieldValue <= r.Max {
					return true, ""
				}
			}
		}
	}

	return false, fmt.Sprintf("字段值不在允许的范围内: %s", pe.RawRule)
}

// 基础规则校验
func (rv *RuleValidator) validateBasicRule(rule string) (bool, string) {
	switch rule {
	case "ip":
		if !IsIPv4(rv.FieldValue) && !IsIPv6(rv.FieldValue) {
			return false, "不是有效的IP地址（IPv4或IPv6）"
		}
	case "ipv4":
		if !IsIPv4(rv.FieldValue) {
			return false, "不是有效的IPv4地址"
		}
	case "ipv6":
		if !IsIPv6(rv.FieldValue) {
			return false, "不是有效的IPv6地址"
		}
	case "ip_compressed":
		if !IsIPv4(rv.FieldValue) && !IsIPv6Compressed(rv.FieldValue) {
			return false, "不是有效的ip_compressed格式(IPv4或IPv6压缩格式)"
		}
	case "ip_exploded":
		if !IsIPv4(rv.FieldValue) && !IsIPv6Exploded(rv.FieldValue) {
			return false, "不是有效的ip_exploded格式(IPv4或IPv6展开格式)"
		}
	case "base64":
		if _, err := base64.StdEncoding.DecodeString(rv.FieldValue); err != nil {
			return false, "不是有效的Base64编码"
		}
	case "datetime":
		if _, err := time.Parse("2006-01-02 15:04:05", rv.FieldValue); err != nil {
			return false, "不是有效的时间格式"
		}
	default:
		// 默认情况下，如果规则不为空但字段值为空，则校验失败
		if rule != "" && rv.FieldValue == "" {
			return false, "字段值为空"
		}
	}

	return true, ""
}

// ValidateCondition 验证条件表达式
func (rv *RuleValidator) ValidateCondition(condition string) (bool, string) {
	if condition == "" {
		return true, ""
	}

	// 优先使用预解析缓存
	if rv.ParsedCond != nil {
		return rv.validateConditionParsed(rv.ParsedCond)
	}

	// 回退到原始逻辑（兼容旧调用方式）
	// 解析条件表达式
	if !strings.HasPrefix(condition, "if(") || !strings.HasSuffix(condition, ")") {
		return false, "条件表达式格式错误"
	}

	// 提取条件内容
	condContent := strings.TrimPrefix(condition, "if(")
	condContent = strings.TrimSuffix(condContent, ")")

	// 解析条件
	if strings.Contains(condContent, "==") {
		return rv.validateEqualCondition(condContent)
	} else if strings.Contains(condContent, "!=") {
		return rv.validateNotEqualCondition(condContent)
	}

	return false, "不支持的比较操作符"
}

// validateConditionParsed 使用预解析结果进行条件校验（零Split开销）
func (rv *RuleValidator) validateConditionParsed(pc *parser.ParsedCondition) (bool, string) {
	// 检查字段索引是否有效
	if pc.FieldIndex < 0 || pc.FieldIndex >= len(rv.AllFields) {
		return false, ""
	}

	actualValue := strings.TrimSpace(rv.AllFields[pc.FieldIndex])

	if pc.IsEqual {
		// == 条件：值在期望列表中则条件满足
		_, found := pc.ExpectedExact[actualValue]
		return found, ""
	}

	// != 条件：值不在期望列表中则条件满足
	_, found := pc.ExpectedExact[actualValue]
	return !found, ""
}

// validateEqualCondition 验证等于条件
// 返回值说明：
//   - true: 条件满足（字段值等于期望值之一），需要继续校验当前字段
//   - false: 条件不满足（字段值不等于任何期望值），不需要校验当前字段
func (rv *RuleValidator) validateEqualCondition(condContent string) (bool, string) {
	parts := strings.Split(condContent, "==")
	if len(parts) != 2 {
		return false, "等于条件格式错误"
	}

	fieldRef := strings.TrimSpace(parts[0])
	expectedValues := strings.TrimSpace(parts[1])

	// 解析字段引用（如 $13）
	fieldIndex, _, err := rv.parseFieldReference(fieldRef)
	if err != nil {
		return false, fmt.Sprintf("字段引用错误: %v", err)
	}

	// 检查字段索引是否有效
	if fieldIndex < 0 || fieldIndex >= len(rv.AllFields) {
		return false, "" // 字段索引超出范围，条件不满足
	}

	actualValue := strings.TrimSpace(rv.AllFields[fieldIndex])

	// 解析期望值（支持多个值，如 5,8）
	expectedList := strings.Split(expectedValues, ",")
	for _, expected := range expectedList {
		expected = strings.TrimSpace(expected)
		// 处理字符串值（带引号的情况）
		if strings.HasPrefix(expected, "\"") && strings.HasSuffix(expected, "\"") {
			expected = strings.Trim(expected, "\"")
		}

		if actualValue == expected {
			// 条件满足（字段值等于某个期望值）
			// 返回true表示"条件满足，需要校验当前字段"
			return true, ""
		}
	}

	// 条件不满足（字段值不等于任何期望值）
	// 返回false表示"条件不满足，不需要校验当前字段"
	return false, ""
}

// validateNotEqualCondition 验证不等于条件
// 返回值说明：
//   - true: 条件满足（字段值不等于期望值），需要继续校验当前字段
//   - false: 条件不满足（字段值等于期望值），不需要校验当前字段
func (rv *RuleValidator) validateNotEqualCondition(condContent string) (bool, string) {
	parts := strings.Split(condContent, "!=")
	if len(parts) != 2 {
		return false, "不等于条件格式错误"
	}

	fieldRef := strings.TrimSpace(parts[0])
	expectedValue := strings.TrimSpace(parts[1])

	// 解析字段引用
	fieldIndex, _, err := rv.parseFieldReference(fieldRef)
	if err != nil {
		return false, fmt.Sprintf("字段引用错误: %v", err)
	}

	// 检查字段索引是否有效
	if fieldIndex < 0 || fieldIndex >= len(rv.AllFields) {
		return false, "" // 字段索引超出范围，条件不满足
	}

	actualValue := strings.TrimSpace(rv.AllFields[fieldIndex])

	// 处理字符串值
	if strings.HasPrefix(expectedValue, "\"") && strings.HasSuffix(expectedValue, "\"") {
		expectedValue = strings.Trim(expectedValue, "\"")
	}

	if actualValue != expectedValue {
		// 条件满足（字段值确实不等于期望值）
		// 返回true表示"条件满足，需要校验当前字段"
		return true, ""
	}

	// 条件不满足（字段值等于期望值）
	// 返回false表示"条件不满足，不需要校验当前字段"
	return false, ""
}

// parseFieldReference 解析字段引用（如 $13）
// 返回字段索引和原始字段编号
func (rv *RuleValidator) parseFieldReference(fieldRef string) (int, string, error) {
	if !strings.HasPrefix(fieldRef, "$") {
		return 0, "", fmt.Errorf("字段引用必须以$开头")
	}

	fieldNumberStr := strings.TrimPrefix(fieldRef, "$")

	// 如果有字段编号映射，使用映射关系
	if rv.FieldNumberMap != nil {
		if fieldIndex, exists := rv.FieldNumberMap[fieldNumberStr]; exists {
			return fieldIndex, fieldNumberStr, nil
		}
		return 0, "", fmt.Errorf("字段编号%s在映射表中不存在", fieldNumberStr)
	}

	// 如果没有映射表，使用直接数字（向后兼容）
	// Excel字段编号从1开始，数组索引从0开始，需要减1
	fieldNumber, err := strconv.Atoi(fieldNumberStr)
	if err != nil {
		return 0, "", fmt.Errorf("字段索引不是有效数字")
	}

	return fieldNumber - 1, fieldNumberStr, nil
}

// ApplyOffset 应用偏移规则，从字段值中提取子串
// offsetRule 格式: "offset(6,4)" - 跳过6个字符，从第7个字符开始取4个字符
// 特殊处理: 去除前导零，如 "0014" → "14"，"1400" → "1400"
func ApplyOffset(value, offsetRule string) string {
	// 解析 offset(offset,length)
	inner := strings.TrimPrefix(offsetRule, "offset(")
	inner = strings.TrimSuffix(inner, ")")
	parts := strings.Split(inner, ",")
	if len(parts) != 2 {
		return value
	}

	offset, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	length, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil {
		return value
	}

	if offset >= len(value) {
		return ""
	}
	end := offset + length
	if end > len(value) {
		end = len(value)
	}
	result := value[offset:end]

	// 去除前导零: "0014" → "14", "1400" → "1400"
	stripped := strings.TrimLeft(result, "0")
	if stripped == "" {
		return "0"
	}
	return stripped
}

// ValidateConditionalType 根据条件映射校验字段类型
// typeRule: 逗号分隔的类型，如 "ipv4,ipv6"
// condition: 条件表达式，如 "if($12==1,2)"
// 条件值与类型按位置映射: 条件值1→类型1, 条件值2→类型2
// 示例: if($12==1,2);type=ipv4,ipv6 表示 $12==1时校验ipv4, $12==2时校验ipv6
func (rv *RuleValidator) ValidateConditionalType(typeRule, condition string) (bool, string) {
	types := strings.Split(typeRule, ",")

	// 尝试根据条件确定具体类型
	if condition != "" && rv.ParsedCond != nil {
		// 获取条件引用字段的实际值
		if rv.ParsedCond.FieldIndex >= 0 && rv.ParsedCond.FieldIndex < len(rv.AllFields) {
			actualValue := strings.TrimSpace(rv.AllFields[rv.ParsedCond.FieldIndex])

			// 查找匹配的期望值，确定对应的类型
			for i, expected := range rv.ParsedCond.ExpectedOrder {
				if actualValue == expected && i < len(types) {
					targetType := strings.TrimSpace(types[i])
					return rv.ValidateType(targetType)
				}
			}
		}
	}

	// 降级: 校验是否符合任意一种类型（OR逻辑）
	for _, t := range types {
		t = strings.TrimSpace(t)
		if valid, _ := rv.ValidateType(t); valid {
			return true, ""
		}
	}
	return false, fmt.Sprintf("不符合%s格式", typeRule)
}

// GetConditionMatchedValue 获取条件表达式中引用字段的实际值
// 用于条件类型映射等场景
func (rv *RuleValidator) GetConditionMatchedValue(condition string) string {
	if rv.ParsedCond != nil {
		if rv.ParsedCond.FieldIndex >= 0 && rv.ParsedCond.FieldIndex < len(rv.AllFields) {
			return strings.TrimSpace(rv.AllFields[rv.ParsedCond.FieldIndex])
		}
	}
	return ""
}
