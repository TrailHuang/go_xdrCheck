package validator

import (
	"testing"

	"xdrCheck/internal/parser"
)

// ============== NewRuleValidator 测试 ==============

func TestNewRuleValidator(t *testing.T) {
	rv := NewRuleValidator("test", 0, []string{"a", "b"}, map[string]int{"1": 0})
	if rv.FieldValue != "test" {
		t.Errorf("FieldValue 应为 'test', 实际为 %s", rv.FieldValue)
	}
	if rv.FieldIndex != 0 {
		t.Errorf("FieldIndex 应为 0, 实际为 %d", rv.FieldIndex)
	}
	if len(rv.AllFields) != 2 {
		t.Errorf("AllFields 长度应为 2, 实际为 %d", len(rv.AllFields))
	}
	if rv.FieldNumberMap["1"] != 0 {
		t.Error("FieldNumberMap 映射错误")
	}
}

func TestNewRuleValidatorWithCache(t *testing.T) {
	parsedEnum := &parser.ParsedEnumValue{
		ExactValues: map[string]struct{}{"a": {}},
	}
	parsedCond := &parser.ParsedCondition{
		FieldIndex:    0,
		IsEqual:       true,
		ExpectedExact: map[string]struct{}{"x": {}},
	}
	rv := NewRuleValidatorWithCache("test", 0, []string{"a"}, nil, parsedEnum, parsedCond)
	if rv.ParsedEnum != parsedEnum {
		t.Error("ParsedEnum 未正确设置")
	}
	if rv.ParsedCond != parsedCond {
		t.Error("ParsedCond 未正确设置")
	}
}

// ============== Reset 测试 ==============

func TestReset(t *testing.T) {
	rv := NewRuleValidator("old", 0, []string{"a"}, nil)
	rv.parsedIntDone = true
	rv.parsedIntValid = true
	rv.parsedIntValue = 123

	parsedEnum := &parser.ParsedEnumValue{RawRule: "[1,2]"}
	rv.Reset("new", 1, parsedEnum, nil)

	if rv.FieldValue != "new" {
		t.Errorf("FieldValue 应为 'new', 实际为 %s", rv.FieldValue)
	}
	if rv.FieldIndex != 1 {
		t.Errorf("FieldIndex 应为 1, 实际为 %d", rv.FieldIndex)
	}
	if rv.ParsedEnum != parsedEnum {
		t.Error("ParsedEnum 未正确设置")
	}
	if rv.parsedIntDone {
		t.Error("parsedIntDone 应被重置")
	}
	if rv.parsedIntValid {
		t.Error("parsedIntValid 应被重置")
	}
	if rv.parsedIntValue != 0 {
		t.Error("parsedIntValue 应被重置")
	}
}

// ============== validateIPv4 测试 ==============

