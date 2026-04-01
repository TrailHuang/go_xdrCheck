package validator

import (
	"testing"
)

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
