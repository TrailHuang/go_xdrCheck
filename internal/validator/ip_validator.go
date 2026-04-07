package validator

import (
	"net"
	"strconv"
	"strings"
)

var (
	AllIPTypes   = []string{"ip", "ip_compressed", "ip_exploded", "ipv4", "ipv6", "ipv6_compressed", "ipv6_exploded"}
	AllIPv6Types = []string{"ipv6", "ipv6_compressed", "ipv6_exploded"}
)

func IsIPv4(ip string) bool {
	// 首先检查是否为有效的IPv4格式
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return false
	}

	// 检查每个部分是否为有效的数字（0-255）
	for _, part := range parts {
		if part == "" {
			return false
		}

		// 检查是否为纯数字
		for _, char := range part {
			if char < '0' || char > '9' {
				return false
			}
		}

		// 检查数字范围
		num, err := strconv.Atoi(part)
		if err != nil || num < 0 || num > 255 {
			return false
		}

		// 检查前导零（除了"0"本身）
		if len(part) > 1 && part[0] == '0' {
			return false
		}
	}

	// 使用标准库进行最终验证
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}
	return parsedIP.To4() != nil
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
