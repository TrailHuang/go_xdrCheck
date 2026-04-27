package parser

import (
	"testing"
)

// ============== parseRules 测试 ==============

func TestParseRules_SingleRule(t *testing.T) {
	rules := parseRules("len_ge(5)")
	if len(rules) != 1 {
		t.Errorf("应解析为 1 个规则, 实际为 %d", len(rules))
	}
	if rules[0] != "len_ge(5)" {
		t.Errorf("规则应为 'len_ge(5)', 实际为 %s", rules[0])
	}
}

func TestParseRules_MultipleRules(t *testing.T) {
	rules := parseRules("len_ge(5); len_le(10); reg=^[a-z]+$")
	if len(rules) != 3 {
		t.Errorf("应解析为 3 个规则, 实际为 %d", len(rules))
	}
	if rules[0] != "len_ge(5)" {
		t.Errorf("第一个规则应为 'len_ge(5)', 实际为 %s", rules[0])
	}
	if rules[1] != "len_le(10)" {
		t.Errorf("第二个规则应为 'len_le(10)', 实际为 %s", rules[1])
	}
	if rules[2] != "reg=^[a-z]+$" {
		t.Errorf("第三个规则应为 'reg=^[a-z]+$', 实际为 %s", rules[2])
	}
}

func TestParseRules_WithSpaces(t *testing.T) {
	rules := parseRules("  len_ge(5)  ;  len_le(10)  ")
	if len(rules) != 2 {
		t.Errorf("应解析为 2 个规则, 实际为 %d", len(rules))
	}
	if rules[0] != "len_ge(5)" {
		t.Errorf("规则应去除空格, 实际为 %s", rules[0])
	}
}

func TestParseRules_EmptyRules(t *testing.T) {
	rules := parseRules("")
	if len(rules) != 0 {
		t.Errorf("空字符串应返回空规则列表, 实际为 %d", len(rules))
	}
}

func TestParseRules_WithEmptyParts(t *testing.T) {
	rules := parseRules("len_ge(5);len_le(10)")
	if len(rules) != 2 {
		t.Errorf("应解析为 2 个规则, 实际为 %d", len(rules))
	}
}

// ============== parseEnumRule 测试 ==============

func TestParseEnumRule_ExactValues(t *testing.T) {
	result := parseEnumRule("[1,2,3]")
	if len(result.ExactValues) != 3 {
		t.Errorf("应有 3 个精确值, 实际为 %d", len(result.ExactValues))
	}
	if _, exists := result.ExactValues["1"]; !exists {
		t.Error("应包含值 '1'")
	}
	if _, exists := result.ExactValues["2"]; !exists {
		t.Error("应包含值 '2'")
	}
	if _, exists := result.ExactValues["3"]; !exists {
		t.Error("应包含值 '3'")
	}
}

func TestParseEnumRule_Ranges(t *testing.T) {
	result := parseEnumRule("[1-10, 20-30]")
	if len(result.Ranges) != 2 {
		t.Errorf("应有 2 个范围, 实际为 %d", len(result.Ranges))
	}
	if result.Ranges[0].Min != 1 || result.Ranges[0].Max != 10 {
		t.Errorf("第一个范围应为 [1, 10], 实际为 [%d, %d]", result.Ranges[0].Min, result.Ranges[0].Max)
	}
	if result.Ranges[1].Min != 20 || result.Ranges[1].Max != 30 {
		t.Errorf("第二个范围应为 [20, 30], 实际为 [%d, %d]", result.Ranges[1].Min, result.Ranges[1].Max)
	}
}

func TestParseEnumRule_Mixed(t *testing.T) {
	result := parseEnumRule("[1, 5-10, 20]")
	if len(result.ExactValues) != 2 {
		t.Errorf("应有 2 个精确值, 实际为 %d", len(result.ExactValues))
	}
	if len(result.Ranges) != 1 {
		t.Errorf("应有 1 个范围, 实际为 %d", len(result.Ranges))
	}
}

func TestParseEnumRule_Empty(t *testing.T) {
	result := parseEnumRule("[]")
	if len(result.ExactValues) != 0 {
		t.Errorf("空枚举应无精确值, 实际为 %d", len(result.ExactValues))
	}
	if len(result.Ranges) != 0 {
		t.Errorf("空枚举应无范围, 实际为 %d", len(result.Ranges))
	}
}

