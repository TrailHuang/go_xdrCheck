package validator

import (
	"sync"
	"testing"
)

// ============== CompileRegex 测试 ==============

func TestCompileRegex_ValidPattern(t *testing.T) {
	ClearRegexCache()

	pattern := `^[a-z]+$`
	re, err := CompileRegex(pattern)
	if err != nil {
		t.Fatalf("CompileRegex(%s) 失败: %v", pattern, err)
	}

	if re == nil {
		t.Fatal("CompileRegex 返回 nil")
	}

	if !re.MatchString("abc") {
		t.Error("正则表达式应匹配 'abc'")
	}

	if re.MatchString("123") {
		t.Error("正则表达式不应匹配 '123'")
	}
}

func TestCompileRegex_InvalidPattern(t *testing.T) {
	ClearRegexCache()

	invalidPatterns := []string{
		"[",
		"(",
		"*",
		"\\",
	}

	for _, pattern := range invalidPatterns {
		_, err := CompileRegex(pattern)
		if err == nil {
			t.Errorf("CompileRegex(%s) 应返回错误", pattern)
		}
	}
}

func TestCompileRegex_EmptyPattern(t *testing.T) {
	ClearRegexCache()

	re, err := CompileRegex("")
	if err != nil {
		t.Fatalf("CompileRegex(\"\") 应成功: %v", err)
	}

	if re == nil {
		t.Fatal("CompileRegex 返回 nil")
	}

	// 空正则表达式匹配任何字符串
	if !re.MatchString("") {
		t.Error("空正则表达式应匹配空字符串")
	}

	if !re.MatchString("abc") {
		t.Error("空正则表达式应匹配任何字符串")
	}
}

func TestCompileRegex_ComplexPattern(t *testing.T) {
	ClearRegexCache()

	complexPatterns := []string{
		`^[\w.-]+@[\w.-]+\.\w+$`,
		`^(https?|ftp)://[^\s/$.?#].[^\s]*$`,
		`^(\+\d{1,3}[- ]?)?\d{10,11}$`,
		`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`,
	}

	for _, pattern := range complexPatterns {
		re, err := CompileRegex(pattern)
		if err != nil {
			t.Errorf("CompileRegex(%s) 失败: %v", pattern, err)
		}
		if re == nil {
			t.Errorf("CompileRegex(%s) 返回 nil", pattern)
		}
	}
}

func TestCompileRegex_Caching(t *testing.T) {
	ClearRegexCache()

	pattern := `^test$`

	re1, err := CompileRegex(pattern)
	if err != nil {
		t.Fatalf("第一次 CompileRegex 失败: %v", err)
	}

	re2, err := CompileRegex(pattern)
	if err != nil {
		t.Fatalf("第二次 CompileRegex 失败: %v", err)
	}

	if re1 != re2 {
		t.Error("相同的正则表达式应返回相同的缓存实例")
	}
}

func TestCompileRegex_ConcurrentAccess(t *testing.T) {
	ClearRegexCache()

	pattern := `^[a-z]+$`
	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			re, err := CompileRegex(pattern)
			if err != nil {
				t.Errorf("并发 CompileRegex 失败: %v", err)
				return
			}
			if re == nil {
				t.Error("并发 CompileRegex 返回 nil")
			}
		}()
	}

	wg.Wait()
}

// ============== GetRegex 测试 ==============

func TestGetRegex_FromCache(t *testing.T) {
	ClearRegexCache()

	pattern := `^\d+$`

	re1, err := CompileRegex(pattern)
	if err != nil {
		t.Fatalf("CompileRegex 失败: %v", err)
	}

	re2, err := GetRegex(pattern)
	if err != nil {
		t.Fatalf("GetRegex 失败: %v", err)
	}

	if re1 != re2 {
		t.Error("GetRegex 应返回缓存的正则表达式")
	}
}

func TestGetRegex_NotInCache(t *testing.T) {
	ClearRegexCache()

	pattern := `^[A-Z]+$`

	re, err := GetRegex(pattern)
	if err != nil {
		t.Fatalf("GetRegex 失败: %v", err)
	}

	if re == nil {
		t.Fatal("GetRegex 返回 nil")
	}

	if !re.MatchString("ABC") {
		t.Error("正则表达式应匹配 'ABC'")
	}
}

func TestGetRegex_InvalidPattern(t *testing.T) {
	ClearRegexCache()

	_, err := GetRegex("[")
	if err == nil {
		t.Error("GetRegex 对无效模式应返回错误")
	}
}

