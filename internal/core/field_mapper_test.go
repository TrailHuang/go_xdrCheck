package core

import (
	"strconv"
	"testing"

	"xdrCheck/internal/parser"
)

// makeFieldNumberMap 构造 FieldNumberMap：字段序号 = fieldRules 索引（模拟实际 Excel 解析行为）
func makeFieldNumberMap(fieldRules []parser.FieldRule) map[string]int {
	fnm := make(map[string]int)
	for i, fr := range fieldRules {
		if fr.FieldName != "" {
			fnm[strconv.Itoa(i)] = i
		}
	}
	return fnm
}

// makeFieldNumberMapWithOffset 构造有偏移的 FieldNumberMap
// offset 表示字段序号与索引的偏移量（如序号从1开始，则 offset=1）
func makeFieldNumberMapWithOffset(fieldRules []parser.FieldRule, offset int) map[string]int {
	fnm := make(map[string]int)
	for i, fr := range fieldRules {
		if fr.FieldName != "" {
			fnm[strconv.Itoa(i+offset)] = i
		}
	}
	return fnm
}

// TestBuildFieldMapping_ArrayLoopRealLayout 模拟真实 0x06c0 的字段布局
// FieldNumberMap 序号与索引一致（如实际 Excel 解析结果）
func TestBuildFieldMapping_ArrayLoopRealLayout(t *testing.T) {
	// 真实 0x06c0 的字段规则（简化）
	fieldRules := []parser.FieldRule{
		{FieldName: "LogId"},       // 0
		{FieldName: "CommandID"},   // 1
		{FieldName: "House_ID"},    // 2
		{FieldName: "RuleId"},      // 3
		{FieldName: "Rule_Desc"},   // 4
		{FieldName: "AssetsIp"},    // 5
		{FieldName: "DataFileType"},// 6
		{FieldName: "AssetsSize"},  // 7
		{FieldName: "AssetsNum"},   // 8
		{FieldName: "DataInfoNum", Array: "array(10,11,12)"}, // 9 - array owner
		{FieldName: "DataType", Loop: "loop(start=,)"},       // 10
		{FieldName: "DataLevel", Loop: "loop(active=,)"},     // 11
		{FieldName: "DataContent", Loop: "loop(end=,)"},      // 12
		{FieldName: "IsUploadFile"},                           // 13
		{FieldName: "FileMD5"},                                // 14
		{FieldName: "CurTime"},                                // 15
		{FieldName: "SrcIP"},                                  // 16
		{FieldName: "DestIP"},                                 // 17
		{FieldName: "SrcPort"},                                // 18
		{FieldName: "DestPort"},                               // 19
		{FieldName: "ProtocolType"},                           // 20
		{FieldName: "ApplicationProtocol"},                    // 21
		{FieldName: "BusinessProtocol"},                       // 22
		{FieldName: "IsMatchEvent"},                           // 23
	}

	// FieldNumberMap: 序号与索引一致（实际 Excel 解析行为）
	fnm := makeFieldNumberMap(fieldRules)

	// 模拟 DataInfoNum=3, 3组 DataType/DataLevel/DataContent 的实际数据
	fields := []string{
		"log1", "cmd1", "house1", "rule1", "desc1", "ip1", "4", "0.39", "3",
		"3", // DataInfoNum=3 → 3组
		"1", "1", "1001,1",    // 第1组: DataType=1, DataLevel=1, DataContent=1001,1
		"1", "1", "1008,1",    // 第2组
		"1", "2", "1045,1",    // 第3组
		"0",   // IsUploadFile
		"md5hash", // FileMD5
		"1775807838",    // CurTime
		"125.73.8.33",   // SrcIP
		"180.142.128.84", // DestIP
		"38543",         // SrcPort
		"80",            // DestPort
		"1",             // ProtocolType
		"5",             // ApplicationProtocol
		"1",             // BusinessProtocol
		"1",             // IsMatchEvent
	}

	mappings := BuildFieldMapping(fields, fieldRules, fnm)

	// 验证固定字段
	fixedCases := []struct {
		mappingIdx int
		ruleIdx    int
		value      string
		repeatNo   int
	}{
		{0, 0, "log1", -1},
		{1, 1, "cmd1", -1},
		{2, 2, "house1", -1},
		{3, 3, "rule1", -1},
		{4, 4, "desc1", -1},
		{5, 5, "ip1", -1},
		{6, 6, "4", -1},
		{7, 7, "0.39", -1},
		{8, 8, "3", -1},
		{9, 9, "3", -1},  // DataInfoNum
	}
	for _, fc := range fixedCases {
		m := mappings[fc.mappingIdx]
		if m.RuleIndex != fc.ruleIdx {
			t.Errorf("fixed[%d]: RuleIndex=%d want %d", fc.mappingIdx, m.RuleIndex, fc.ruleIdx)
		}
		if m.Value != fc.value {
			t.Errorf("fixed[%d]: Value=%q want %q", fc.mappingIdx, m.Value, fc.value)
		}
		if m.RepeatNo != fc.repeatNo {
			t.Errorf("fixed[%d]: RepeatNo=%d want %d", fc.mappingIdx, m.RepeatNo, fc.repeatNo)
		}
	}

	// 验证3组重复字段
	groupCases := []struct {
		mappingIdx int
		ruleIdx    int
		value      string
		repeatNo   int
	}{
		{10, 10, "1", 0},
		{11, 11, "1", 0},
		{12, 12, "1001,1", 0},
		{13, 10, "1", 1},
		{14, 11, "1", 1},
		{15, 12, "1008,1", 1},
		{16, 10, "1", 2},
		{17, 11, "2", 2},
		{18, 12, "1045,1", 2},
	}
	for _, gc := range groupCases {
		m := mappings[gc.mappingIdx]
		if m.RuleIndex != gc.ruleIdx {
			t.Errorf("group[%d]: RuleIndex=%d want %d", gc.mappingIdx, m.RuleIndex, gc.ruleIdx)
		}
		if m.Value != gc.value {
			t.Errorf("group[%d]: Value=%q want %q", gc.mappingIdx, m.Value, gc.value)
		}
		if m.RepeatNo != gc.repeatNo {
			t.Errorf("group[%d]: RepeatNo=%d want %d", gc.mappingIdx, m.RepeatNo, gc.repeatNo)
		}
	}

	// 验证循环后的字段（关键：不应有偏移）
	postCases := []struct {
		mappingIdx int
		ruleIdx    int
		value      string
	}{
		{19, 13, "0"},
		{20, 14, "md5hash"},
		{21, 15, "1775807838"},
		{22, 16, "125.73.8.33"},
		{23, 17, "180.142.128.84"},
		{24, 18, "38543"},
		{25, 19, "80"},
		{26, 20, "1"},
		{27, 21, "5"},
		{28, 22, "1"},
		{29, 23, "1"},
	}
	for _, pc := range postCases {
		m := mappings[pc.mappingIdx]
		if m.RuleIndex != pc.ruleIdx {
			t.Errorf("post[%d]: RuleIndex=%d want %d (Value=%q)", pc.mappingIdx, m.RuleIndex, pc.ruleIdx, m.Value)
		}
		if m.Value != pc.value {
			t.Errorf("post[%d]: Value=%q want %q", pc.mappingIdx, m.Value, pc.value)
		}
		if m.RepeatNo != -1 {
			t.Errorf("post[%d]: RepeatNo=%d want -1", pc.mappingIdx, m.RepeatNo)
		}
	}
}