func TestParseEnumRule_WithSpaces(t *testing.T) {
	result := parseEnumRule("[ 1 , 2 , 3 ]")
	if len(result.ExactValues) != 3 {
		t.Errorf("应解析为 3 个值, 实际为 %d", len(result.ExactValues))
	}
}

// ============== parseConditionRule 测试 ==============

func TestParseConditionRule_Equal(t *testing.T) {
	fieldNumberMap := map[string]int{"13": 12}
	result := parseConditionRule("if($13==5,8)", fieldNumberMap)
	if result == nil {
		t.Fatal("应成功解析条件规则")
	}
	if result.FieldIndex != 12 {
		t.Errorf("字段索引应为 12, 实际为 %d", result.FieldIndex)
	}
	if !result.IsEqual {
		t.Error("应为等于条件")
	}
	if len(result.ExpectedExact) != 2 {
		t.Errorf("应有 2 个期望值, 实际为 %d", len(result.ExpectedExact))
	}
}

func TestParseConditionRule_NotEqual(t *testing.T) {
	fieldNumberMap := map[string]int{"13": 12}
	result := parseConditionRule("if($13!=5)", fieldNumberMap)
	if result == nil {
		t.Fatal("应成功解析条件规则")
	}
	if result.FieldIndex != 12 {
		t.Errorf("字段索引应为 12, 实际为 %d", result.FieldIndex)
	}
	if result.IsEqual {
		t.Error("应为不等于条件")
	}
}

func TestParseConditionRule_InvalidFormat(t *testing.T) {
	result := parseConditionRule("invalid", nil)
	if result != nil {
		t.Error("无效格式应返回 nil")
	}
}

func TestParseConditionRule_MissingPrefix(t *testing.T) {
	result := parseConditionRule("$13==5", nil)
	if result != nil {
		t.Error("缺少 if 前缀应返回 nil")
	}
}

func TestParseConditionRule_MissingSuffix(t *testing.T) {
	result := parseConditionRule("if($13==5", nil)
	if result != nil {
		t.Error("缺少右括号应返回 nil")
	}
}

func TestParseConditionRule_WithQuotes(t *testing.T) {
	fieldNumberMap := map[string]int{"13": 12}
	result := parseConditionRule(`if($13=="value1","value2")`, fieldNumberMap)
	if result == nil {
		t.Fatal("应成功解析条件规则")
	}
	if len(result.ExpectedExact) != 2 {
		t.Errorf("应有 2 个期望值, 实际为 %d", len(result.ExpectedExact))
	}
	if _, exists := result.ExpectedExact["value1"]; !exists {
		t.Error("应包含值 'value1'")
	}
	if _, exists := result.ExpectedExact["value2"]; !exists {
		t.Error("应包含值 'value2'")
	}
}

func TestParseConditionRule_FieldByNumber(t *testing.T) {
	result := parseConditionRule("if($13==5)", nil)
	if result == nil {
		t.Fatal("应成功解析条件规则")
	}
	if result.FieldIndex != 12 {
		t.Errorf("字段索引应为 12 (13-1), 实际为 %d", result.FieldIndex)
	}
}

// ============== parseComplexRules 测试 ==============

func TestParseComplexRules_Condition(t *testing.T) {
	fr := &FieldRule{}
	parseComplexRules(fr, "if($13==5,8)")
	if fr.Condition != "if($13==5,8)" {
		t.Errorf("条件规则应为 'if($13==5,8)', 实际为 %s", fr.Condition)
	}
}

func TestParseComplexRules_Offset(t *testing.T) {
	fr := &FieldRule{}
	parseComplexRules(fr, "offset(6,4)")
	if fr.Offset != "offset(6,4)" {
		t.Errorf("偏移规则应为 'offset(6,4)', 实际为 %s", fr.Offset)
	}
}

func TestParseComplexRules_Array(t *testing.T) {
	fr := &FieldRule{}
	parseComplexRules(fr, "array(10,11,12)")
	if fr.Array != "array(10,11,12)" {
		t.Errorf("数组规则应为 'array(10,11,12)', 实际为 %s", fr.Array)
	}
}

