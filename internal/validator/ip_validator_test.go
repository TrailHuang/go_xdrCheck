package validator

import (
	"testing"
)

// ============== IsIPv4 测试 ==============

func TestIsIPv4_ValidIPv4(t *testing.T) {
	validIPs := []string{
		"192.168.1.1",
		"10.0.0.1",
		"172.16.0.1",
		"255.255.255.255",
		"0.0.0.0",
		"127.0.0.1",
	}

	for _, ip := range validIPs {
		if !IsIPv4(ip) {
			t.Errorf("IsIPv4(%s) 应返回 true", ip)
		}
	}
}

func TestIsIPv4_InvalidIPv4(t *testing.T) {
	invalidIPs := []string{
		"192.168.001.1",
		"192.168.1.01",
		"01.02.03.04",
		"256.1.1.1",
		"1.1.1",
		"1.1.1.1.1",
		"abc.def.ghi.jkl",
		"",
		"192.168.1",
	}

	for _, ip := range invalidIPs {
		if IsIPv4(ip) {
			t.Errorf("IsIPv4(%s) 应返回 false", ip)
		}
	}
}

func TestIsIPv4_IPv6ShouldReturnFalse(t *testing.T) {
	ipv6Addresses := []string{
		"::1",
		"fe80::1",
		"2001:db8::1",
		"2001:0db8:0000:0000:0000:0000:0000:0001",
	}

	for _, ip := range ipv6Addresses {
		if IsIPv4(ip) {
			t.Errorf("IsIPv4(%s) 对 IPv6 应返回 false", ip)
		}
	}
}

// ============== IsIPv6 测试 ==============

func TestIsIPv6_ValidIPv6(t *testing.T) {
	validIPs := []string{
		"::1",
		"fe80::1",
		"2001:db8::1",
		"2001:0db8:0000:0000:0000:0000:0000:0001",
		"::",
	}

	for _, ip := range validIPs {
		if !IsIPv6(ip) {
			t.Errorf("IsIPv6(%s) 应返回 true", ip)
		}
	}
}

func TestIsIPv6_InvalidIPv6(t *testing.T) {
	invalidIPs := []string{
		"192.168.1.1",
		"10.0.0.1",
		"not_an_ip",
		"",
		"2001:db8:::1",
		"gggg::1",
	}

	for _, ip := range invalidIPs {
		if IsIPv6(ip) {
			t.Errorf("IsIPv6(%s) 应返回 false", ip)
		}
	}
}

// ============== IsIPv6Compressed 测试 ==============

func TestIsIPv6Compressed_Valid(t *testing.T) {
	compressedIPs := []string{
		"::1",
		"fe80::1",
		"2001:db8::1",
		"::",
	}

	for _, ip := range compressedIPs {
		if !IsIPv6Compressed(ip) {
			t.Errorf("IsIPv6Compressed(%s) 应返回 true", ip)
		}
	}
}

func TestIsIPv6Compressed_NotCompressed(t *testing.T) {
	explodedIPs := []string{
		"2001:0db8:0000:0000:0000:0000:0000:0001",
		"fe80:0000:0000:0000:0000:0000:0000:0001",
	}

	for _, ip := range explodedIPs {
		if IsIPv6Compressed(ip) {
			t.Errorf("IsIPv6Compressed(%s) 对非压缩格式应返回 false", ip)
		}
	}
}

func TestIsIPv6Compressed_NonIPv6(t *testing.T) {
	nonIPv6 := []string{
		"192.168.1.1",
		"not_an_ip",
		"",
	}

	for _, ip := range nonIPv6 {
		if IsIPv6Compressed(ip) {
			t.Errorf("IsIPv6Compressed(%s) 对非 IPv6 应返回 false", ip)
		}
	}
}

// ============== IsIPv6Exploded 测试 ==============

func TestIsIPv6Exploded_Valid(t *testing.T) {
	explodedIPs := []string{
		"2001:0db8:0000:0000:0000:0000:0000:0001",
		"fe80:0000:0000:0000:0000:0000:0000:0001",
	}

	for _, ip := range explodedIPs {
		if !IsIPv6Exploded(ip) {
			t.Errorf("IsIPv6Exploded(%s) 应返回 true", ip)
		}
	}
}

