package validator

import (
	"net"
	"strings"
)

var (
	AllIPTypes   = []string{"ip", "ip_compressed", "ip_exploded", "ipv4", "ipv6", "ipv6_compressed", "ipv6_exploded"}
	AllIPv6Types = []string{"ipv6", "ipv6_compressed", "ipv6_exploded"}
)

func IsIPv4(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	// To4 返回 4 字节表示，如果是 IPv4 则非 nil
	if parsedIP.To4() == nil {
		return false
	}

	// 额外检查：确保没有前导零（如 "192.168.001.1"）
	// 使用字节扫描代替 strings.Split，避免内存分配
	dotCount := 0
	partStart := 0
	for i := 0; i < len(ip); i++ {
		if ip[i] == '.' {
			dotCount++
			// 检查当前段是否有前导零（段长度>1且首字符为'0'）
			partLen := i - partStart
			if partLen > 1 && ip[partStart] == '0' {
				return false
			}
			partStart = i + 1
		}
	}
	if dotCount != 3 {
		return false
	}
	// 检查最后一段
	lastPartLen := len(ip) - partStart
	if lastPartLen > 1 && ip[partStart] == '0' {
		return false
	}

	return true
}

func IsIPv6(ip string) bool {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}
	return parsedIP.To4() == nil && parsedIP.To16() != nil
}

func IsIPv6Compressed(ip string) bool {
	if !IsIPv6(ip) {
		return false
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	// 检查是否为压缩格式（包含::）
	return strings.Contains(ip, "::")
}

func IsIPv6Exploded(ip string) bool {
	if !IsIPv6(ip) {
		return false
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	// 检查是否为展开格式（不包含::，所有段都完整）
	return !strings.Contains(ip, "::") && strings.Count(ip, ":") == 7
}

func ValidIPAddress(ipType, ipAddr string) bool {
	switch ipType {
	case "ip":
		return IsIPv4(ipAddr) || IsIPv6(ipAddr)
	case "ip_compressed":
		return IsIPv4(ipAddr) || IsIPv6Compressed(ipAddr)
	case "ip_exploded":
		return IsIPv4(ipAddr) || IsIPv6Exploded(ipAddr)
	case "ipv4":
		return IsIPv4(ipAddr)
	case "ipv6":
		return IsIPv6(ipAddr)
	case "ipv6_compressed":
		return IsIPv6Compressed(ipAddr)
	case "ipv6_exploded":
		return IsIPv6Exploded(ipAddr)
	default:
		return false
	}
}

// 正则表达式模式匹配（简化版，Go版本使用标准库）
func MatchIPv4Pattern(ip string) bool {
	return IsIPv4(ip)
}


func MatchIPv6Pattern(ip string) bool {
	return IsIPv6(ip)
}
