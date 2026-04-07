package validator

import (
	"testing"
)

// 测试基础数据类型验证
func TestValidateBasicTypes(t *testing.T) {
	tests := []struct {
		name       string
		fieldValue string
		dataType   string
		want       bool
		wantMsg    string
	}{
		// int类型测试
		{
			name:       "int有效",
			fieldValue: "123",
			dataType:   "int",
			want:       true,
			wantMsg:    "",
		},
		{
			name:       "int无效",
			fieldValue: "abc",
			dataType:   "int",
			want:       false,
			wantMsg:    "不是有效的整数",
		},
		{
			name:       "int负数",
			fieldValue: "-123",
			dataType:   "int",
			want:       true,
			wantMsg:    "",
		},

		// IPv4类型测试
		{
			name:       "IPv4有效",
			fieldValue: "192.168.1.1",
			dataType:   "ip",
			want:       true,
			wantMsg:    "",
		},
		{
			name:       "IPv4无效",
			fieldValue: "256.168.1.1",
			dataType:   "ip",
			want:       false,
			wantMsg:    "不是有效的IP地址（IPv4或IPv6）",
		},

		// IPv6类型测试
		{
			name:       "IPv6有效",
			fieldValue: "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			dataType:   "ip",
			want:       true,
			wantMsg:    "",
		},

		// datetime类型测试
		{
			name:       "datetime有效",
			fieldValue: "2023-01-01 12:00:00",
			dataType:   "datetime",
			want:       true,
			wantMsg:    "",
		},
		{
			name:       "datetime无效",
			fieldValue: "2023-13-01 12:00:00",
			dataType:   "datetime",
			want:       false,
			wantMsg:    "不是有效的日期时间格式 (yyyy-MM-dd HH:mm:ss)",
		},

		// base64类型测试
		{
			name:       "base64有效",
			fieldValue: "aGVsbG8=",
			dataType:   "base64",
			want:       true,
			wantMsg:    "",
		},
		{
			name:       "base64无效",
			fieldValue: "!!!",
			dataType:   "base64",
			want:       false,
			wantMsg:    "不是有效的Base64编码",
		},

		// json类型测试
		{
			name:       "json有效",
			fieldValue: `{"name":"test"}`,
			dataType:   "json",
			want:       true,
			wantMsg:    "",
		},
		{
			name:       "json无效",
			fieldValue: `{"name":"test"`,
			dataType:   "json",
			want:       false,
			wantMsg:    "不是有效的JSON格式",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rv := &RuleValidator{
				FieldValue: tt.fieldValue,
			}

			got, gotMsg := rv.ValidateType(tt.dataType)

			if got != tt.want {
				t.Errorf("ValidateType(%s) got = %v, want %v", tt.dataType, got, tt.want)
			}

			if gotMsg != tt.wantMsg {
				t.Errorf("ValidateType(%s) gotMsg = %v, want %v", tt.dataType, gotMsg, tt.wantMsg)
			}
		})
	}
}