func TestIsIPv6Exploded_NotExploded(t *testing.T) {
	compressedIPs := []string{
		"::1",
		"fe80::1",
		"2001:db8::1",
	}

	for _, ip := range compressedIPs {
		if IsIPv6Exploded(ip) {
			t.Errorf("IsIPv6Exploded(%s) 对压缩格式应返回 false", ip)
		}
	}
}

func TestIsIPv6Exploded_NonIPv6(t *testing.T) {
	nonIPv6 := []string{
		"192.168.1.1",
		"not_an_ip",
		"",
	}

	for _, ip := range nonIPv6 {
		if IsIPv6Exploded(ip) {
			t.Errorf("IsIPv6Exploded(%s) 对非 IPv6 应返回 false", ip)
		}
	}
}

// ============== ValidIPAddress 测试 ==============

func TestValidIPAddress_IP(t *testing.T) {
	validIPs := []string{
		"192.168.1.1",
		"::1",
		"2001:db8::1",
	}

	for _, ip := range validIPs {
		if !ValidIPAddress("ip", ip) {
			t.Errorf("ValidIPAddress('ip', %s) 应返回 true", ip)
		}
	}
}

func TestValidIPAddress_IPCompressed(t *testing.T) {
	validIPs := []string{
		"192.168.1.1",
		"::1",
		"fe80::1",
	}

	for _, ip := range validIPs {
		if !ValidIPAddress("ip_compressed", ip) {
			t.Errorf("ValidIPAddress('ip_compressed', %s) 应返回 true", ip)
		}
	}

	invalidIPs := []string{
		"2001:0db8:0000:0000:0000:0000:0000:0001",
	}

	for _, ip := range invalidIPs {
		if ValidIPAddress("ip_compressed", ip) {
			t.Errorf("ValidIPAddress('ip_compressed', %s) 应返回 false", ip)
		}
	}
}

func TestValidIPAddress_IPExploded(t *testing.T) {
	validIPs := []string{
		"192.168.1.1",
		"2001:0db8:0000:0000:0000:0000:0000:0001",
	}

	for _, ip := range validIPs {
		if !ValidIPAddress("ip_exploded", ip) {
			t.Errorf("ValidIPAddress('ip_exploded', %s) 应返回 true", ip)
		}
	}

	invalidIPs := []string{
		"::1",
		"fe80::1",
	}

	for _, ip := range invalidIPs {
		if ValidIPAddress("ip_exploded", ip) {
			t.Errorf("ValidIPAddress('ip_exploded', %s) 应返回 false", ip)
		}
	}
}

func TestValidIPAddress_IPv4(t *testing.T) {
	validIPs := []string{
		"192.168.1.1",
		"10.0.0.1",
	}

	for _, ip := range validIPs {
		if !ValidIPAddress("ipv4", ip) {
			t.Errorf("ValidIPAddress('ipv4', %s) 应返回 true", ip)
		}
	}

	invalidIPs := []string{
		"::1",
		"2001:db8::1",
	}

	for _, ip := range invalidIPs {
		if ValidIPAddress("ipv4", ip) {
			t.Errorf("ValidIPAddress('ipv4', %s) 应返回 false", ip)
		}
	}
}

func TestValidIPAddress_IPv6(t *testing.T) {
	validIPs := []string{
		"::1",
		"2001:db8::1",
		"2001:0db8:0000:0000:0000:0000:0000:0001",
	}

	for _, ip := range validIPs {
		if !ValidIPAddress("ipv6", ip) {
			t.Errorf("ValidIPAddress('ipv6', %s) 应返回 true", ip)
		}
	}

	invalidIPs := []string{
		"192.168.1.1",
	}

	for _, ip := range invalidIPs {
		if ValidIPAddress("ipv6", ip) {
			t.Errorf("ValidIPAddress('ipv6', %s) 应返回 false", ip)
		}
	}
}

