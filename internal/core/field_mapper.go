package core

import (
	"strconv"
	"strings"

	"xdrCheck/internal/parser"
)

// FieldMapping 表示字段规则到实际字段值的映射
// 用于处理 jump/array/loop 等动态字段映射规则
type FieldMapping struct {
	RuleIndex  int    // FieldRules 中的索引
	FieldIndex int    // 实际字段数组中的索引（-1 表示字段不存在/被跳过）
	Value      string // 字段值
	Skipped    bool   // 是否因 jump 规则被跳过（字段在数据中不存在）
	RepeatNo   int    // 重复字段的第几次重复（0-based），-1 表示非重复字段
}

// ArrayControl 描述 array 规则控制的字段组
// 当字段F有 array(A,B,C) 时，表示 F 的值控制 A,B,C 这组字段的重复次数
type ArrayControl struct {
	OwnerRuleIdx int   // 带 array 规则的字段在 FieldRules 中的索引
	GroupIndices []int // 被控制的字段组在 FieldRules 中的索引（有序）
	GroupSize    int   // 组内字段数量
}

// HasDynamicRules 检查字段规则中是否包含动态映射规则（jump/array/loop）
func HasDynamicRules(fieldRules []parser.FieldRule) bool {
	for _, rule := range fieldRules {
		if rule.Jump != "" || rule.Array != "" || rule.Loop != "" {
			return true
		}
	}
	return false
}

// BuildFieldMapping 根据字段规则和实际字段值构建映射关系
//
// 核心语义：
//   - array(N,M,K) 定义在字段F上: 字段F的值 = 字段N,M,K这组的重复次数
//   - loop(start=,) / loop(active=,) / loop(end=,): 配合array标记循环边界
//   - jump=N: 当前值为0时跳过后续N个字段；值非0时(配合array)重复N次
func BuildFieldMapping(fields []string, fieldRules []parser.FieldRule, fieldNumberMap map[string]int) []FieldMapping {
	// 预扫描：构建 array 控制映射（需要 fieldNumberMap 来转换字段序号）
	arrayControls := buildArrayControlMap(fieldRules, fieldNumberMap)

	var mappings []FieldMapping
	fieldIdx := 0

	// 用于展开 array 组时跳过已处理的规则索引
	skipUntilRuleIdx := -1

	for ruleIdx := 0; ruleIdx < len(fieldRules); ruleIdx++ {
		rule := fieldRules[ruleIdx]

		// 如果当前规则索引在已展开的组范围内，跳过
		if ruleIdx <= skipUntilRuleIdx {
			// 该规则已在 array 展开时被处理（通过映射加入），不再处理
			continue
		}

		if fieldIdx > len(fields) {
			// 数据已耗尽，剩余规则标记为跳过
			mappings = append(mappings, FieldMapping{
				RuleIndex:  ruleIdx,
				FieldIndex: -1,
				Value:      "",
				Skipped:    true,
			})
			continue
		}

		// === 处理 jump 规则 ===
		if rule.Jump != "" {
			jumpCount := parseJumpCount(rule.Jump)
			value := ""
			if fieldIdx < len(fields) {
				value = strings.TrimSpace(fields[fieldIdx])
			}

			// 映射 jump 字段本身
			mappings = append(mappings, FieldMapping{
				RuleIndex:  ruleIdx,
				FieldIndex: fieldIdx,
				Value:      value,
				RepeatNo:   -1,
			})
			if fieldIdx < len(fields) {
				fieldIdx++
			}

			if value == "0" {
				// 值为0：后续 jumpCount 个字段在数据中不存在
				for j := 1; j <= jumpCount && ruleIdx+j < len(fieldRules); j++ {
					mappings = append(mappings, FieldMapping{
						RuleIndex:  ruleIdx + j,
						FieldIndex: -1,
						Value:      "",
						Skipped:    true,
					})
				}
				ruleIdx += jumpCount
			} else if rule.Array != "" {
				// 值非0 + 有 array: 后续字段构成重复组
				repeatCount, _ := strconv.Atoi(value)
				for r := 0; r < repeatCount; r++ {
					for j := 1; j <= jumpCount && ruleIdx+j < len(fieldRules); j++ {
						if fieldIdx < len(fields) {
							arrValue := strings.TrimSpace(fields[fieldIdx])
							mappings = append(mappings, FieldMapping{
								RuleIndex:  ruleIdx + j,
								FieldIndex: fieldIdx,
								Value:      arrValue,
								RepeatNo:   r,
							})
							fieldIdx++
						}
					}
				}
				ruleIdx += jumpCount
			}
			// 否则仅有jump且值非0：后续字段正常出现，不做特殊处理
			continue
		}

		// === 检查当前字段是否是某个 array control 的 owner ===
		if ctrl, isArrayOwner := arrayControls[ruleIdx]; isArrayOwner {
			// 读取控制字段（owner）的值作为重复次数
			value := ""
			if fieldIdx < len(fields) {
				value = strings.TrimSpace(fields[fieldIdx])
			}

			// 映射控制字段自身（只出现一次）
			mappings = append(mappings, FieldMapping{
				RuleIndex:  ruleIdx,
				FieldIndex: fieldIdx,
				Value:      value,
				RepeatNo:   -1,
			})
			if fieldIdx < len(fields) {
				fieldIdx++
			}

			repeatCount, _ := strconv.Atoi(value)
			if repeatCount > 0 {
				// 展开整个组：按行优先（先第1次所有字段，再第2次...）
				for r := 0; r < repeatCount; r++ {
					for _, groupRuleIdx := range ctrl.GroupIndices {
						if fieldIdx < len(fields) {
							groupValue := strings.TrimSpace(fields[fieldIdx])
							mappings = append(mappings, FieldMapping{
								RuleIndex:  groupRuleIdx,
								FieldIndex: fieldIdx,
								Value:      groupValue,
								RepeatNo:   r,
							})
							fieldIdx++
						} else {
							// 数据不足，标记跳过
							mappings = append(mappings, FieldMapping{
								RuleIndex:  groupRuleIdx,
								FieldIndex: -1,
								Value:      "",
								Skipped:    true,
								RepeatNo:   r,
							})
						}
					}
				}
				// 跳过所有已展开的组规则索引
				// 这样后续循环到这些 ruleIdx 时会直接 continue
				skipUntilRuleIdx = ctrl.GroupIndices[len(ctrl.GroupIndices)-1]
			} else {
				// 重复次数为0或无效：跳过整个组
				for _, gIdx := range ctrl.GroupIndices {
					mappings = append(mappings, FieldMapping{
						RuleIndex:  gIdx,
						FieldIndex: -1,
						Value:      "",
						Skipped:    true,
					})
				}
				skipUntilRuleIdx = ctrl.GroupIndices[len(ctrl.GroupIndices)-1]
			}
			continue
		}

		// === 普通字段处理 ===
		if fieldIdx < len(fields) {
			value := strings.TrimSpace(fields[fieldIdx])
			mappings = append(mappings, FieldMapping{
				RuleIndex:  ruleIdx,
				FieldIndex: fieldIdx,
				Value:      value,
				RepeatNo:   -1,
			})
			fieldIdx++
		} else {
			mappings = append(mappings, FieldMapping{
				RuleIndex:  ruleIdx,
				FieldIndex: -1,
				Value:      "",
				Skipped:    true,
			})
		}
	}

	return mappings
}