func TestBuildFieldMapping_ArrayLoopZeroGroups(t *testing.T) {
	fieldRules := []parser.FieldRule{
		{FieldName: "F0"},
		{FieldName: "DataInfoNum", Array: "array(3,4)"},
		{FieldName: "DataType", Loop: "loop(start=,)"},
		{FieldName: "DataContent", Loop: "loop(end=,)"},
		{FieldName: "F4"},
	}
	fnm := makeFieldNumberMapWithOffset(fieldRules, 1)

	fields := []string{"v0", "0", "v4"}
	mappings := BuildFieldMapping(fields, fieldRules, fnm)

	if len(mappings) != 5 {
		t.Fatalf("expected 5 mappings, got %d: %+v", len(mappings), mappings)
	}
	if mappings[0].RuleIndex != 0 || mappings[0].Value != "v0" { t.Errorf("F0 wrong") }
	if mappings[1].RuleIndex != 1 || mappings[1].Value != "0" { t.Errorf("DataInfoNum wrong") }
	if !mappings[2].Skipped || mappings[2].RuleIndex != 2 { t.Errorf("DataType should be skipped") }
	if !mappings[3].Skipped || mappings[3].RuleIndex != 3 { t.Errorf("DataContent should be skipped") }
	if mappings[4].RuleIndex != 4 || mappings[4].Value != "v4" { t.Errorf("F4 wrong") }
}