func TestValidIPAddress_IPv6Compressed(t *testing.T) {
	validIPs := []string{
		"::1",
		"fe80::1",
	}

	for _, ip := range validIPs {
		if !ValidIPAddress("ipv6_compressed", ip) {
			t.Errorf("ValidIPAddress('ipv6_compressed', %s) 应返回 true", ip)
		}
	}

	invalidIPs := []string{
		"192.168.1.1",
		"2001:0db8:0000:0000:0000:0000:0000:0001",
	}

	for _, ip := range invalidIPs {
		if ValidIPAddress("ipv6_compressed", ip) {
			t.Errorf("ValidIPAddress('ipv6_compressed', %s) 应返回 false", ip)
		}
	}
}

func TestValidIPAddress_IPv6Exploded(t *testing.T) {
	validIPs := []string{
		"2001:0db8:0000:0000:0000:0000:0000:0001",
	}

	for _, ip := range validIPs {
		if !ValidIPAddress("ipv6_exploded", ip) {
			t.Errorf("ValidIPAddress('ipv6_exploded', %s) 应返回 true", ip)
		}
	}

	invalidIPs := []string{
		"192.168.1.1",
		"::1",
		"fe80::1",
	}

	for _, ip := range invalidIPs {
		if ValidIPAddress("ipv6_exploded", ip) {
			t.Errorf("ValidIPAddress('ipv6_exploded', %s) 应返回 false", ip)
		}
	}
}

func TestValidIPAddress_InvalidType(t *testing.T) {
	if ValidIPAddress("invalid_type", "192.168.1.1") {
		t.Error("ValidIPAddress 对无效类型应返回 false")
	}
}

// ============== 正则表达式匹配测试 ==============

func TestMatchIPv4Pattern(t *testing.T) {
	validIPs := []string{
		"192.168.1.1",
		"10.0.0.1",
	}

	for _, ip := range validIPs {
		if !MatchIPv4Pattern(ip) {
			t.Errorf("MatchIPv4Pattern(%s) 应返回 true", ip)
		}
	}

	invalidIPs := []string{
		"::1",
		"not_an_ip",
	}

	for _, ip := range invalidIPs {
		if MatchIPv4Pattern(ip) {
			t.Errorf("MatchIPv4Pattern(%s) 应返回 false", ip)
		}
	}
}

func TestMatchIPv6Pattern(t *testing.T) {
	validIPs := []string{
		"::1",
		"2001:db8::1",
	}

	for _, ip := range validIPs {
		if !MatchIPv6Pattern(ip) {
			t.Errorf("MatchIPv6Pattern(%s) 应返回 true", ip)
		}
	}

	invalidIPs := []string{
		"192.168.1.1",
		"not_an_ip",
	}

	for _, ip := range invalidIPs {
		if MatchIPv6Pattern(ip) {
			t.Errorf("MatchIPv6Pattern(%s) 应返回 false", ip)
		}
	}
}

// ============== 常量测试 ==============

func TestAllIPTypes(t *testing.T) {
	expected := []string{"ip", "ip_compressed", "ip_exploded", "ipv4", "ipv6", "ipv6_compressed", "ipv6_exploded"}
	if len(AllIPTypes) != len(expected) {
		t.Errorf("AllIPTypes 长度应为 %d, 实际为 %d", len(expected), len(AllIPTypes))
	}

	for i, ipType := range expected {
		if AllIPTypes[i] != ipType {
			t.Errorf("AllIPTypes[%d] 应为 %s, 实际为 %s", i, ipType, AllIPTypes[i])
		}
	}
}

func TestAllIPv6Types(t *testing.T) {
	expected := []string{"ipv6", "ipv6_compressed", "ipv6_exploded"}
	if len(AllIPv6Types) != len(expected) {
		t.Errorf("AllIPv6Types 长度应为 %d, 实际为 %d", len(expected), len(AllIPv6Types))
	}

	for i, ipType := range expected {
		if AllIPv6Types[i] != ipType {
			t.Errorf("AllIPv6Types[%d] 应为 %s, 实际为 %s", i, ipType, AllIPv6Types[i])
		}
	}
}
