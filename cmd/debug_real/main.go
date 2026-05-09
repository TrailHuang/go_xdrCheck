package main

import (
	"fmt"
	"os"
	"strings"

	"xdrCheck/internal/core"
	"xdrCheck/internal/parser"
)

func main() {
	templatePath := os.Args[1]
	sheetConfigs, err := parser.ParseExcelTemplate(templatePath, nil)
	if err != nil {
		panic(err)
	}

	for _, sheet := range sheetConfigs {
		if !strings.Contains(sheet.SheetName, "06c0") {
			continue
		}

		fmt.Printf("=== Sheet: %s ===\n", sheet.SheetName)

		realLine := "20260410003005155804000000173798|2041000020979|77901|1045|身份证号码|180.142.128.84|4|0.39|3|3|1|1|1001,1|1|1|1008,1|1|2|1045,1|0|A1481F33059DE5E00EC3368C74B1B198|1775807838|125.73.8.33|180.142.128.84|38543|80|1|5|1|1"
		fields := strings.Split(realLine, "|")

		mappings := core.BuildFieldMapping(fields, sheet.FieldRules, sheet.FieldNumberMap)
		core.BuildCompoundValues(mappings, sheet.FieldRules)

		fmt.Println("\n--- DataContent mappings with compound values ---")
		for i, m := range mappings {
			fr := sheet.FieldRules[m.RuleIndex]
			if fr.FieldName == "DataContent" || fr.FieldName == "DataType" || fr.FieldName == "DataLevel" {
				compound := ""
				if m.CompoundValue != "" {
					compound = " CompoundValue=" + m.CompoundValue
				}
				fmt.Printf("[%2d] %s Val=%q RptNo=%d%s\n",
					i, fr.FieldName, m.Value, m.RepeatNo, compound)
			}
		}

		// 测试枚举匹配
		fmt.Println("\n--- Enum matching test ---")
		for _, m := range mappings {
			fr := sheet.FieldRules[m.RuleIndex]
			if fr.FieldName != "DataContent" {
				continue
			}
			enumValue := m.Value
			if m.CompoundValue != "" && len(fr.Rules) > 0 && strings.HasPrefix(fr.Rules[0], "[") && strings.Contains(fr.Rules[0], "|") {
				enumValue = m.CompoundValue
			}
			fmt.Printf("DataContent[RptNo=%d] Value=%q CompoundValue=%q EnumMatchValue=%q\n",
				m.RepeatNo, m.Value, m.CompoundValue, enumValue)

			// 手动测试匹配
			if len(fr.Rules) > 0 && fr.ParsedEnums != nil {
				pe := fr.ParsedEnums[fr.Rules[0]]
				if pe != nil {
					if _, ok := pe.ExactValues[enumValue]; ok {
						fmt.Printf("  → MATCH in enum rule!\n")
					} else {
						fmt.Printf("  → NO MATCH (value %q not in enum)\n", enumValue)
						// 显示前几个枚举值
						count := 0
						for k := range pe.ExactValues {
							fmt.Printf("    enum has: %q\n", k)
							count++
							if count >= 5 {
								fmt.Printf("    ... (%d more)\n", len(pe.ExactValues)-5)
								break
							}
						}
					}
				}
			}
		}
		return
	}
}