// 测试长度验证规则
func TestValidateLengthRules(t *testing.T) {
	tests := []struct {
		name       string
		fieldValue string
		rule       string
		want       bool
		wantMsg    string
	}{
		// len_eq规则
		{
			name:       "len_eq匹配",
			fieldValue: "abc",
			rule:       "len=3",
			want:       true,
			wantMsg:    "",
		},
		{
			name:       "len_eq不匹配",
			fieldValue: "abc",
			rule:       "len=5",
			want:       false,
			wantMsg:    "长度应为5，实际为3",
		},

		// len_gt规则
		{
			name:       "len_gt匹配",
			fieldValue: "abcde",
			rule:       "len>3",
			want:       true,
			wantMsg:    "",
		},
		{
			name:       "len_gt不匹配",
			fieldValue: "abc",
			rule:       "len>5",
			want:       false,
			wantMsg:    "长度应大于5，实际为3",
		},

		// len_lt规则
		{
			name:       "len_lt匹配",
			fieldValue: "abc",
			rule:       "len<5",
			want:       true,
			wantMsg:    "",
		},
		{
			name:       "len_lt不匹配",
			fieldValue: "abcde",
			rule:       "len<3",
			want:       false,
			wantMsg:    "长度应小于3，实际为5",
		},

		// len_ge规则
		{
			name:       "len_ge匹配等于",
			fieldValue: "abc",
			rule:       "len>=3",
			want:       true,
			wantMsg:    "",
		},
		{
			name:       "len_ge匹配大于",
			fieldValue: "abcde",
			rule:       "len>=3",
			want:       true,
			wantMsg:    "",
		},
		{
			name:       "len_ge不匹配",
			fieldValue: "ab",
			rule:       "len>=3",
			want:       false,
			wantMsg:    "长度应大于等于3，实际为2",
		},

		// len_le规则
		{
			name:       "len_le匹配等于",
			fieldValue: "abc",
			rule:       "len<=3",
			want:       true,
			wantMsg:    "",
		},
		{
			name:       "len_le匹配小于",
			fieldValue: "ab",
			rule:       "len<=3",
			want:       true,
			wantMsg:    "",
		},
		{
			name:       "len_le不匹配",
			fieldValue: "abcde",
			rule:       "len<=3",
			want:       false,
			wantMsg:    "长度应小于等于3，实际为5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rv := &RuleValidator{
				FieldValue: tt.fieldValue,
			}

			got, gotMsg := rv.ValidateRule(tt.rule)

			if got != tt.want {
				t.Errorf("ValidateRule(%s) got = %v, want %v", tt.rule, got, tt.want)
			}

			if gotMsg != tt.wantMsg {
				t.Errorf("ValidateRule(%s) gotMsg = %v, want %v", tt.rule, gotMsg, tt.wantMsg)
			}
		})
	}
}

// 测试范围验证规则
func TestValidateRangeRules(t *testing.T) {
	tests := []struct {
		name       string
		fieldValue string
		rule       string
		want       bool
		wantMsg    string
	}{
		// range规则
		{
			name:       "range匹配最小值",
			fieldValue: "1",
			rule:       "1-5",
			want:       true,
			wantMsg:    "",
		},
		{
			name:       "range匹配中间值",
			fieldValue: "3",
			rule:       "1-5",
			want:       true,
			wantMsg:    "",
		},
		{
			name:       "range匹配最大值",
			fieldValue: "5",
			rule:       "1-5",
			want:       true,
			wantMsg:    "",
		},
		{
			name:       "range不匹配小于",
			fieldValue: "0",
			rule:       "1-5",
			want:       false,
			wantMsg:    "值应在1-5范围内，实际为0",
		},
		{
			name:       "range不匹配大于",
			fieldValue: "6",
			rule:       "1-5",
			want:       false,
			wantMsg:    "值应在1-5范围内，实际为6",
		},
		{
			name:       "range非数字",
			fieldValue: "abc",
			rule:       "1-5",
			want:       false,
			wantMsg:    "字段值不是有效数字",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rv := &RuleValidator{
				FieldValue: tt.fieldValue,
			}

			got, gotMsg := rv.ValidateRule(tt.rule)

			if got != tt.want {
				t.Errorf("ValidateRule(%s) got = %v, want %v", tt.rule, got, tt.want)
			}

			if gotMsg != tt.wantMsg {
				t.Errorf("ValidateRule(%s) gotMsg = %v, want %v", tt.rule, gotMsg, tt.wantMsg)
			}
		})
	}
}

// 测试正则表达式验证规则
func TestValidateRegexRules(t *testing.T) {
	tests := []struct {
		name       string
		fieldValue string
		rule       string
		want       bool
		wantMsg    string
	}{
		// reg规则
		{
			name:       "reg匹配",
			fieldValue: "hello123",
			rule:       "reg=[a-z0-9]+",
			want:       true,
			wantMsg:    "",
		},
		{
			name:       "reg不匹配",
			fieldValue: "HELLO",
			rule:       "reg=[a-z0-9]+",
			want:       false,
			wantMsg:    "字段值不符合正则表达式规则",
		},
		{
			name:       "reg空值",
			fieldValue: "",
			rule:       "reg=[a-z0-9]+",
			want:       false,
			wantMsg:    "字段值不符合正则表达式规则",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rv := &RuleValidator{
				FieldValue: tt.fieldValue,
			}

			got, gotMsg := rv.ValidateRule(tt.rule)

			if got != tt.want {
				t.Errorf("ValidateRule(%s) got = %v, want %v", tt.rule, got, tt.want)
			}

			if gotMsg != tt.wantMsg {
				t.Errorf("ValidateRule(%s) gotMsg = %v, want %v", tt.rule, gotMsg, tt.wantMsg)
			}
		})
	}
}