func TestParseComplexRules_Loop(t *testing.T) {
	fr := &FieldRule{}
	parseComplexRules(fr, "loop(start=,)")
	if fr.Loop != "loop(start=,)" {
		t.Errorf("循环规则应为 'loop(start=,)', 实际为 %s", fr.Loop)
	}
}

func TestParseComplexRules_Jump(t *testing.T) {
	fr := &FieldRule{}
	parseComplexRules(fr, "jump=1")
	if fr.Jump != "jump=1" {
		t.Errorf("跳转规则应为 'jump=1', 实际为 %s", fr.Jump)
	}
}

func TestParseComplexRules_Regex(t *testing.T) {
	fr := &FieldRule{}
	parseComplexRules(fr, "reg=[^ ]+")
	if fr.Regex != "[^ ]+" {
		t.Errorf("正则规则应为 '[^ ]+', 实际为 %s", fr.Regex)
	}
}

func TestParseComplexRules_Type(t *testing.T) {
	fr := &FieldRule{}
	parseComplexRules(fr, "type=ipv4,ipv6")
	if fr.Type != "ipv4,ipv6" {
		t.Errorf("类型规则应为 'ipv4,ipv6', 实际为 %s", fr.Type)
	}
}

func TestParseComplexRules_TypeMerge(t *testing.T) {
	fr := &FieldRule{Type: "ip"}
	parseComplexRules(fr, "type=ipv4,ipv6")
	if fr.Type != "ip,ipv4,ipv6" {
		t.Errorf("类型规则应合并, 实际为 %s", fr.Type)
	}
}

func TestParseComplexRules_MultipleRules(t *testing.T) {
	fr := &FieldRule{}
	parseComplexRules(fr, "if($13==5,8); type=ipv4; reg=^[0-9]+$")
	if fr.Condition != "if($13==5,8)" {
		t.Errorf("条件规则应为 'if($13==5,8)', 实际为 %s", fr.Condition)
	}
	if fr.Type != "ipv4" {
		t.Errorf("类型规则应为 'ipv4', 实际为 %s", fr.Type)
	}
	if fr.Regex != "^[0-9]+$" {
		t.Errorf("正则规则应为 '^[0-9]+$', 实际为 %s", fr.Regex)
	}
}

// ============== preParseFieldRule 测试 ==============

func TestPreParseFieldRule_EnumRules(t *testing.T) {
	fr := &FieldRule{
		Rules: []string{"[1,2,3]", "len_ge(5)"},
	}
	preParseFieldRule(fr, nil)
	if fr.ParsedEnums == nil {
		t.Fatal("应解析枚举规则")
	}
	if len(fr.ParsedEnums) != 1 {
		t.Errorf("应有 1 个枚举规则, 实际为 %d", len(fr.ParsedEnums))
	}
}

func TestPreParseFieldRule_ConditionRule(t *testing.T) {
	fieldNumberMap := map[string]int{"13": 12}
	fr := &FieldRule{
		Condition: "if($13==5,8)",
	}
	preParseFieldRule(fr, fieldNumberMap)
	if fr.ParsedCondition == nil {
		t.Fatal("应解析条件规则")
	}
	if fr.ParsedCondition.FieldIndex != 12 {
		t.Errorf("字段索引应为 12, 实际为 %d", fr.ParsedCondition.FieldIndex)
	}
}

func TestPreParseFieldRule_CompoundEnumRules(t *testing.T) {
	fr := &FieldRule{
		Rules: []string{"[1,2,3]; len_ge(5); [10-20]"},
	}
	preParseFieldRule(fr, nil)
	if fr.ParsedEnums == nil {
		t.Fatal("应解析复合规则中的枚举")
	}
	if len(fr.ParsedEnums) < 2 {
		t.Errorf("应至少解析 2 个枚举规则, 实际为 %d", len(fr.ParsedEnums))
	}
}

// ============== PreParseRules 测试 ==============

