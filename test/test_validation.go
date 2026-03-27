package main

import (
	"fmt"

	"github.com/user/go_xdrCheck/validator"
)

func main() {
	fmt.Println("=== XDR验证器功能测试 ===")

	// 测试1: 类型验证
	testTypeValidation()

	// 测试2: 属性条件验证
	testConditionValidation()

	// 测试3: 校验规则验证
	testRuleValidation()

	// 测试4: 复合规则验证
	testComplexValidation()
}

// 测试类型验证功能
func testTypeValidation() {
	fmt.Println("\n--- 测试1: 类型验证 ---")

	testCases := []struct {
		fieldValue  string
		dataType    string
		expectPass  bool
		description string
	}{
		// int类型测试
		{"123", "int", true, "有效整数"},
		{"abc", "int", false, "无效整数"},
		{"", "int", true, "空值跳过"},

		// ip类型测试
		{"192.168.1.1", "ip", true, "有效IPv4"},
		{"2001:db8::1", "ip", true, "有效IPv6"},
		{"invalid", "ip", false, "无效IP"},

		// ipv4类型测试
		{"192.168.1.1", "ipv4", true, "有效IPv4"},
		{"2001:db8::1", "ipv4", false, "IPv6不匹配IPv4"},

		// ipv6类型测试
		{"2001:db8::1", "ipv6", true, "有效IPv6"},
		{"192.168.1.1", "ipv6", false, "IPv4不匹配IPv6"},

		// datetime类型测试
		{"2024-01-15 10:30:00", "datetime", true, "有效日期时间"},
		{"2024/01/15 10:30:00", "datetime", false, "无效日期格式"},

		// base64类型测试
		{"dGVzdA==", "base64", true, "有效Base64"},
		{"invalid", "base64", false, "无效Base64"},

		// string类型测试（任意字符都通过）
		{"任意内容", "string", true, "字符串类型"},
		{"", "string", true, "空字符串"},
	}

	for i, tc := range testCases {
		validator := validator.NewRuleValidator(tc.fieldValue, 0, []string{})
		valid, msg := validator.ValidateType(tc.dataType)

		status := "✅ 通过"
		if valid != tc.expectPass {
			status = "❌ 失败"
		}

		fmt.Printf("测试%d: %s - %s (期望: %v, 实际: %v, 消息: %s)\n",
			i+1, status, tc.description, tc.expectPass, valid, msg)
	}
}

// 测试属性条件验证
func testConditionValidation() {
	fmt.Println("\n--- 测试2: 属性条件验证 ---")

	testCases := []struct {
		fieldValue  string
		allFields   []string
		condition   string
		expectPass  bool
		description string
	}{
		// 等于条件测试
		{"必须填写", []string{"", "5", "test"}, "if($2==5)", true, "条件满足，字段有值"},
		{"", []string{"", "5", "test"}, "if($2==5)", false, "条件满足，字段为空"},
		{"", []string{"", "8", "test"}, "if($2==5)", true, "条件不满足，字段为空"},

		// 不等于条件测试
		{"必须填写", []string{"", "8", "test"}, "if($2!=5)", true, "条件满足，字段有值"},
		{"", []string{"", "8", "test"}, "if($2!=5)", false, "条件满足，字段为空"},
		{"", []string{"", "5", "test"}, "if($2!=5)", true, "条件不满足，字段为空"},

		// 多个值条件测试
		{"必须填写", []string{"", "8", "test"}, "if($2==5,8)", true, "多值条件满足，字段有值"},
		{"", []string{"", "8", "test"}, "if($2==5,8)", false, "多值条件满足，字段为空"},

		// 字符串条件测试
		{"必须填写", []string{"", "test", ""}, "if($2==\"test\")", true, "字符串条件满足"},
		{"", []string{"", "test", ""}, "if($2==\"test\")", false, "字符串条件满足但字段为空"},
	}

	for i, tc := range testCases {
		validator := validator.NewRuleValidator(tc.fieldValue, 0, tc.allFields)
		valid, msg := validator.ValidateCondition(tc.condition)

		status := "✅ 通过"
		if valid != tc.expectPass {
			status = "❌ 失败"
		}

		fmt.Printf("测试%d: %s - %s (期望: %v, 实际: %v, 消息: %s)\n",
			i+1, status, tc.description, tc.expectPass, valid, msg)
	}
}

