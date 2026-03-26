package validator

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type RuleValidator struct {
	FieldValue string
	FieldIndex int
	AllFields  []string
}

func NewRuleValidator(fieldValue string, fieldIndex int, allFields []string) *RuleValidator {
	return &RuleValidator{
		FieldValue: fieldValue,
		FieldIndex: fieldIndex,
		AllFields:  allFields,
	}
}

// 校验类型主函数
func (rv *RuleValidator) ValidateType(dataType string) (bool, string) {
	dataType = strings.TrimSpace(dataType)

	switch dataType {
	case "int":
		return rv.validateInteger()
	case "ip", "ipv4":
		return rv.validateIPv4()
	case "ipv6":
		return rv.validateIPv6()
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
	case strings.HasPrefix(rule, "len="):
		return rv.validateLengthEqual(rule)
	case strings.HasPrefix(rule, "len>"):
		return rv.validateLengthGreater(rule)
	case strings.HasPrefix(rule, "len<"):
		return rv.validateLengthLess(rule)
	case strings.HasPrefix(rule, "len>="):
		return rv.validateLengthGreaterEqual(rule)
	case strings.HasPrefix(rule, "len<="):
		return rv.validateLengthLessEqual(rule)
	case strings.HasPrefix(rule, "size="):
		return rv.validateSizeEqual(rule)
	case strings.HasPrefix(rule, "size>"):
		return rv.validateSizeGreater(rule)
	case strings.HasPrefix(rule, "size<"):
		return rv.validateSizeLess(rule)
	case strings.HasPrefix(rule, "size>="):
		return rv.validateSizeGreaterEqual(rule)
	case strings.HasPrefix(rule, "size<="):
		return rv.validateSizeLessEqual(rule)
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
	re, err := regexp.Compile(pattern)
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
	case "ipv4":
		if !IsIPv4(rv.FieldValue) {
			return false, "不是有效的IPv4地址"
		}
	case "ipv6":
		if !IsIPv6(rv.FieldValue) {
			return false, "不是有效的IPv6地址"
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
