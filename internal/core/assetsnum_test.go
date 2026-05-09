package core

import (
	"strings"
	"testing"
	"xdrCheck/internal/parser"
)

// TestValidateAssetsNumConsistency_SimpleCase 简单测试用例：AssetsNum与DataContent次数总和一致
func TestValidateAssetsNumConsistency_SimpleCase(t *testing.T) {
	sheetConfig := parser.SheetConfig{
		SheetName: "0x31+0x06c0",
		FieldRules: []parser.FieldRule{
			{FieldName: "Field1"},                             // 字段编号1 -> RuleIndex 0
			{FieldName: "AssetsNum"},                          // 字段编号2 -> RuleIndex 1
			{FieldName: "DataInfoNum", Array: "array(4,5,6)"}, // 字段编号3 -> RuleIndex 2, array控制字段4,5,6
			{FieldName: "DataType", Loop: "loop(start=,)"},    // 字段编号4 -> RuleIndex 3
			{FieldName: "DataLevel", Loop: "loop(active=,)"},  // 字段编号5 -> RuleIndex 4
			{FieldName: "DataContent", Loop: "loop(end=,)"},   // 字段编号6 -> RuleIndex 5
		},
		FieldNumberMap: map[string]int{
			"1": 0,
			"2": 1,
			"3": 2,
			"4": 3,
			"5": 4,
			"6": 5,
		},
	}

	// 测试用例：AssetsNum=5, DataContent有3个，次数分别为1,2,2，总和=5
	// DataInfoNum=3 表示有3组数据，每组包含 DataType|DataLevel|DataContent
	line := "field1|5|3|1|1|1001,1|1|1|1002,2|1|1|1003,2"
	errors := validateAssetsNumConsistency(sheetConfig, line, 1, "test.txt", "|")

	if len(errors) != 0 {
		t.Errorf("期望无错误，但得到 %d 个错误", len(errors))
		for _, err := range errors {
			t.Logf("错误: %s", err.Message)
		}
	} else {
		t.Log("测试通过：AssetsNum与DataContent次数总和一致")
	}
}

// TestValidateAssetsNumConsistency_Mismatch 测试不匹配情况
func TestValidateAssetsNumConsistency_Mismatch(t *testing.T) {
	sheetConfig := parser.SheetConfig{
		SheetName: "0x31+0x06c0",
		FieldRules: []parser.FieldRule{
			{FieldName: "Field1"},
			{FieldName: "AssetsNum"},
			{FieldName: "DataInfoNum", Array: "array(4,5,6)"},
			{FieldName: "DataType", Loop: "loop(start=,)"},
			{FieldName: "DataLevel", Loop: "loop(active=,)"},
			{FieldName: "DataContent", Loop: "loop(end=,)"},
		},
		FieldNumberMap: map[string]int{
			"1": 0,
			"2": 1,
			"3": 2,
			"4": 3,
			"5": 4,
			"6": 5,
		},
	}

	// 测试用例：AssetsNum=10, DataContent有3个，次数分别为1,2,2，总和=5，不匹配
	line := "field1|10|3|1|1|1001,1|1|1|1002,2|1|1|1003,2"
	errors := validateAssetsNumConsistency(sheetConfig, line, 1, "test.txt", "|")

	if len(errors) == 0 {
		t.Error("期望有错误但没有得到错误")
	} else {
		t.Logf("正确检测到不一致: %s", errors[0].Message)
		if !strings.Contains(errors[0].Message, "AssetsNum=10") {
			t.Errorf("错误消息应包含 AssetsNum=10，实际: %s", errors[0].Message)
		}
		if !strings.Contains(errors[0].Message, "DataContent次数总和=5") {
			t.Errorf("错误消息应包含 DataContent次数总和=5，实际: %s", errors[0].Message)
		}
	}
}

// TestValidateAssetsNumConsistency_EmptyAssetsNum 测试 AssetsNum 为空的情况
func TestValidateAssetsNumConsistency_EmptyAssetsNum(t *testing.T) {
	sheetConfig := parser.SheetConfig{
		SheetName: "0x31+0x06c0",
		FieldRules: []parser.FieldRule{
			{FieldName: "Field1"},
			{FieldName: "AssetsNum"},
			{FieldName: "DataInfoNum", Array: "array(4,5,6)"},
			{FieldName: "DataType", Loop: "loop(start=,)"},
			{FieldName: "DataLevel", Loop: "loop(active=,)"},
			{FieldName: "DataContent", Loop: "loop(end=,)"},
		},
		FieldNumberMap: map[string]int{
			"1": 0,
			"2": 1,
			"3": 2,
			"4": 3,
			"5": 4,
			"6": 5,
		},
	}

	// AssetsNum 为空，应该跳过校验
	line := "field1||3|1|1|1001,1|1|1|1002,2|1|1|1003,2"
	errors := validateAssetsNumConsistency(sheetConfig, line, 1, "test.txt", "|")

	if len(errors) != 0 {
		t.Errorf("AssetsNum为空时应跳过校验，但得到 %d 个错误", len(errors))
	} else {
		t.Log("测试通过：AssetsNum为空时跳过校验")
	}
}
