package validator

import (
	"regexp"
	"sync"
)

var (
	regexCache = make(map[string]*regexp.Regexp)
	regexMu    sync.RWMutex
)

// CompileRegex 编译正则表达式并缓存
func CompileRegex(pattern string) (*regexp.Regexp, error) {
	regexMu.RLock()
	re, exists := regexCache[pattern]
	regexMu.RUnlock()

	if exists {
		return re, nil
	}

	regexMu.Lock()
	defer regexMu.Unlock()

	// 双重检查,避免并发写入
	re, exists = regexCache[pattern]
	if exists {
		return re, nil
	}

	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	regexCache[pattern] = compiled
	return compiled, nil
}

// GetRegex 从缓存中获取正则表达式
func GetRegex(pattern string) (*regexp.Regexp, error) {
	regexMu.RLock()
	re, exists := regexCache[pattern]
	regexMu.RUnlock()

	if !exists {
		return CompileRegex(pattern)
	}

	return re, nil
}

// ClearRegexCache 清空正则表达式缓存(用于测试)
func ClearRegexCache() {
	regexMu.Lock()
	defer regexMu.Unlock()
	regexCache = make(map[string]*regexp.Regexp)
}