// 测试条件验证规则
func TestValidateConditionRules(t *testing.T) {
	tests := []struct {
		name       string
		fieldValue string
		allFields  []string
		rule       string
		want       bool
		wantMsg    string
	}{
		// if条件规则
		{
			name:       "if条件匹配",
			fieldValue: "8",
			allFields:  []string{"", "", "", "", "", "", "", "", "", "", "", "", "5"},
			rule:       "if($13==5,8)",
			want:       true,
			wantMsg:    "",
		},
		{
			name:       "if条件不匹配",
			fieldValue: "8",
			allFields:  []string{"", "", "", "", "", "", "", "", "", "", "", "", "10"},
			rule:       "if($13==5,8)",
			want:       true,
			wantMsg:    "",
		},
		{
			name:       "if条件字段值不匹配",
			fieldValue: "10",
			allFields:  []string{"", "", "", "", "", "", "", "", "", "", "", "", "5"},
			rule:       "if($13==5,8)",
			want:       true,
			wantMsg:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rv := &RuleValidator{
				FieldValue: tt.fieldValue,
				AllFields:  tt.allFields,
				FieldNumberMap: map[string]int{
					"13": 12, // $13对应索引12
				},
			}

			got, gotMsg := rv.ValidateCondition(tt.rule)

			if got != tt.want {
				t.Errorf("ValidateCondition(%s) got = %v, want %v", tt.rule, got, tt.want)
			}

			if gotMsg != tt.wantMsg {
				t.Errorf("ValidateCondition(%s) gotMsg = %v, want %v", tt.rule, gotMsg, tt.wantMsg)
			}
		})
	}
}