func TestGetRegex_ConcurrentAccess(t *testing.T) {
	ClearRegexCache()

	pattern := `^[a-z]+$`
	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			re, err := GetRegex(pattern)
			if err != nil {
				t.Errorf("并发 GetRegex 失败: %v", err)
				return
			}
			if re == nil {
				t.Error("并发 GetRegex 返回 nil")
			}
		}()
	}

	wg.Wait()
}

// ============== ClearRegexCache 测试 ==============

func TestClearRegexCache(t *testing.T) {
	pattern := `^test$`

	_, err := CompileRegex(pattern)
	if err != nil {
		t.Fatalf("CompileRegex 失败: %v", err)
	}

	regexMu.RLock()
	_, existsBefore := regexCache[pattern]
	regexMu.RUnlock()

	if !existsBefore {
		t.Error("编译后缓存应存在")
	}

	ClearRegexCache()

	regexMu.RLock()
	_, existsAfter := regexCache[pattern]
	regexMu.RUnlock()

	if existsAfter {
		t.Error("ClearRegexCache 后缓存应为空")
	}

	re, err := GetRegex(pattern)
	if err != nil {
		t.Fatalf("ClearRegexCache 后 GetRegex 失败: %v", err)
	}

	if re == nil {
		t.Error("GetRegex 应重新编译正则表达式")
	}
}

func TestClearRegexCache_ConcurrentAccess(t *testing.T) {
	ClearRegexCache()

	var wg sync.WaitGroup
	numGoroutines := 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(3)

		go func(n int) {
			defer wg.Done()
			pattern := `^test` + string(rune('a'+n%26)) + `$`
			_, _ = CompileRegex(pattern)
		}(i)

		go func() {
			defer wg.Done()
			ClearRegexCache()
		}()

		go func() {
			defer wg.Done()
			_, _ = GetRegex(`^test$`)
		}()
	}

	wg.Wait()
}

// ============== 集成测试 ==============

func TestRegexCache_Integration(t *testing.T) {
	ClearRegexCache()

	patterns := []string{
		`^[a-z]+$`,
		`^\d+$`,
		`^[A-Z]+$`,
		`^[\w.-]+@[\w.-]+\.\w+$`,
	}

	for _, pattern := range patterns {
		re1, err := CompileRegex(pattern)
		if err != nil {
			t.Fatalf("CompileRegex(%s) 失败: %v", pattern, err)
		}

		re2, err := GetRegex(pattern)
		if err != nil {
			t.Fatalf("GetRegex(%s) 失败: %v", pattern, err)
		}

		if re1 != re2 {
			t.Errorf("模式 %s 的缓存应返回相同实例", pattern)
		}
	}

	ClearRegexCache()

	for _, pattern := range patterns {
		re, err := GetRegex(pattern)
		if err != nil {
			t.Fatalf("ClearCache 后 GetRegex(%s) 失败: %v", pattern, err)
		}

		if re == nil {
			t.Errorf("ClearCache 后 GetRegex(%s) 应重新编译", pattern)
		}
	}
}

func TestRegexCache_MultiplePatterns(t *testing.T) {
	ClearRegexCache()

	pattern1 := `^[a-z]+$`
	pattern2 := `^\d+$`

	re1a, _ := CompileRegex(pattern1)
	re2a, _ := CompileRegex(pattern2)

	re1b, _ := GetRegex(pattern1)
	re2b, _ := GetRegex(pattern2)

	if re1a != re1b {
		t.Error("pattern1 的缓存应返回相同实例")
	}

	if re2a != re2b {
		t.Error("pattern2 的缓存应返回相同实例")
	}

	if re1a == re2a {
		t.Error("不同模式应返回不同实例")
	}
}

func TestRegexCache_RegexpMatch(t *testing.T) {
	ClearRegexCache()

	testCases := []struct {
		pattern string
		input   string
		should  bool
	}{
		{`^[a-z]+$`, "abc", true},
		{`^[a-z]+$`, "ABC", false},
		{`^\d+$`, "123", true},
		{`^\d+$`, "abc", false},
		{`^[\w.-]+@[\w.-]+\.\w+$`, "test@example.com", true},
		{`^[\w.-]+@[\w.-]+\.\w+$`, "invalid", false},
	}

	for _, tc := range testCases {
		re, err := CompileRegex(tc.pattern)
		if err != nil {
			t.Fatalf("CompileRegex(%s) 失败: %v", tc.pattern, err)
		}

		result := re.MatchString(tc.input)
		if result != tc.should {
			t.Errorf("模式 %s 匹配 %s 应返回 %v, 实际为 %v",
				tc.pattern, tc.input, tc.should, result)
		}
	}
}