func TestBuildFieldMapping_NoDynamicRules(t *testing.T) {
	fieldRules := []parser.FieldRule{
		{FieldName: "F0"},
		{FieldName: "F1"},
		{FieldName: "F2"},
	}
	fnm := makeFieldNumberMap(fieldRules)
	fields := []string{"a", "b", "c"}
	mappings := BuildFieldMapping(fields, fieldRules, fnm)

	if len(mappings) != 3 { t.Fatalf("expected 3, got %d", len(mappings)) }
	for i, m := range mappings {
		if m.RuleIndex != i || m.FieldIndex != i || m.Value != fields[i] {
			t.Errorf("[%d] wrong: %+v", i, m)
		}
	}
}

func TestBuildFieldMapping_JumpZero(t *testing.T) {
	fieldRules := []parser.FieldRule{
		{FieldName: "F0"},
		{FieldName: "Ctrl", Jump: "jump=2"},
		{FieldName: "SkipA"},
		{FieldName: "SkipB"},
		{FieldName: "F4"},
	}
	fnm := makeFieldNumberMap(fieldRules)
	fields := []string{"a", "0", "f4"}
	mappings := BuildFieldMapping(fields, fieldRules, fnm)

	if mappings[0].Value != "a" { t.Errorf("F0 wrong") }
	if mappings[1].Value != "0" { t.Errorf("Ctrl wrong") }
	if !mappings[2].Skipped || mappings[2].RuleIndex != 2 { t.Errorf("SkipA not skipped") }
	if !mappings[3].Skipped || mappings[3].RuleIndex != 3 { t.Errorf("SkipB not skipped") }
	if mappings[4].Value != "f4" { t.Errorf("F4 wrong: %+v", mappings[4]) }
}

func TestBuildFieldMapping_JumpArray(t *testing.T) {
	// jump=2;array(2,3): owner ruleIdx=0, array 控制 ruleIdx[2,3]
	fieldRules := []parser.FieldRule{
		{FieldName: "Ctrl", Jump: "jump=2", Array: "array(2,3)"},
		{FieldName: "ItemA"},
		{FieldName: "ItemB"},
		{FieldName: "F3"},
	}
	// FieldNumberMap: 序号直接等于索引
	fnm := makeFieldNumberMap(fieldRules)

	fields := []string{"2", "a1", "b1", "a2", "b2", "f3"}
	mappings := BuildFieldMapping(fields, fieldRules, fnm)

	if mappings[0].Value != "2" { t.Errorf("Ctrl wrong") }
	jumpCases := []struct {
		idx     int
		ruleIdx int
		value   string
		repeatNo int
	}{
		{1, 1, "a1", 0},
		{2, 2, "b1", 0},
		{3, 1, "a2", 1},
		{4, 2, "b2", 1},
	}
	for _, jc := range jumpCases {
		m := mappings[jc.idx]
		if m.RuleIndex != jc.ruleIdx { t.Errorf("[%d]: RuleIndex=%d want %d", jc.idx, m.RuleIndex, jc.ruleIdx) }
		if m.Value != jc.value { t.Errorf("[%d]: Value=%s want %s", jc.idx, m.Value, jc.value) }
		if m.RepeatNo != jc.repeatNo { t.Errorf("[%d]: RepeatNo=%d want %d", jc.idx, m.RepeatNo, jc.repeatNo) }
	}
	if mappings[5].Value != "f3" { t.Errorf("F3 wrong: %+v", mappings[5]) }
}

func TestHasDynamicRules(t *testing.T) {
	tests := []struct {
		name     string
		rules    []parser.FieldRule
		expected bool
	}{
		{"no dynamic", []parser.FieldRule{{FieldName: "A"}, {FieldName: "B"}}, false},
		{"has jump", []parser.FieldRule{{FieldName: "A", Jump: "jump=1"}}, true},
		{"has array", []parser.FieldRule{{FieldName: "A", Array: "array(2,3)"}}, true},
		{"has loop", []parser.FieldRule{{FieldName: "A", Loop: "loop(start=,)"}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasDynamicRules(tt.rules)
			if result != tt.expected { t.Errorf("expected %v, got %v", tt.expected, result) }
		})
	}
}