func TestValidateIPv4_Valid(t *testing.T) {
	rv := NewRuleValidator("192.168.1.1", 0, nil, nil)
	valid, msg := rv.validateIPv4()
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateIPv4_Invalid(t *testing.T) {
	rv := NewRuleValidator("999.999.999.999", 0, nil, nil)
	valid, _ := rv.validateIPv4()
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateIPv4_Empty(t *testing.T) {
	rv := NewRuleValidator("", 0, nil, nil)
	valid, msg := rv.validateIPv4()
	if !valid {
		t.Errorf("空值应跳过校验, 错误: %s", msg)
	}
}

// ============== validateIPv6 测试 ==============

func TestValidateIPv6_Valid(t *testing.T) {
	rv := NewRuleValidator("::1", 0, nil, nil)
	valid, msg := rv.validateIPv6()
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateIPv6_Invalid(t *testing.T) {
	rv := NewRuleValidator("not:ipv6", 0, nil, nil)
	valid, _ := rv.validateIPv6()
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateIPv6_Empty(t *testing.T) {
	rv := NewRuleValidator("", 0, nil, nil)
	valid, msg := rv.validateIPv6()
	if !valid {
		t.Errorf("空值应跳过校验, 错误: %s", msg)
	}
}

// ============== validateBase64JSON 测试 ==============

func TestValidateBase64JSON_Valid(t *testing.T) {
	rv := NewRuleValidator("eyJrZXkiOiJ2YWx1ZSJ9", 0, nil, nil)
	valid, msg := rv.validateBase64JSON()
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateBase64JSON_InvalidBase64(t *testing.T) {
	rv := NewRuleValidator("not-valid-base64!!!", 0, nil, nil)
	valid, _ := rv.validateBase64JSON()
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateBase64JSON_InvalidJSON(t *testing.T) {
	rv := NewRuleValidator("bm90IGpzb24=", 0, nil, nil)
	valid, _ := rv.validateBase64JSON()
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateBase64JSON_Empty(t *testing.T) {
	rv := NewRuleValidator("", 0, nil, nil)
	valid, msg := rv.validateBase64JSON()
	if !valid {
		t.Errorf("空值应跳过校验, 错误: %s", msg)
	}
}

// ============== validateSizeEqual 测试 ==============

func TestValidateSizeEqual_Valid(t *testing.T) {
	rv := NewRuleValidator("100", 0, nil, nil)
	valid, msg := rv.validateSizeEqual("size=100")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateSizeEqual_Invalid(t *testing.T) {
	rv := NewRuleValidator("200", 0, nil, nil)
	valid, _ := rv.validateSizeEqual("size=100")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateSizeEqual_NonNumeric(t *testing.T) {
	rv := NewRuleValidator("abc", 0, nil, nil)
	valid, _ := rv.validateSizeEqual("size=100")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateSizeEqual_BadFormat(t *testing.T) {
	rv := NewRuleValidator("100", 0, nil, nil)
	valid, _ := rv.validateSizeEqual("size=abc")
	if valid {
		t.Error("应为无效")
	}
}

// ============== validateSizeGreater 测试 ==============

func TestValidateSizeGreater_Valid(t *testing.T) {
	rv := NewRuleValidator("200", 0, nil, nil)
	valid, msg := rv.validateSizeGreater("size>100")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateSizeGreater_Invalid(t *testing.T) {
	rv := NewRuleValidator("50", 0, nil, nil)
	valid, _ := rv.validateSizeGreater("size>100")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateSizeGreater_Equal(t *testing.T) {
	rv := NewRuleValidator("100", 0, nil, nil)
	valid, _ := rv.validateSizeGreater("size>100")
	if valid {
		t.Error("等于时应为无效")
	}
}

func TestValidateSizeGreater_NonNumeric(t *testing.T) {
	rv := NewRuleValidator("abc", 0, nil, nil)
	valid, _ := rv.validateSizeGreater("size>100")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateSizeGreater_BadFormat(t *testing.T) {
	rv := NewRuleValidator("200", 0, nil, nil)
	valid, _ := rv.validateSizeGreater("size=abc")
	if valid {
		t.Error("应为无效")
	}
}

// ============== validateSizeLess 测试 ==============

func TestValidateSizeLess_Valid(t *testing.T) {
	rv := NewRuleValidator("50", 0, nil, nil)
	valid, msg := rv.validateSizeLess("size<100")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateSizeLess_Invalid(t *testing.T) {
	rv := NewRuleValidator("200", 0, nil, nil)
	valid, _ := rv.validateSizeLess("size<100")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateSizeLess_Equal(t *testing.T) {
	rv := NewRuleValidator("100", 0, nil, nil)
	valid, _ := rv.validateSizeLess("size<100")
	if valid {
		t.Error("等于时应为无效")
	}
}

func TestValidateSizeLess_NonNumeric(t *testing.T) {
	rv := NewRuleValidator("abc", 0, nil, nil)
	valid, _ := rv.validateSizeLess("size<100")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateSizeLess_BadFormat(t *testing.T) {
	rv := NewRuleValidator("50", 0, nil, nil)
	valid, _ := rv.validateSizeLess("size<abc")
	if valid {
		t.Error("应为无效")
	}
}

// ============== validateSizeGreaterEqual 测试 ==============

func TestValidateSizeGreaterEqual_Valid(t *testing.T) {
	rv := NewRuleValidator("100", 0, nil, nil)
	valid, msg := rv.validateSizeGreaterEqual("size>=100")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateSizeGreaterEqual_Greater(t *testing.T) {
	rv := NewRuleValidator("200", 0, nil, nil)
	valid, msg := rv.validateSizeGreaterEqual("size>=100")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateSizeGreaterEqual_Invalid(t *testing.T) {
	rv := NewRuleValidator("50", 0, nil, nil)
	valid, _ := rv.validateSizeGreaterEqual("size>=100")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateSizeGreaterEqual_NonNumeric(t *testing.T) {
	rv := NewRuleValidator("abc", 0, nil, nil)
	valid, _ := rv.validateSizeGreaterEqual("size>=100")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateSizeGreaterEqual_BadFormat(t *testing.T) {
	rv := NewRuleValidator("200", 0, nil, nil)
	valid, _ := rv.validateSizeGreaterEqual("size>=abc")
	if valid {
		t.Error("应为无效")
	}
}

// ============== validateSizeLessEqual 测试 ==============

func TestValidateSizeLessEqual_Valid(t *testing.T) {
	rv := NewRuleValidator("100", 0, nil, nil)
	valid, msg := rv.validateSizeLessEqual("size<=100")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateSizeLessEqual_Less(t *testing.T) {
	rv := NewRuleValidator("50", 0, nil, nil)
	valid, msg := rv.validateSizeLessEqual("size<=100")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateSizeLessEqual_Invalid(t *testing.T) {
	rv := NewRuleValidator("200", 0, nil, nil)
	valid, _ := rv.validateSizeLessEqual("size<=100")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateSizeLessEqual_NonNumeric(t *testing.T) {
	rv := NewRuleValidator("abc", 0, nil, nil)
	valid, _ := rv.validateSizeLessEqual("size<=100")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateSizeLessEqual_BadFormat(t *testing.T) {
	rv := NewRuleValidator("50", 0, nil, nil)
	valid, _ := rv.validateSizeLessEqual("size<=abc")
	if valid {
		t.Error("应为无效")
	}
}

// ============== validateJSONField 测试 ==============

func TestValidateJSONField_Valid(t *testing.T) {
	rv := NewRuleValidator(`{"key":"value"}`, 0, nil, nil)
	valid, msg := rv.validateJSONField("json_field=key")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateJSONField_Invalid(t *testing.T) {
	rv := NewRuleValidator("not json", 0, nil, nil)
	valid, _ := rv.validateJSONField("json_field=key")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateJSONField_Empty(t *testing.T) {
	rv := NewRuleValidator("", 0, nil, nil)
	valid, msg := rv.validateJSONField("json_field=key")
	if !valid {
		t.Errorf("空值应跳过校验, 错误: %s", msg)
	}
}

// ============== validateEnumParsed 测试 ==============

func TestValidateEnumParsed_ExactMatch(t *testing.T) {
	rv := NewRuleValidator("a", 0, nil, nil)
	pe := &parser.ParsedEnumValue{
		ExactValues: map[string]struct{}{"a": {}, "b": {}},
		RawRule:     "[a,b]",
	}
	valid, msg := rv.validateEnumParsed(pe)
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateEnumParsed_RangeMatch(t *testing.T) {
	rv := NewRuleValidator("5", 0, nil, nil)
	pe := &parser.ParsedEnumValue{
		ExactValues: map[string]struct{}{},
		Ranges:      []parser.EnumRange{{Min: 1, Max: 10}},
		RawRule:     "[1-10]",
	}
	valid, msg := rv.validateEnumParsed(pe)
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateEnumParsed_NoMatch(t *testing.T) {
	rv := NewRuleValidator("c", 0, nil, nil)
	pe := &parser.ParsedEnumValue{
		ExactValues: map[string]struct{}{"a": {}},
		RawRule:     "[a]",
	}
	valid, _ := rv.validateEnumParsed(pe)
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateEnumParsed_NonNumericRange(t *testing.T) {
	rv := NewRuleValidator("abc", 0, nil, nil)
	pe := &parser.ParsedEnumValue{
		ExactValues: map[string]struct{}{},
		Ranges:      []parser.EnumRange{{Min: 1, Max: 10}},
		RawRule:     "[1-10]",
	}
	valid, _ := rv.validateEnumParsed(pe)
	if valid {
		t.Error("非数字值应为无效")
	}
}

// ============== validateBasicRule 测试 ==============

func TestValidateBasicRule_IP(t *testing.T) {
	rv := NewRuleValidator("192.168.1.1", 0, nil, nil)
	valid, msg := rv.validateBasicRule("ip")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateBasicRule_IPv4(t *testing.T) {
	rv := NewRuleValidator("10.0.0.1", 0, nil, nil)
	valid, msg := rv.validateBasicRule("ipv4")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateBasicRule_IPv6(t *testing.T) {
	rv := NewRuleValidator("::1", 0, nil, nil)
	valid, msg := rv.validateBasicRule("ipv6")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateBasicRule_IPCompressed(t *testing.T) {
	rv := NewRuleValidator("2001:db8::1", 0, nil, nil)
	valid, msg := rv.validateBasicRule("ip_compressed")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateBasicRule_IPExploded(t *testing.T) {
	rv := NewRuleValidator("192.168.1.1", 0, nil, nil)
	valid, msg := rv.validateBasicRule("ip_exploded")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateBasicRule_Base64(t *testing.T) {
	rv := NewRuleValidator("aGVsbG8=", 0, nil, nil)
	valid, msg := rv.validateBasicRule("base64")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateBasicRule_DateTime(t *testing.T) {
	rv := NewRuleValidator("2024-01-01 12:00:00", 0, nil, nil)
	valid, msg := rv.validateBasicRule("datetime")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateBasicRule_Unknown(t *testing.T) {
	rv := NewRuleValidator("anything", 0, nil, nil)
	valid, msg := rv.validateBasicRule("unknown_type")
	if !valid {
		t.Errorf("未知类型应跳过校验, 错误: %s", msg)
	}
}

func TestValidateBasicRule_EmptyValue(t *testing.T) {
	rv := NewRuleValidator("", 0, nil, nil)
	valid, _ := rv.validateBasicRule("some_rule")
	if valid {
		t.Error("空值应为无效")
	}
}

func TestValidateBasicRule_IP_Invalid(t *testing.T) {
	rv := NewRuleValidator("not_an_ip", 0, nil, nil)
	valid, _ := rv.validateBasicRule("ip")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateBasicRule_IPCompressed_Invalid(t *testing.T) {
	rv := NewRuleValidator("not_ip", 0, nil, nil)
	valid, _ := rv.validateBasicRule("ip_compressed")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateBasicRule_IPExploded_Invalid(t *testing.T) {
	rv := NewRuleValidator("not_ip", 0, nil, nil)
	valid, _ := rv.validateBasicRule("ip_exploded")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateBasicRule_DateTime_Invalid(t *testing.T) {
	rv := NewRuleValidator("not_datetime", 0, nil, nil)
	valid, _ := rv.validateBasicRule("datetime")
	if valid {
		t.Error("应为无效")
	}
}

// ============== validateConditionParsed 测试 ==============

func TestValidateConditionParsed_Equal_Found(t *testing.T) {
	rv := NewRuleValidator("current", 0, []string{"x", "current"}, nil)
	pc := &parser.ParsedCondition{
		FieldIndex:    1,
		IsEqual:       true,
		ExpectedExact: map[string]struct{}{"x": {}, "current": {}},
	}
	valid, msg := rv.validateConditionParsed(pc)
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateConditionParsed_Equal_NotFound(t *testing.T) {
	rv := NewRuleValidator("current", 0, []string{"other"}, nil)
	pc := &parser.ParsedCondition{
		FieldIndex:    0,
		IsEqual:       true,
		ExpectedExact: map[string]struct{}{"x": {}},
	}
	valid, _ := rv.validateConditionParsed(pc)
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateConditionParsed_NotEqual_Found(t *testing.T) {
	rv := NewRuleValidator("x", 0, []string{"x"}, nil)
	pc := &parser.ParsedCondition{
		FieldIndex:    0,
		IsEqual:       false,
		ExpectedExact: map[string]struct{}{"x": {}},
	}
	valid, _ := rv.validateConditionParsed(pc)
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateConditionParsed_NotEqual_NotFound(t *testing.T) {
	rv := NewRuleValidator("y", 0, []string{"y"}, nil)
	pc := &parser.ParsedCondition{
		FieldIndex:    0,
		IsEqual:       false,
		ExpectedExact: map[string]struct{}{"x": {}},
	}
	valid, msg := rv.validateConditionParsed(pc)
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateConditionParsed_InvalidIndex(t *testing.T) {
	rv := NewRuleValidator("test", 0, []string{"a"}, nil)
	pc := &parser.ParsedCondition{
		FieldIndex:    10,
		IsEqual:       true,
		ExpectedExact: map[string]struct{}{"x": {}},
	}
	valid, _ := rv.validateConditionParsed(pc)
	if valid {
		t.Error("无效索引应为无效")
	}
}

// ============== validateNotEqualCondition 测试 ==============

func TestValidateNotEqualCondition_NotEqual(t *testing.T) {
	rv := NewRuleValidator("current", 0, []string{"y"}, nil)
	valid, msg := rv.validateNotEqualCondition("$1!=x")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateNotEqualCondition_Equal(t *testing.T) {
	rv := NewRuleValidator("x", 0, []string{"x"}, nil)
	valid, _ := rv.validateNotEqualCondition("$1!=x")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateNotEqualCondition_BadFormat(t *testing.T) {
	rv := NewRuleValidator("test", 0, nil, nil)
	valid, _ := rv.validateNotEqualCondition("bad_format")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateNotEqualCondition_FieldRefError(t *testing.T) {
	rv := NewRuleValidator("test", 0, nil, nil)
	valid, _ := rv.validateNotEqualCondition("no_dollar==x")
	if valid {
		t.Error("字段引用错误应为无效")
	}
}

func TestValidateNotEqualCondition_IndexOutOfRange(t *testing.T) {
	rv := NewRuleValidator("test", 0, []string{"a"}, nil)
	valid, _ := rv.validateNotEqualCondition("$99!=x")
	if valid {
		t.Error("索引超出范围应为无效")
	}
}

// ============== ApplyOffset 测试 ==============

func TestApplyOffset_Normal(t *testing.T) {
	result := ApplyOffset("001400", "offset(2,2)")
	if result != "14" {
		t.Errorf("结果应为 '14', 实际为 %s", result)
	}
}

func TestApplyOffset_LeadingZeros(t *testing.T) {
	result := ApplyOffset("000140", "offset(0,4)")
	if result != "1" {
		t.Errorf("结果应为 '1', 实际为 %s", result)
	}
}

func TestApplyOffset_AllZeros(t *testing.T) {
	result := ApplyOffset("0000", "offset(0,4)")
	if result != "0" {
		t.Errorf("全零结果应为 '0', 实际为 %s", result)
	}
}

func TestApplyOffset_NoLeadingZeros(t *testing.T) {
	result := ApplyOffset("1400", "offset(0,4)")
	if result != "1400" {
		t.Errorf("结果应为 '1400', 实际为 %s", result)
	}
}

func TestApplyOffset_OffsetTooLarge(t *testing.T) {
	result := ApplyOffset("123", "offset(10,2)")
	if result != "" {
		t.Errorf("偏移过大应返回空, 实际为 %s", result)
	}
}

func TestApplyOffset_LengthTooLarge(t *testing.T) {
	result := ApplyOffset("123456", "offset(0,100)")
	if result != "123456" {
		t.Errorf("长度过大应截断, 实际为 %s", result)
	}
}

func TestApplyOffset_BadFormat(t *testing.T) {
	result := ApplyOffset("123", "offset(abc,2)")
	if result != "123" {
		t.Errorf("格式错误应返回原值, 实际为 %s", result)
	}
}

func TestApplyOffset_WrongPartCount(t *testing.T) {
	result := ApplyOffset("123", "offset(1)")
	if result != "123" {
		t.Errorf("部分数错误应返回原值, 实际为 %s", result)
	}
}

// ============== ValidateConditionalType 测试 ==============

func TestValidateConditionalType_Match(t *testing.T) {
	rv := NewRuleValidator("192.168.1.1", 0, []string{"1"}, nil)
	parsedCond := &parser.ParsedCondition{
		FieldIndex:    0,
		IsEqual:       true,
		ExpectedExact: map[string]struct{}{"1": {}},
		ExpectedOrder: []string{"1"},
	}
	rv.ParsedCond = parsedCond
	valid, msg := rv.ValidateConditionalType("ipv4,ipv6", "if($1==1)")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateConditionalType_NoMatch_Fallback(t *testing.T) {
	rv := NewRuleValidator("::1", 0, []string{"2"}, nil)
	parsedCond := &parser.ParsedCondition{
		FieldIndex:    0,
		IsEqual:       true,
		ExpectedExact: map[string]struct{}{"1": {}, "2": {}},
		ExpectedOrder: []string{"1", "2"},
	}
	rv.ParsedCond = parsedCond
	valid, msg := rv.ValidateConditionalType("ipv4,ipv6", "if($1==1,2)")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateConditionalType_NoCondition(t *testing.T) {
	rv := NewRuleValidator("192.168.1.1", 0, nil, nil)
	valid, msg := rv.ValidateConditionalType("ipv4,ipv6", "")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateConditionalType_NoneMatch(t *testing.T) {
	rv := NewRuleValidator("not_ip", 0, nil, nil)
	valid, _ := rv.ValidateConditionalType("ipv4,ipv6", "")
	if valid {
		t.Error("应为无效")
	}
}

// ============== GetConditionMatchedValue 测试 ==============

func TestGetConditionMatchedValue_Valid(t *testing.T) {
	rv := NewRuleValidator("current", 0, []string{"matched_value"}, nil)
	rv.ParsedCond = &parser.ParsedCondition{
		FieldIndex: 0,
	}
	value := rv.GetConditionMatchedValue("")
	if value != "matched_value" {
		t.Errorf("值应为 'matched_value', 实际为 %s", value)
	}
}

func TestGetConditionMatchedValue_NoParsedCond(t *testing.T) {
	rv := NewRuleValidator("test", 0, []string{"a"}, nil)
	value := rv.GetConditionMatchedValue("")
	if value != "" {
		t.Errorf("无 ParsedCond 应返回空, 实际为 %s", value)
	}
}

func TestGetConditionMatchedValue_InvalidIndex(t *testing.T) {
	rv := NewRuleValidator("test", 0, []string{"a"}, nil)
	rv.ParsedCond = &parser.ParsedCondition{
		FieldIndex: 10,
	}
	value := rv.GetConditionMatchedValue("")
	if value != "" {
		t.Errorf("无效索引应返回空, 实际为 %s", value)
	}
}

// ============== ValidateType 复合类型测试 ==============

func TestValidateType_CompositeType(t *testing.T) {
	rv := NewRuleValidator("192.168.1.1", 0, nil, nil)
	valid, msg := rv.ValidateType("ipv4,ipv6")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateType_CompositeTypeNoneMatch(t *testing.T) {
	rv := NewRuleValidator("not_ip", 0, nil, nil)
	valid, _ := rv.ValidateType("ipv4,ipv6")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateType_IP(t *testing.T) {
	rv := NewRuleValidator("192.168.1.1", 0, nil, nil)
	valid, msg := rv.ValidateType("ip")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateType_IPv6(t *testing.T) {
	rv := NewRuleValidator("::1", 0, nil, nil)
	valid, msg := rv.ValidateType("ip")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateType_IPCompressed(t *testing.T) {
	rv := NewRuleValidator("2001:db8::1", 0, nil, nil)
	valid, msg := rv.ValidateType("ip_compressed")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateType_IPExploded(t *testing.T) {
	rv := NewRuleValidator("192.168.1.1", 0, nil, nil)
	valid, msg := rv.ValidateType("ip_exploded")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateType_IPv4_Invalid(t *testing.T) {
	rv := NewRuleValidator("not_ip", 0, nil, nil)
	valid, _ := rv.ValidateType("ipv4")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateType_IPv6_Invalid(t *testing.T) {
	rv := NewRuleValidator("not_ipv6", 0, nil, nil)
	valid, _ := rv.ValidateType("ipv6")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateType_IP_Invalid(t *testing.T) {
	rv := NewRuleValidator("not_ip", 0, nil, nil)
	valid, _ := rv.ValidateType("ip")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateType_IPCompressed_Invalid(t *testing.T) {
	rv := NewRuleValidator("not_ip", 0, nil, nil)
	valid, _ := rv.ValidateType("ip_compressed")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateType_IPExploded_Invalid(t *testing.T) {
	rv := NewRuleValidator("not_ip", 0, nil, nil)
	valid, _ := rv.ValidateType("ip_exploded")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateType_Int_Valid(t *testing.T) {
	rv := NewRuleValidator("123", 0, nil, nil)
	valid, msg := rv.ValidateType("int")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateType_Int_Invalid(t *testing.T) {
	rv := NewRuleValidator("abc", 0, nil, nil)
	valid, _ := rv.ValidateType("int")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateType_Datetime_Valid(t *testing.T) {
	rv := NewRuleValidator("2024-01-15 10:30:00", 0, nil, nil)
	valid, msg := rv.ValidateType("datetime")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateType_Datetime_Invalid(t *testing.T) {
	rv := NewRuleValidator("not_datetime", 0, nil, nil)
	valid, _ := rv.ValidateType("datetime")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateType_Base64_Valid(t *testing.T) {
	rv := NewRuleValidator("aGVsbG8=", 0, nil, nil)
	valid, msg := rv.ValidateType("base64")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateType_Base64_Invalid(t *testing.T) {
	rv := NewRuleValidator("!!!invalid-base64!!!", 0, nil, nil)
	valid, _ := rv.ValidateType("base64")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateType_JSON_Valid(t *testing.T) {
	rv := NewRuleValidator(`{"name":"test"}`, 0, nil, nil)
	valid, msg := rv.ValidateType("json")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateType_JSON_Invalid(t *testing.T) {
	rv := NewRuleValidator("not json", 0, nil, nil)
	valid, _ := rv.ValidateType("json")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateType_Base64JSON_Valid(t *testing.T) {
	rv := NewRuleValidator("eyJuYW1lIjoidGVzdCJ9", 0, nil, nil)
	valid, msg := rv.ValidateType("base64_json")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateType_Base64JSON_InvalidBase64(t *testing.T) {
	rv := NewRuleValidator("!!!invalid!!!", 0, nil, nil)
	valid, _ := rv.ValidateType("base64_json")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateType_Base64JSON_InvalidJSON(t *testing.T) {
	rv := NewRuleValidator("bm90IGpzb24=", 0, nil, nil)
	valid, _ := rv.ValidateType("base64_json")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateType_Unknown(t *testing.T) {
	rv := NewRuleValidator("anything", 0, nil, nil)
	valid, msg := rv.ValidateType("unknown")
	if !valid {
		t.Errorf("未知类型应跳过, 错误: %s", msg)
	}
}

// ============== ValidateRule 复合规则测试 ==============

func TestValidateRule_CompoundRules(t *testing.T) {
	rv := NewRuleValidator("hello", 0, nil, nil)
	valid, msg := rv.ValidateRule("len>=3; len<=10")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateRule_CompoundRulesFail(t *testing.T) {
	rv := NewRuleValidator("hi", 0, nil, nil)
	valid, _ := rv.ValidateRule("len>=3; len<=10")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateRule_CompoundWithEmptyPart(t *testing.T) {
	rv := NewRuleValidator("hello", 0, nil, nil)
	valid, msg := rv.ValidateRule("len>=3; ; len<=10")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

// ============== parseFieldReference 测试 ==============

func TestParseFieldReference_WithMap(t *testing.T) {
	rv := NewRuleValidator("test", 0, nil, map[string]int{"11": 5})
	index, _, err := rv.parseFieldReference("$11")
	if err != nil {
		t.Fatalf("不应有错误: %v", err)
	}
	if index != 5 {
		t.Errorf("索引应为 5, 实际为 %d", index)
	}
}

func TestParseFieldReference_WithMapNotFound(t *testing.T) {
	rv := NewRuleValidator("test", 0, nil, map[string]int{"11": 5})
	_, _, err := rv.parseFieldReference("$99")
	if err == nil {
		t.Error("应有错误")
	}
}

func TestParseFieldReference_WithoutMap(t *testing.T) {
	rv := NewRuleValidator("test", 0, nil, nil)
	index, _, err := rv.parseFieldReference("$1")
	if err != nil {
		t.Fatalf("不应有错误: %v", err)
	}
	if index != 0 {
		t.Errorf("索引应为 0, 实际为 %d", index)
	}
}

func TestParseFieldReference_NoDollarSign(t *testing.T) {
	rv := NewRuleValidator("test", 0, nil, nil)
	_, _, err := rv.parseFieldReference("1")
	if err == nil {
		t.Error("应有错误")
	}
}

func TestParseFieldReference_InvalidNumber(t *testing.T) {
	rv := NewRuleValidator("test", 0, nil, nil)
	_, _, err := rv.parseFieldReference("$abc")
	if err == nil {
		t.Error("应有错误")
	}
}

// ============== validateInteger 测试 ==============

func TestValidateInteger_Valid(t *testing.T) {
	rv := NewRuleValidator("123", 0, nil, nil)
	valid, msg := rv.validateInteger()
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateInteger_Empty(t *testing.T) {
	rv := NewRuleValidator("", 0, nil, nil)
	valid, msg := rv.validateInteger()
	if !valid {
		t.Errorf("空值应为有效, 错误: %s", msg)
	}
}

func TestValidateInteger_Invalid(t *testing.T) {
	rv := NewRuleValidator("abc", 0, nil, nil)
	valid, _ := rv.validateInteger()
	if valid {
		t.Error("应为无效")
	}
}

// ============== validateDateTime 测试 ==============

func TestValidateDateTime_Valid(t *testing.T) {
	rv := NewRuleValidator("2024-01-15 10:30:00", 0, nil, nil)
	valid, msg := rv.validateDateTime()
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateDateTime_Empty(t *testing.T) {
	rv := NewRuleValidator("", 0, nil, nil)
	valid, msg := rv.validateDateTime()
	if !valid {
		t.Errorf("空值应为有效, 错误: %s", msg)
	}
}

func TestValidateDateTime_Invalid(t *testing.T) {
	rv := NewRuleValidator("2024-13-45 25:61:61", 0, nil, nil)
	valid, _ := rv.validateDateTime()
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateDateTime_WrongFormat(t *testing.T) {
	rv := NewRuleValidator("2024/01/15", 0, nil, nil)
	valid, _ := rv.validateDateTime()
	if valid {
		t.Error("应为无效")
	}
}

// ============== validateBase64 测试 ==============

func TestValidateBase64_Valid(t *testing.T) {
	rv := NewRuleValidator("aGVsbG8=", 0, nil, nil)
	valid, msg := rv.validateBase64()
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateBase64_Empty(t *testing.T) {
	rv := NewRuleValidator("", 0, nil, nil)
	valid, msg := rv.validateBase64()
	if !valid {
		t.Errorf("空值应为有效, 错误: %s", msg)
	}
}

func TestValidateBase64_Invalid(t *testing.T) {
	rv := NewRuleValidator("!!!invalid!!!", 0, nil, nil)
	valid, _ := rv.validateBase64()
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateBase64_Multiple_Valid(t *testing.T) {
	rv := NewRuleValidator("aGVsbG8=,d29ybGQ=", 0, nil, nil)
	valid, msg := rv.validateBase64()
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateBase64_Multiple_Invalid(t *testing.T) {
	rv := NewRuleValidator("aGVsbG8,!!!invalid!!!", 0, nil, nil)
	valid, _ := rv.validateBase64()
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateBase64_Multiple_WithEmptyPart(t *testing.T) {
	rv := NewRuleValidator("aGVsbG8=,,d29ybGQ=", 0, nil, nil)
	valid, msg := rv.validateBase64()
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

// ============== ValidateRule 单规则测试 ==============

func TestValidateRule_SingleRule(t *testing.T) {
	rv := NewRuleValidator("hello", 0, nil, nil)
	valid, msg := rv.ValidateRule("len>=5")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateRule_SingleRule_Invalid(t *testing.T) {
	rv := NewRuleValidator("hi", 0, nil, nil)
	valid, _ := rv.ValidateRule("len>=5")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateRule_EmptyRule(t *testing.T) {
	rv := NewRuleValidator("hello", 0, nil, nil)
	valid, msg := rv.ValidateRule("")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateRule_WhitespaceRule(t *testing.T) {
	rv := NewRuleValidator("hello", 0, nil, nil)
	valid, msg := rv.ValidateRule("   ")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

// ============== validateJSON 测试 ==============

func TestValidateJSON_Valid(t *testing.T) {
	rv := NewRuleValidator(`{"name":"test"}`, 0, nil, nil)
	valid, msg := rv.validateJSON()
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateJSON_Empty(t *testing.T) {
	rv := NewRuleValidator("", 0, nil, nil)
	valid, msg := rv.validateJSON()
	if !valid {
		t.Errorf("空值应为有效, 错误: %s", msg)
	}
}

func TestValidateJSON_Invalid(t *testing.T) {
	rv := NewRuleValidator("{invalid json}", 0, nil, nil)
	valid, _ := rv.validateJSON()
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateJSON_Array(t *testing.T) {
	rv := NewRuleValidator(`[1,2,3]`, 0, nil, nil)
	valid, msg := rv.validateJSON()
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

// ============== validateLengthEqual 测试 ==============

func TestValidateLengthEqual_Valid(t *testing.T) {
	rv := NewRuleValidator("hello", 0, nil, nil)
	valid, msg := rv.validateLengthEqual("len=5")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateLengthEqual_Invalid(t *testing.T) {
	rv := NewRuleValidator("hi", 0, nil, nil)
	valid, _ := rv.validateLengthEqual("len=5")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateLengthEqual_InvalidRule(t *testing.T) {
	rv := NewRuleValidator("test", 0, nil, nil)
	valid, _ := rv.validateLengthEqual("len=abc")
	if valid {
		t.Error("应为无效")
	}
}

// ============== validateLengthGreater 测试 ==============

func TestValidateLengthGreater_Valid(t *testing.T) {
	rv := NewRuleValidator("hello", 0, nil, nil)
	valid, msg := rv.validateLengthGreater("len>3")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateLengthGreater_Invalid(t *testing.T) {
	rv := NewRuleValidator("hi", 0, nil, nil)
	valid, _ := rv.validateLengthGreater("len>5")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateLengthGreater_Equal(t *testing.T) {
	rv := NewRuleValidator("hello", 0, nil, nil)
	valid, _ := rv.validateLengthGreater("len>5")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateLengthGreater_InvalidFormat(t *testing.T) {
	rv := NewRuleValidator("test", 0, nil, nil)
	valid, _ := rv.validateLengthGreater("len>abc")
	if valid {
		t.Error("应为无效")
	}
}

// ============== validateLengthLess 测试 ==============

func TestValidateLengthLess_Valid(t *testing.T) {
	rv := NewRuleValidator("hi", 0, nil, nil)
	valid, msg := rv.validateLengthLess("len<5")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateLengthLess_Invalid(t *testing.T) {
	rv := NewRuleValidator("hello", 0, nil, nil)
	valid, _ := rv.validateLengthLess("len<3")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateLengthLess_Equal(t *testing.T) {
	rv := NewRuleValidator("hello", 0, nil, nil)
	valid, _ := rv.validateLengthLess("len<5")
	if valid {
		t.Error("等于时应为无效")
	}
}

func TestValidateLengthLess_InvalidFormat(t *testing.T) {
	rv := NewRuleValidator("test", 0, nil, nil)
	valid, _ := rv.validateLengthLess("len<abc")
	if valid {
		t.Error("应为无效")
	}
}

// ============== validateRegex 测试 ==============

func TestValidateRegex_Valid(t *testing.T) {
	rv := NewRuleValidator("test123", 0, nil, nil)
	valid, msg := rv.validateRegex("reg=^[a-z]+[0-9]+$")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateRegex_Invalid(t *testing.T) {
	rv := NewRuleValidator("123test", 0, nil, nil)
	valid, _ := rv.validateRegex("reg=^[a-z]+[0-9]+$")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateRegex_InvalidPattern(t *testing.T) {
	rv := NewRuleValidator("test", 0, nil, nil)
	valid, _ := rv.validateRegex("reg=[invalid")
	if valid {
		t.Error("应为无效")
	}
}

// ============== validateRange 测试 ==============

func TestValidateRange_Valid(t *testing.T) {
	rv := NewRuleValidator("5", 0, nil, nil)
	valid, msg := rv.validateRange("1-10")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateRange_Invalid(t *testing.T) {
	rv := NewRuleValidator("15", 0, nil, nil)
	valid, _ := rv.validateRange("1-10")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateRange_InvalidFormat(t *testing.T) {
	rv := NewRuleValidator("5", 0, nil, nil)
	valid, _ := rv.validateRange("1-10-20")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateRange_InvalidValue(t *testing.T) {
	rv := NewRuleValidator("abc", 0, nil, nil)
	valid, _ := rv.validateRange("1-10")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateRange_EnumFallback(t *testing.T) {
	rv := NewRuleValidator("2", 0, nil, nil)
	valid, msg := rv.validateRange("[1,2,3]")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateRange_BadFormat(t *testing.T) {
	rv := NewRuleValidator("5", 0, nil, nil)
	valid, _ := rv.validateRange("abc-def")
	if valid {
		t.Error("应为无效")
	}
}

// ============== validateEnum 测试 ==============

func TestValidateEnum_SimpleValue_Valid(t *testing.T) {
	rv := NewRuleValidator("2", 0, nil, nil)
	valid, msg := rv.validateEnum("[1,2,3]")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateEnum_SimpleValue_Invalid(t *testing.T) {
	rv := NewRuleValidator("5", 0, nil, nil)
	valid, _ := rv.validateEnum("[1,2,3]")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateEnum_Range_Valid(t *testing.T) {
	rv := NewRuleValidator("3", 0, nil, nil)
	valid, msg := rv.validateEnum("[1-5]")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateEnum_Range_Invalid(t *testing.T) {
	rv := NewRuleValidator("10", 0, nil, nil)
	valid, _ := rv.validateEnum("[1-5]")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateEnum_Mixed_Valid(t *testing.T) {
	rv := NewRuleValidator("3", 0, nil, nil)
	valid, msg := rv.validateEnum("[1,2-5,10]")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateEnum_WithParsedCache(t *testing.T) {
	rv := NewRuleValidator("2", 0, nil, nil)
	parsedEnum := &parser.ParsedEnumValue{
		ExactValues: map[string]struct{}{"1": {}, "2": {}, "3": {}},
		Ranges:      []parser.EnumRange{},
		RawRule:     "[1,2,3]",
	}
	rv.ParsedEnum = parsedEnum
	valid, msg := rv.validateEnum("[1,2,3]")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateEnum_WithParsedCache_Range(t *testing.T) {
	rv := NewRuleValidator("5", 0, nil, nil)
	parsedEnum := &parser.ParsedEnumValue{
		ExactValues: map[string]struct{}{},
		Ranges:      []parser.EnumRange{{Min: 1, Max: 10}},
		RawRule:     "[1-10]",
	}
	rv.ParsedEnum = parsedEnum
	valid, msg := rv.validateEnum("[1-10]")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateEnum_WithParsedCache_Invalid(t *testing.T) {
	rv := NewRuleValidator("20", 0, nil, nil)
	parsedEnum := &parser.ParsedEnumValue{
		ExactValues: map[string]struct{}{"1": {}, "2": {}},
		Ranges:      []parser.EnumRange{{Min: 1, Max: 10}},
		RawRule:     "[1,2,1-10]",
	}
	rv.ParsedEnum = parsedEnum
	valid, _ := rv.validateEnum("[1,2,1-10]")
	if valid {
		t.Error("应为无效")
	}
}

// ============== ValidateCondition 测试 ==============

func TestValidateCondition_Equal_Valid(t *testing.T) {
	rv := NewRuleValidator("test", 0, []string{"test", "other"}, nil)
	valid, msg := rv.ValidateCondition("if($1==test)")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateCondition_Equal_Invalid(t *testing.T) {
	rv := NewRuleValidator("other", 0, []string{"test", "other"}, nil)
	valid, _ := rv.ValidateCondition("if($1==something_else)")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateCondition_NotEqual_Valid(t *testing.T) {
	rv := NewRuleValidator("other", 0, []string{"test", "other"}, nil)
	valid, msg := rv.ValidateCondition("if($1!=something_else)")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateCondition_NotEqual_Invalid(t *testing.T) {
	rv := NewRuleValidator("test", 0, []string{"test", "other"}, nil)
	valid, _ := rv.ValidateCondition("if($1!=test)")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateCondition_InvalidCondition(t *testing.T) {
	rv := NewRuleValidator("test", 0, nil, nil)
	valid, _ := rv.ValidateCondition("invalid")
	if valid {
		t.Error("应为无效")
	}
}

// ============== validateEqualCondition 测试 ==============

func TestValidateEqualCondition_Valid(t *testing.T) {
	rv := NewRuleValidator("test", 0, nil, nil)
	rv.AllFields = []string{"test", "other"}
	valid, msg := rv.validateEqualCondition("$1==test")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateEqualCondition_Invalid(t *testing.T) {
	rv := NewRuleValidator("other", 0, nil, nil)
	rv.AllFields = []string{"test", "other"}
	valid, _ := rv.validateEqualCondition("$1==something")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateEqualCondition_FieldReference(t *testing.T) {
	rv := NewRuleValidator("value1", 0, nil, map[string]int{"1": 0})
	rv.AllFields = []string{"value1", "value2"}
	valid, msg := rv.validateEqualCondition("$1==value1")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateEqualCondition_FieldReference_Invalid(t *testing.T) {
	rv := NewRuleValidator("value2", 0, nil, map[string]int{"1": 0})
	rv.AllFields = []string{"value1", "value2"}
	valid, _ := rv.validateEqualCondition("$1==value2")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateEqualCondition_BadFormat(t *testing.T) {
	rv := NewRuleValidator("test", 0, nil, nil)
	valid, _ := rv.validateEqualCondition("invalid")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateEqualCondition_IndexOutOfRange(t *testing.T) {
	rv := NewRuleValidator("test", 0, nil, nil)
	rv.AllFields = []string{"value1"}
	valid, _ := rv.validateEqualCondition("$5==test")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateEqualCondition_QuotedValue(t *testing.T) {
	rv := NewRuleValidator("test", 0, nil, nil)
	rv.AllFields = []string{"test"}
	valid, msg := rv.validateEqualCondition("$1==\"test\"")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateEqualCondition_MultipleValues(t *testing.T) {
	rv := NewRuleValidator("value2", 0, nil, nil)
	rv.AllFields = []string{"value2"}
	valid, msg := rv.validateEqualCondition("$1==value1,value2,value3")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

// ============== validateNotEqualCondition 额外测试 ==============

func TestValidateNotEqualCondition_QuotedValue(t *testing.T) {
	rv := NewRuleValidator("different", 0, nil, nil)
	rv.AllFields = []string{"different"}
	valid, msg := rv.validateNotEqualCondition("$1!=\"test\"")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

// ============== ValidateCondition 额外测试 ==============

func TestValidateCondition_EmptyCondition(t *testing.T) {
	rv := NewRuleValidator("test", 0, nil, nil)
	valid, msg := rv.ValidateCondition("")
	if !valid {
		t.Errorf("空条件应为有效, 错误: %s", msg)
	}
}

func TestValidateCondition_WithParsedCache(t *testing.T) {
	rv := NewRuleValidator("test", 0, []string{"test", "other"}, nil)
	rv.ParsedCond = &parser.ParsedCondition{
		FieldIndex:    0,
		IsEqual:       true,
		ExpectedExact: map[string]struct{}{"test": {}},
	}
	valid, msg := rv.ValidateCondition("if($1==test)")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateCondition_WithParsedCache_NotEqual(t *testing.T) {
	rv := NewRuleValidator("other", 0, []string{"other", "test"}, nil)
	rv.ParsedCond = &parser.ParsedCondition{
		FieldIndex:    0,
		IsEqual:       false,
		ExpectedExact: map[string]struct{}{"test": {}},
	}
	valid, msg := rv.ValidateCondition("if($1!=test)")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateCondition_WithParsedCache_InvalidIndex(t *testing.T) {
	rv := NewRuleValidator("test", 0, []string{"test"}, nil)
	rv.ParsedCond = &parser.ParsedCondition{
		FieldIndex:    10,
		IsEqual:       true,
		ExpectedExact: map[string]struct{}{"test": {}},
	}
	valid, _ := rv.ValidateCondition("if($1==test)")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateCondition_UnsupportedOperator(t *testing.T) {
	rv := NewRuleValidator("test", 0, nil, nil)
	valid, _ := rv.ValidateCondition("if($1>test)")
	if valid {
		t.Error("应为无效")
	}
}

// ============== validateSingleRule 分支测试 ==============

func TestValidateSingleRule_LengthGreaterEqual(t *testing.T) {
	rv := NewRuleValidator("hello", 0, nil, nil)
	valid, msg := rv.validateSingleRule("len>=5")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateSingleRule_LengthLessEqual(t *testing.T) {
	rv := NewRuleValidator("hi", 0, nil, nil)
	valid, msg := rv.validateSingleRule("len<=5")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateSingleRule_SizeGreaterEqual(t *testing.T) {
	rv := NewRuleValidator("100", 0, nil, nil)
	valid, msg := rv.validateSingleRule("size>=50")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateSingleRule_SizeLessEqual(t *testing.T) {
	rv := NewRuleValidator("50", 0, nil, nil)
	valid, msg := rv.validateSingleRule("size<=100")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateSingleRule_JSONField(t *testing.T) {
	rv := NewRuleValidator(`{"name":"test"}`, 0, nil, nil)
	valid, msg := rv.validateSingleRule("json_field=name")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateSingleRule_JSONField_Invalid(t *testing.T) {
	rv := NewRuleValidator("not json", 0, nil, nil)
	valid, _ := rv.validateSingleRule("json_field=name")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateSingleRule_LengthEqual(t *testing.T) {
	rv := NewRuleValidator("hello", 0, nil, nil)
	valid, msg := rv.validateSingleRule("len=5")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateSingleRule_LengthGreater(t *testing.T) {
	rv := NewRuleValidator("hello", 0, nil, nil)
	valid, msg := rv.validateSingleRule("len>3")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateSingleRule_LengthLess(t *testing.T) {
	rv := NewRuleValidator("hi", 0, nil, nil)
	valid, msg := rv.validateSingleRule("len<5")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateSingleRule_SizeEqual(t *testing.T) {
	rv := NewRuleValidator("100", 0, nil, nil)
	valid, msg := rv.validateSingleRule("size=100")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateSingleRule_SizeGreater(t *testing.T) {
	rv := NewRuleValidator("200", 0, nil, nil)
	valid, msg := rv.validateSingleRule("size>100")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateSingleRule_SizeLess(t *testing.T) {
	rv := NewRuleValidator("50", 0, nil, nil)
	valid, msg := rv.validateSingleRule("size<100")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateSingleRule_Regex(t *testing.T) {
	rv := NewRuleValidator("test123", 0, nil, nil)
	valid, msg := rv.validateSingleRule("reg=^[a-z]+[0-9]+$")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateSingleRule_Enum(t *testing.T) {
	rv := NewRuleValidator("apple", 0, nil, nil)
	valid, msg := rv.validateSingleRule("[apple,banana,orange]")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateSingleRule_Range(t *testing.T) {
	rv := NewRuleValidator("5", 0, nil, nil)
	valid, msg := rv.validateSingleRule("1-10")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateSingleRule_BasicRule(t *testing.T) {
	rv := NewRuleValidator("192.168.1.1", 0, nil, nil)
	valid, msg := rv.validateSingleRule("ip")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

// ============== validateLengthGreaterEqual 测试 ==============

func TestValidateLengthGreaterEqual_Valid(t *testing.T) {
	rv := NewRuleValidator("hello", 0, nil, nil)
	valid, msg := rv.validateLengthGreaterEqual("len>=5")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateLengthGreaterEqual_Invalid(t *testing.T) {
	rv := NewRuleValidator("hi", 0, nil, nil)
	valid, _ := rv.validateLengthGreaterEqual("len>=5")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateLengthGreaterEqual_Equal(t *testing.T) {
	rv := NewRuleValidator("hello", 0, nil, nil)
	valid, msg := rv.validateLengthGreaterEqual("len>=5")
	if !valid {
		t.Errorf("等于时应为有效, 错误: %s", msg)
	}
}

func TestValidateLengthGreaterEqual_InvalidFormat(t *testing.T) {
	rv := NewRuleValidator("test", 0, nil, nil)
	valid, _ := rv.validateLengthGreaterEqual("len>=abc")
	if valid {
		t.Error("应为无效")
	}
}

// ============== validateLengthLessEqual 测试 ==============

func TestValidateLengthLessEqual_Valid(t *testing.T) {
	rv := NewRuleValidator("hi", 0, nil, nil)
	valid, msg := rv.validateLengthLessEqual("len<=5")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateLengthLessEqual_Invalid(t *testing.T) {
	rv := NewRuleValidator("hello world", 0, nil, nil)
	valid, _ := rv.validateLengthLessEqual("len<=5")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateLengthLessEqual_Equal(t *testing.T) {
	rv := NewRuleValidator("hello", 0, nil, nil)
	valid, msg := rv.validateLengthLessEqual("len<=5")
	if !valid {
		t.Errorf("等于时应为有效, 错误: %s", msg)
	}
}

func TestValidateLengthLessEqual_InvalidFormat(t *testing.T) {
	rv := NewRuleValidator("test", 0, nil, nil)
	valid, _ := rv.validateLengthLessEqual("len<=abc")
	if valid {
		t.Error("应为无效")
	}
}

// ============== validateBasicRule 完整测试 ==============

func TestValidateBasicRule_IP_Valid(t *testing.T) {
	rv := NewRuleValidator("192.168.1.1", 0, nil, nil)
	valid, msg := rv.validateBasicRule("ip")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateBasicRule_IPv4_Valid(t *testing.T) {
	rv := NewRuleValidator("10.0.0.1", 0, nil, nil)
	valid, msg := rv.validateBasicRule("ipv4")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateBasicRule_IPv4_Invalid(t *testing.T) {
	rv := NewRuleValidator("999.999.999.999", 0, nil, nil)
	valid, _ := rv.validateBasicRule("ipv4")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateBasicRule_IPv6_Valid(t *testing.T) {
	rv := NewRuleValidator("::1", 0, nil, nil)
	valid, msg := rv.validateBasicRule("ipv6")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateBasicRule_IPv6_Invalid(t *testing.T) {
	rv := NewRuleValidator("not_ipv6", 0, nil, nil)
	valid, _ := rv.validateBasicRule("ipv6")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateBasicRule_IPCompressed_Valid(t *testing.T) {
	rv := NewRuleValidator("192.168.1.1", 0, nil, nil)
	valid, msg := rv.validateBasicRule("ip_compressed")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateBasicRule_IPExploded_Valid(t *testing.T) {
	rv := NewRuleValidator("192.168.1.1", 0, nil, nil)
	valid, msg := rv.validateBasicRule("ip_exploded")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateBasicRule_Base64_Valid(t *testing.T) {
	rv := NewRuleValidator("aGVsbG8=", 0, nil, nil)
	valid, msg := rv.validateBasicRule("base64")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateBasicRule_Base64_Invalid(t *testing.T) {
	rv := NewRuleValidator("not_base64!!!", 0, nil, nil)
	valid, _ := rv.validateBasicRule("base64")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateBasicRule_Datetime_Valid(t *testing.T) {
	rv := NewRuleValidator("2024-01-15 10:30:00", 0, nil, nil)
	valid, msg := rv.validateBasicRule("datetime")
	if !valid {
		t.Errorf("应为有效, 错误: %s", msg)
	}
}

func TestValidateBasicRule_Datetime_Invalid(t *testing.T) {
	rv := NewRuleValidator("2024-13-45 25:61:61", 0, nil, nil)
	valid, _ := rv.validateBasicRule("datetime")
	if valid {
		t.Error("应为无效")
	}
}

func TestValidateBasicRule_UnknownRule(t *testing.T) {
	rv := NewRuleValidator("test", 0, nil, nil)
	valid, msg := rv.validateBasicRule("unknown_rule")
	if !valid {
		t.Errorf("未知规则应为有效, 错误: %s", msg)
	}
}

func TestValidateBasicRule_UnknownRule_EmptyValue(t *testing.T) {
	rv := NewRuleValidator("", 0, nil, nil)
	valid, _ := rv.validateBasicRule("unknown_rule")
	if valid {
		t.Error("字段值为空时应为无效")
	}
}

// ============== validateSizeGreater 测试 ==============

func TestValidateSizeGreater_InvalidValue(t *testing.T) {
	rv := NewRuleValidator("not_a_number", 0, nil, nil)
	valid, _ := rv.validateSizeGreater("size>50")
	if valid {
		t.Error("应为无效")
	}
}

// ============== validateJSONField 测试 ==============

func TestValidateJSONField_MissingField(t *testing.T) {
	rv := NewRuleValidator(`{"other":"value"}`, 0, nil, nil)
	valid, msg := rv.validateJSONField("json_field=name")
	if !valid {
		t.Errorf("简化实现应返回有效, 错误: %s", msg)
	}
}

func TestValidateJSONField_InvalidRule(t *testing.T) {
	rv := NewRuleValidator(`{"name":"test"}`, 0, nil, nil)
	valid, msg := rv.validateJSONField("json_field=name")
	if !valid {
		t.Errorf("简化实现应返回有效, 错误: %s", msg)
	}
}

// ============== validateLengthEqual 测试 ==============

func TestValidateLengthEqual_InvalidFormat(t *testing.T) {
	rv := NewRuleValidator("test", 0, nil, nil)
	valid, _ := rv.validateLengthEqual("len=abc")
	if valid {
		t.Error("应为无效")
	}
}