// 测试校验规则验证
func testRuleValidation() {
	fmt.Println("\n--- 测试3: 校验规则验证 ---")

	testCases := []struct {
		fieldValue  string
		rule        string
		expectPass  bool
		description string
	}{
		// 长度规则测试
		{"12345", "len=5", true, "长度等于5"},
		{"1234", "len=5", false, "长度不等于5"},
		{"123456", "len>4", true, "长度大于4"},
		{"123", "len>4", false, "长度不大于4"},

		// 大小规则测试
		{"100", "size=100", true, "大小等于100"},
		{"99", "size=100", false, "大小不等于100"},
		{"101", "size>100", true, "大小大于100"},
		{"100", "size>100", false, "大小不大于100"},

		// 枚举规则测试
		{"100", "[100-200]", true, "在连续范围内"},
		{"50", "[100-200]", false, "不在连续范围内"},
		{"150", "[100-200,300]", true, "在混合范围内"},
		{"300", "[100-200,300]", true, "在枚举值内"},
		{"250", "[100-200,300]", false, "不在枚举范围内"},

		// 字符串枚举测试
		{"0110", "[0110,1010,0401]", true, "在字符串枚举内"},
		{"9999", "[0110,1010,0401]", false, "不在字符串枚举内"},

		// 正则规则测试
		{"12345", "reg=\\d+", true, "匹配数字正则"},
		{"abc", "reg=\\d+", false, "不匹配数字正则"},
		{"test123", "reg=\\w+", true, "匹配单词正则"},

		// 固定值测试
		{"65535", "[65535]", true, "匹配固定值"},
		{"65536", "[65535]", false, "不匹配固定值"},

		// 复杂字符串测试
		{"1|1|1001", "[1|1|1001,1|1|1002,1|1|1003]", true, "匹配复杂字符串"},
		{"1|1|9999", "[1|1|1001,1|1|1002,1|1|1003]", false, "不匹配复杂字符串"},
	}

	for i, tc := range testCases {
		validator := validator.NewRuleValidator(tc.fieldValue, 0, []string{})
		valid, msg := validator.ValidateRule(tc.rule)

		status := "✅ 通过"
		if valid != tc.expectPass {
			status = "❌ 失败"
		}

		fmt.Printf("测试%d: %s - %s (期望: %v, 实际: %v, 消息: %s)\n",
			i+1, status, tc.description, tc.expectPass, valid, msg)
	}
}

// 测试复合规则验证
func testComplexValidation() {
	fmt.Println("\n--- 测试4: 复合规则验证 ---")

	testCases := []struct {
		fieldValue  string
		allFields   []string
		typeRule    string
		condition   string
		rules       []string
		expectPass  bool
		description string
	}{
		{
			"192.168.1.1",
			[]string{"", "5", ""},
			"ip",
			"if($2==5)",
			[]string{"len>7", "len<16"},
			true,
			"IP类型+条件+长度规则",
		},
		{
			"",
			[]string{"", "5", ""},
			"ip",
			"if($2==5)",
			[]string{"len>7"},
			false,
			"条件满足但字段为空",
		},
		{
			"2024-01-15 10:30:00",
			[]string{"", "8", ""},
			"datetime",
			"if($2!=5)",
			[]string{"len=19"},
			true,
			"日期时间+条件+精确长度",
		},
		{
			"12345",
			[]string{"", "test", ""},
			"int",
			"if($2==\"test\")",
			[]string{"[10000-20000]"},
			true,
			"整数+条件+范围规则（条件不满足，跳过范围验证）",
		},
	}

	for i, tc := range testCases {
		validator := validator.NewRuleValidator(tc.fieldValue, 0, tc.allFields)

		// 验证类型
		typeValid, typeMsg := validator.ValidateType(tc.typeRule)

		// 验证条件
		condValid, condMsg := validator.ValidateCondition(tc.condition)

		// 验证规则
		rulesValid, rulesMsg := true, ""
		for _, rule := range tc.rules {
			valid, msg := validator.ValidateRule(rule)
			if !valid {
				rulesValid = false
				rulesMsg = msg
				break
			}
		}

		// 综合验证结果
		valid := typeValid && condValid && rulesValid

		status := "✅ 通过"
		if valid != tc.expectPass {
			status = "❌ 失败"
		}

		fmt.Printf("测试%d: %s - %s\n", i+1, status, tc.description)
		fmt.Printf("   类型验证: %v (%s)\n", typeValid, typeMsg)
		fmt.Printf("   条件验证: %v (%s)\n", condValid, condMsg)
		fmt.Printf("   规则验证: %v (%s)\n", rulesValid, rulesMsg)
		fmt.Printf("   综合结果: %v (期望: %v)\n\n", valid, tc.expectPass)
	}
}