// buildArrayControlMap 预扫描字段规则，构建 array 控制映射
// 返回: { ownerRuleIdx → ArrayControl }
//
// 解析 array(N,M,K) 格式，其中 N,M,K 是 Excel 中的字段序号
// 需要通过 fieldNumberMap 转换为 FieldRules 中的索引
func buildArrayControlMap(fieldRules []parser.FieldRule, fieldNumberMap map[string]int) map[int]ArrayControl {
	result := make(map[int]ArrayControl)

	for ruleIdx, rule := range fieldRules {
		if rule.Array == "" {
			continue
		}

		// 解析 array(N,M,K,...)
		inner := strings.TrimPrefix(rule.Array, "array(")
		inner = strings.TrimSuffix(inner, ")")
		parts := strings.Split(inner, ",")

		var groupIndices []int
		for _, p := range parts {
			fieldNum := strings.TrimSpace(p)
			// 通过 fieldNumberMap 将字段序号转换为 fieldRules 索引
			if mappedIdx, ok := fieldNumberMap[fieldNum]; ok {
				groupIndices = append(groupIndices, mappedIdx)
			}
		}

		if len(groupIndices) > 0 {
			result[ruleIdx] = ArrayControl{
				OwnerRuleIdx: ruleIdx,
				GroupIndices: groupIndices,
				GroupSize:    len(groupIndices),
			}
		}
	}

	return result
}

// parseJumpCount 从 jump 规则中解析跳过数量
func parseJumpCount(jumpRule string) int {
	parts := strings.SplitN(jumpRule, "=", 2)
	if len(parts) != 2 {
		return 0
	}
	count, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0
	}
	return count
}
