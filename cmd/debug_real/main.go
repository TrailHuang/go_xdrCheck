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
	sheetConfigs, err := parser.ParseExcelTemplate(templatePath)
	if err != nil {
		panic(err)
	}

	for _, sheet := range sheetConfigs {
		if !strings.Contains(sheet.SheetName, "06c0") {
			continue
		}

		fmt.Printf("=== Sheet: %s ===\n", sheet.SheetName)

		// 使用真实数据行
		realLine := "20260410003005155804000000173798|2041000020979|77901|1045|身份证号码|180.142.128.84|4|0.39|3|3|1|1|1001,1|1|1|1008,1|1|2|1045,1|0|A1481F33059DE5E00EC3368C74B1B198|1775807838|125.73.8.33|180.142.128.84|38543|80|1|5|1|1"
		fields := strings.Split(realLine, "|")

		fmt.Printf("Input fields (%d)\n\n", len(fields))

		mappings := core.BuildFieldMapping(fields, sheet.FieldRules, sheet.FieldNumberMap)
		fmt.Printf("Mappings: %d\n", len(mappings))
		for i, m := range mappings {
			skipped := ""
			if m.Skipped {
				skipped = " SKIP"
			}
			fr := sheet.FieldRules[m.RuleIndex]
			fmt.Printf("[%2d] RuleIdx=%2d(%-22s) FldIdx=%2d Val=%-30q RptNo=%d%s\n",
				i, m.RuleIndex, fr.FieldName, m.FieldIndex, m.Value, m.RepeatNo, skipped)
		}

		// 重点验证 CurTime 之后的字段映射
		fmt.Println("\n--- 验证关键字段 ---")
		for _, m := range mappings {
			fr := sheet.FieldRules[m.RuleIndex]
			switch fr.FieldName {
			case "CurTime", "SrcPort", "DestPort", "SrcIP", "DestIP",
				"ProtocolType", "ApplicationProtocol", "BusinessProtocol", "IsMatchEvent":
				fmt.Printf("  %-22s = %q (RuleIdx=%d, FldIdx=%d)\n",
					fr.FieldName, m.Value, m.RuleIndex, m.FieldIndex)
			}
		}
		return
	}
}