func TestPreParseRules_MultipleConfigs(t *testing.T) {
	configs := []SheetConfig{
		{
			SheetName: "test1",
			FieldRules: []FieldRule{
				{Rules: []string{"[1,2,3]"}},
				{Condition: "if($1==1)"},
			},
			FieldNumberMap: map[string]int{"1": 0},
		},
		{
			SheetName: "test2",
			FieldRules: []FieldRule{
				{Rules: []string{"[10-20]"}},
			},
		},
	}

	PreParseRules(configs)

	if configs[0].FieldRules[0].ParsedEnums == nil {
		t.Error("第一个配置的第一个规则应解析枚举")
	}
	if configs[0].FieldRules[1].ParsedCondition == nil {
		t.Error("第一个配置的第二个规则应解析条件")
	}
	if configs[1].FieldRules[0].ParsedEnums == nil {
		t.Error("第二个配置的规则应解析枚举")
	}
}

// ============== parseInt64 测试 ==============

func TestParseInt64_Valid(t *testing.T) {
	result, err := parseInt64("12345")
	if err != nil {
		t.Fatalf("解析有效数字失败: %v", err)
	}
	if result != 12345 {
		t.Errorf("结果应为 12345, 实际为 %d", result)
	}
}

func TestParseInt64_Negative(t *testing.T) {
	result, err := parseInt64("-12345")
	if err != nil {
		t.Fatalf("解析负数失败: %v", err)
	}
	if result != -12345 {
		t.Errorf("结果应为 -12345, 实际为 %d", result)
	}
}

func TestParseInt64_Invalid(t *testing.T) {
	_, err := parseInt64("abc")
	if err == nil {
		t.Error("无效数字应返回错误")
	}
}

// ============== mergeSheetConfigs 测试 ==============

func TestMergeSheetConfigs_BothExist(t *testing.T) {
	fileConfigs := []SheetConfig{
		{
			SheetName: "test",
			FileValidation: FileValidationConfig{
				FileHeader: "header",
				FileSuffix: ".txt",
			},
		},
	}

	sheetConfigs := []SheetConfig{
		{
			SheetName: "test",
			FieldRules: []FieldRule{
				{FieldName: "field1"},
			},
			FieldNumberMap: map[string]int{"1": 0},
		},
	}

	merged := mergeSheetConfigs(fileConfigs, sheetConfigs)
	if len(merged) != 1 {
		t.Errorf("应合并为 1 个配置, 实际为 %d", len(merged))
	}
	if merged[0].FileValidation.FileHeader != "header" {
		t.Error("应包含文件校验配置")
	}
	if len(merged[0].FieldRules) != 1 {
		t.Error("应包含字段规则")
	}
}

func TestMergeSheetConfigs_OnlyFileConfig(t *testing.T) {
	fileConfigs := []SheetConfig{
		{
			SheetName: "test",
			FileValidation: FileValidationConfig{
				FileHeader: "header",
			},
		},
	}

	sheetConfigs := []SheetConfig{}

	merged := mergeSheetConfigs(fileConfigs, sheetConfigs)
	if len(merged) != 1 {
		t.Errorf("应有 1 个配置, 实际为 %d", len(merged))
	}
	if merged[0].FileValidation.FileHeader != "header" {
		t.Error("应包含文件校验配置")
	}
}

func TestMergeSheetConfigs_OnlySheetConfig(t *testing.T) {
	fileConfigs := []SheetConfig{}

	sheetConfigs := []SheetConfig{
		{
			SheetName: "test",
			FieldRules: []FieldRule{
				{FieldName: "field1"},
			},
		},
	}

	merged := mergeSheetConfigs(fileConfigs, sheetConfigs)
	if len(merged) != 1 {
		t.Errorf("应有 1 个配置, 实际为 %d", len(merged))
	}
	if len(merged[0].FieldRules) != 1 {
		t.Error("应包含字段规则")
	}
}

func TestMergeSheetConfigs_DifferentSheets(t *testing.T) {
	fileConfigs := []SheetConfig{
		{
			SheetName: "file_sheet",
			FileValidation: FileValidationConfig{
				FileHeader: "header",
			},
		},
	}

	sheetConfigs := []SheetConfig{
		{
			SheetName: "field_sheet",
			FieldRules: []FieldRule{
				{FieldName: "field1"},
			},
		},
	}

	merged := mergeSheetConfigs(fileConfigs, sheetConfigs)
	if len(merged) != 2 {
		t.Errorf("应有 2 个配置, 实际为 %d", len(merged))
	}
}