// 测试枚举验证规则（原有的测试用例）
func TestValidateEnum(t *testing.T) {
	tests := []struct {
		name       string
		fieldValue string
		rule       string
		want       bool
		wantMsg    string
	}{
		// 单个值匹配
		{
			name:       "单个值匹配",
			fieldValue: "5",
			rule:       "[5]",
			want:       true,
			wantMsg:    "",
		},
		{
			name:       "单个值不匹配",
			fieldValue: "5",
			rule:       "[10]",
			want:       false,
			wantMsg:    "字段值不在允许的范围内: 10",
		},

		// 逗号分隔的多个值
		{
			name:       "多值匹配第一个",
			fieldValue: "1",
			rule:       "[1,2,3]",
			want:       true,
			wantMsg:    "",
		},
		{
			name:       "多值匹配中间",
			fieldValue: "2",
			rule:       "[1,2,3]",
			want:       true,
			wantMsg:    "",
		},
		{
			name:       "多值匹配最后一个",
			fieldValue: "3",
			rule:       "[1,2,3]",
			want:       true,
			wantMsg:    "",
		},
		{
			name:       "多值不匹配",
			fieldValue: "5",
			rule:       "[1,2,3]",
			want:       false,
			wantMsg:    "字段值不在允许的范围内: 1,2,3",
		},

		// 范围格式
		{
			name:       "范围匹配最小值",
			fieldValue: "1",
			rule:       "[1-5]",
			want:       true,
			wantMsg:    "",
		},
		{
			name:       "范围匹配中间值",
			fieldValue: "3",
			rule:       "[1-5]",
			want:       true,
			wantMsg:    "",
		},
		{
			name:       "范围匹配最大值",
			fieldValue: "5",
			rule:       "[1-5]",
			want:       true,
			wantMsg:    "",
		},
		{
			name:       "范围不匹配小于最小值",
			fieldValue: "0",
			rule:       "[1-5]",
			want:       false,
			wantMsg:    "字段值不在允许的范围内: 1-5",
		},
		{
			name:       "范围不匹配大于最大值",
			fieldValue: "6",
			rule:       "[1-5]",
			want:       false,
			wantMsg:    "字段值不在允许的范围内: 1-5",
		},

		// 混合格式：范围 + 单个值
		{
			name:       "混合格式匹配范围",
			fieldValue: "3",
			rule:       "[1-5,10]",
			want:       true,
			wantMsg:    "",
		},
		{
			name:       "混合格式匹配单个值",
			fieldValue: "10",
			rule:       "[1-5,10]",
			want:       true,
			wantMsg:    "",
		},
		{
			name:       "混合格式不匹配",
			fieldValue: "7",
			rule:       "[1-5,10]",
			want:       false,
			wantMsg:    "字段值不在允许的范围内: 1-5,10",
		},

		// 边界情况
		{
			name:       "空字段值",
			fieldValue: "",
			rule:       "[1-5]",
			want:       false,
			wantMsg:    "字段值不在允许的范围内: 1-5",
		},
		{
			name:       "非数字值",
			fieldValue: "abc",
			rule:       "[1-5]",
			want:       false,
			wantMsg:    "字段值不在允许的范围内: 1-5",
		},
		{
			name:       "无效范围格式",
			fieldValue: "3",
			rule:       "[1-abc]",
			want:       false,
			wantMsg:    "字段值不在允许的范围内: 1-abc",
		},

		// 实际案例：ApplicationProtocol规则
		{
			name:       "ApplicationProtocol规则匹配",
			fieldValue: "5",
			rule:       "[1-41,9999]",
			want:       true,
			wantMsg:    "",
		},
		{
			name:       "ApplicationProtocol规则匹配9999",
			fieldValue: "9999",
			rule:       "[1-41,9999]",
			want:       true,
			wantMsg:    "",
		},
		{
			name:       "ApplicationProtocol规则不匹配",
			fieldValue: "42",
			rule:       "[1-41,9999]",
			want:       false,
			wantMsg:    "字段值不在允许的范围内: 1-41,9999",
		},
		{
			name:       "EventType规则匹配",
			fieldValue: "01101000",
			rule:       "[01101000,01102000,01103000,01104000,01105000,01106000,01107000,01108000,01109000,01199000,02001000,03001000,03002000,03003000,03004000,03005000,03006000,03007000,03099000,04101000,04102000,04199000,05001000,05002000,05003000,05004000,05005000,05099000,09000000]",
			want:       true,
			wantMsg:    "",
		},
		{
			name:       "EventType规则匹配",
			fieldValue: "01106000",
			rule:       "[01101000,01102000,01103000,01104000,01105000,01106000,01107000,01108000,01109000,01199000,02001000,03001000,03002000,03003000,03004000,03005000,03006000,03007000,03099000,04101000,04102000,04199000,05001000,05002000,05003000,05004000,05005000,05099000,09000000]",
			want:       true,
			wantMsg:    "",
		},
		{
			name:       "EventType规则匹配",
			fieldValue: "0212012",
			rule:       "[01101000,01102000,01103000,01104000,01105000,01106000,01107000,01108000,01109000,01199000,02001000,03001000,03002000,03003000,03004000,03005000,03006000,03007000,03099000,04101000,04102000,04199000,05001000,05002000,05003000,05004000,05005000,05099000,09000000]",
			want:       false,
			wantMsg:    "字段值不在允许的范围内: 01101000,01102000,01103000,01104000,01105000,01106000,01107000,01108000,01109000,01199000,02001000,03001000,03002000,03003000,03004000,03005000,03006000,03007000,03099000,04101000,04102000,04199000,05001000,05002000,05003000,05004000,05005000,05099000,09000000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rv := &RuleValidator{
				FieldValue: tt.fieldValue,
			}

			got, gotMsg := rv.validateEnum(tt.rule)

			if got != tt.want {
				t.Errorf("validateEnum() got = %v, want %v", got, tt.want)
			}

			if gotMsg != tt.wantMsg {
				t.Errorf("validateEnum() gotMsg = %v, want %v", gotMsg, tt.wantMsg)
			}
		})
	}
}

// 测试性能
func BenchmarkValidateEnum(b *testing.B) {
	rv := &RuleValidator{
		FieldValue: "5",
	}

	// 测试简单值
	b.Run("简单值", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			rv.validateEnum("[5]")
		}
	})

	// 测试范围
	b.Run("范围", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			rv.validateEnum("[1-41]")
		}
	})

	// 测试混合格式
	b.Run("混合格式", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			rv.validateEnum("[1-41,9999]")
		}
	})

	// 测试不匹配的情况
	b.Run("不匹配", func(b *testing.B) {
		rv := &RuleValidator{
			FieldValue: "100",
		}
		for i := 0; i < b.N; i++ {
			rv.validateEnum("[1-41,9999]")
		}
	})
}
