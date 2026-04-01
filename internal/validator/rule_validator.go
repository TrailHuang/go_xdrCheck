package validator

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type RuleValidator struct {
	FieldValue     string
	FieldIndex     int
	AllFields      []string
	FieldNumberMap map[string]int // 字段编号到索引的映射（如 "11" -> 索引）
}

func NewRuleValidator(fieldValue string, fieldIndex int, allFields []string, fieldNumberMap map[string]int) *RuleValidator {
	return &RuleValidator{
		FieldValue:     fieldValue,
		FieldIndex:     fieldIndex,
		AllFields:      allFields,
		FieldNumberMap: fieldNumberMap,
	}
}

// 校验类型主函数
func (rv *RuleValidator) ValidateType(dataType string) (bool, string) {
	dataType = strings.TrimSpace(dataType)

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
		if IsIPv6Compressed(rv.FieldValue) {
			return true, ""
		}
		return false, "不是有效的IPv6压缩格式"
	case "ip_exploded":
		if IsIPv6Exploded(rv.FieldValue) {
			return true, ""
		}
		return false, "不是有效的IPv6展开格式"
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

	_, err := strconv.ParseInt(rv.FieldValue, 10, 64)
	if err != nil {
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
func (rv *RuleValidator) validateBase64() (bool, string) {
	if rv.FieldValue == "" {
		return true, "" // 空值跳过校验
	}

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

	actualSize, err := strconv.ParseInt(rv.FieldValue, 10, 64)
	if err != nil {
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

	actualSize, err := strconv.ParseInt(rv.FieldValue, 10, 64)
	if err != nil {
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

	actualSize, err := strconv.ParseInt(rv.FieldValue, 10, 64)
	if err != nil {
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

	actualSize, err := strconv.ParseInt(rv.FieldValue, 10, 64)
	if err != nil {
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

	actualSize, err := strconv.ParseInt(rv.FieldValue, 10, 64)
	if err != nil {
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

	value, err := strconv.ParseInt(rv.FieldValue, 10, 64)
	if err != nil {
		return false, "字段值不是有效数字"
	}

	if value < min || value > max {
		return false, fmt.Sprintf("值应在%d-%d范围内，实际为%d", min, max, value)
	}

	return true, ""
}

// 枚举校验
func (rv *RuleValidator) validateEnum(rule string) (bool, string) {
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
		if !IsIPv6Compressed(rv.FieldValue) {
			return false, "不是有效的IPv6压缩格式"
		}
	case "ip_exploded":
		if !IsIPv6Exploded(rv.FieldValue) {
			return false, "不是有效的IPv6展开格式"
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

// validateEqualCondition 验证等于条件
func (rv *RuleValidator) validateEqualCondition(condContent string) (bool, string) {
	parts := strings.Split(condContent, "==")
	if len(parts) != 2 {
		return false, "等于条件格式错误"
	}

	fieldRef := strings.TrimSpace(parts[0])
	expectedValues := strings.TrimSpace(parts[1])

	// 解析字段引用（如 $13）
	fieldIndex, fieldNumberStr, err := rv.parseFieldReference(fieldRef)
	if err != nil {
		return false, fmt.Sprintf("字段引用错误: %v", err)
	}

	// 检查字段索引是否有效
	if fieldIndex < 0 || fieldIndex >= len(rv.AllFields) {
		return true, "" // 字段索引超出范围，条件不满足
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
			// 条件满足，当前字段必须有值
			if rv.FieldValue == "" {
				return false, fmt.Sprintf("当字段%s等于%s时，此字段不能为空", fieldNumberStr, expected)
			}
			return true, ""
		}
	}

	// 条件不满足，不需要校验
	return true, ""
}

// validateNotEqualCondition 验证不等于条件
func (rv *RuleValidator) validateNotEqualCondition(condContent string) (bool, string) {
	parts := strings.Split(condContent, "!=")
	if len(parts) != 2 {
		return false, "不等于条件格式错误"
	}

	fieldRef := strings.TrimSpace(parts[0])
	expectedValue := strings.TrimSpace(parts[1])

	// 解析字段引用
	fieldIndex, fieldNumberStr, err := rv.parseFieldReference(fieldRef)
	if err != nil {
		return false, fmt.Sprintf("字段引用错误: %v", err)
	}

	// 检查字段索引是否有效
	if fieldIndex < 0 || fieldIndex >= len(rv.AllFields) {
		return true, "" // 字段索引超出范围，条件不满足
	}

	actualValue := strings.TrimSpace(rv.AllFields[fieldIndex])

	// 处理字符串值
	if strings.HasPrefix(expectedValue, "\"") && strings.HasSuffix(expectedValue, "\"") {
		expectedValue = strings.Trim(expectedValue, "\"")
	}

	if actualValue != expectedValue {
		// 条件满足，当前字段必须有值
		if rv.FieldValue == "" {
			return false, fmt.Sprintf("当字段%s不等于%s时，此字段不能为空", fieldNumberStr, expectedValue)
		}
		return true, ""
	}

	// 条件不满足，不需要校验
	return true, ""
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
