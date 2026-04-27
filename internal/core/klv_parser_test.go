package core

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"xdrCheck/internal/parser"
)

// ============== ByteReader 测试 ==============

func TestNewByteReader(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03}
	reader := NewByteReader(data)

	if reader == nil {
		t.Fatal("NewByteReader 返回 nil")
	}
	if reader.offset != 0 {
		t.Errorf("初始 offset 应为 0, 实际为 %d", reader.offset)
	}
	if reader.Remaining() != 3 {
		t.Errorf("初始剩余字节应为 3, 实际为 %d", reader.Remaining())
	}
}

func TestByteReader_ReadBytes(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	reader := NewByteReader(data)

	result, err := reader.ReadBytes(3)
	if err != nil {
		t.Fatalf("ReadBytes 失败: %v", err)
	}
	if len(result) != 3 {
		t.Errorf("读取长度应为 3, 实际为 %d", len(result))
	}
	if !bytes.Equal(result, []byte{0x01, 0x02, 0x03}) {
		t.Errorf("读取内容不匹配: %v", result)
	}
	if reader.offset != 3 {
		t.Errorf("offset 应为 3, 实际为 %d", reader.offset)
	}
	if reader.Remaining() != 2 {
		t.Errorf("剩余字节应为 2, 实际为 %d", reader.Remaining())
	}
}

func TestByteReader_ReadBytesEOF(t *testing.T) {
	data := []byte{0x01, 0x02}
	reader := NewByteReader(data)

	_, err := reader.ReadBytes(5)
	if err == nil {
		t.Error("读取超出长度时应返回 EOF 错误")
	}
}

func TestByteReader_ReadBytesExact(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03}
	reader := NewByteReader(data)

	result, err := reader.ReadBytes(3)
	if err != nil {
		t.Fatalf("ReadBytes 失败: %v", err)
	}
	if !bytes.Equal(result, data) {
		t.Errorf("读取内容不匹配")
	}
	if reader.Remaining() != 0 {
		t.Errorf("剩余字节应为 0, 实际为 %d", reader.Remaining())
	}
}

func TestByteReader_MultipleReads(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03, 0x04}
	reader := NewByteReader(data)

	r1, _ := reader.ReadBytes(2)
	if !bytes.Equal(r1, []byte{0x01, 0x02}) {
		t.Errorf("第一次读取不匹配")
	}

	r2, _ := reader.ReadBytes(2)
	if !bytes.Equal(r2, []byte{0x03, 0x04}) {
		t.Errorf("第二次读取不匹配")
	}
}

// ============== BinaryLogFields 测试 ==============

func TestBinaryLogFields_SetAndGetHeader(t *testing.T) {
	fields := &BinaryLogFields{}
	header := LogPrefixHeader{}
	copy(header.PrefixLog[:], []byte("test prefix log!"))

	fields.SetHeader(header)
	got := fields.GetHeader()
	if !bytes.Equal(got.PrefixLog[:], header.PrefixLog[:]) {
		t.Error("Header 设置和获取不匹配")
	}
}

func TestBinaryLogFields_ResetHeader(t *testing.T) {
	fields := &BinaryLogFields{}
	header := LogPrefixHeader{}
	copy(header.PrefixLog[:], []byte("test prefix log!"))
	fields.SetHeader(header)

	fields.ResetHeader()
	got := fields.GetHeader()
	if !bytes.Equal(got.PrefixLog[:], make([]byte, 16)) {
		t.Error("ResetHeader 后应为空")
	}
}

func TestBinaryLogFields_AddAndGetFields(t *testing.T) {
	fields := &BinaryLogFields{}

	if fields.FieldsNum != 0 {
		t.Errorf("初始 FieldsNum 应为 0, 实际为 %d", fields.FieldsNum)
	}

	fields.AddField(BField{Name: "Field1", Len: 4, Data: []byte{0x01, 0x02, 0x03, 0x04}, Value: "1024", Type: "uint32"})
	if fields.FieldsNum != 1 {
		t.Errorf("添加后 FieldsNum 应为 1, 实际为 %d", fields.FieldsNum)
	}

	gotFields := fields.GetFields()
	if len(gotFields) != 1 {
		t.Errorf("字段列表长度应为 1, 实际为 %d", len(gotFields))
	}
	if gotFields[0].Name != "Field1" {
		t.Errorf("字段名应为 Field1, 实际为 %s", gotFields[0].Name)
	}
}

func TestBinaryLogFields_Reset(t *testing.T) {
	fields := &BinaryLogFields{}
	header := LogPrefixHeader{}
	copy(header.PrefixLog[:], []byte("test prefix log!"))
	fields.SetHeader(header)
	fields.AddField(BField{Name: "Field1", Len: 4, Value: "test", Type: "string"})

	fields.Reset()

	if fields.FieldsNum != 0 {
		t.Errorf("Reset 后 FieldsNum 应为 0, 实际为 %d", fields.FieldsNum)
	}
	if len(fields.Fields) != 0 {
		t.Errorf("Reset 后 Fields 应为空, 实际长度为 %d", len(fields.Fields))
	}
}

// ============== 服务器信息解析测试 ==============

func TestParseServer_IPv4(t *testing.T) {
	data := make([]byte, 7)
	data[0] = 0 // IPv4
	copy(data[1:5], []byte{192, 168, 1, 1})
	binary.BigEndian.PutUint16(data[5:7], 8080)

	var offset uint32 = 0
	server, err := parseServer(data, &offset)
	if err != nil {
		t.Fatalf("parseServer 失败: %v", err)
	}
	if server.ServerIPType != 0 {
		t.Errorf("ServerIPType 应为 0, 实际为 %d", server.ServerIPType)
	}
	if !bytes.Equal(server.ServerIP, []byte{192, 168, 1, 1}) {
		t.Errorf("ServerIP 不匹配: %v", server.ServerIP)
	}
	if server.ServerPort != 8080 {
		t.Errorf("ServerPort 应为 8080, 实际为 %d", server.ServerPort)
	}
	if offset != 7 {
		t.Errorf("offset 应为 7, 实际为 %d", offset)
	}
}

func TestParseServer_IPv6(t *testing.T) {
	data := make([]byte, 19)
	data[0] = 1 // IPv6
	ipv6Addr := make([]byte, 16)
	for i := range ipv6Addr {
		ipv6Addr[i] = byte(i)
	}
	copy(data[1:17], ipv6Addr)
	binary.BigEndian.PutUint16(data[17:19], 443)

	var offset uint32 = 0
	server, err := parseServer(data, &offset)
	if err != nil {
		t.Fatalf("parseServer 失败: %v", err)
	}
	if server.ServerIPType != 1 {
		t.Errorf("ServerIPType 应为 1, 实际为 %d", server.ServerIPType)
	}
	if !bytes.Equal(server.ServerIP, ipv6Addr) {
		t.Errorf("ServerIP 不匹配")
	}
	if server.ServerPort != 443 {
		t.Errorf("ServerPort 应为 443, 实际为 %d", server.ServerPort)
	}
	if offset != 19 {
		t.Errorf("offset 应为 19, 实际为 %d", offset)
	}
}

func TestCheckAndParseServerInfo_SingleIPv4(t *testing.T) {
	data := make([]byte, 9)
	data[0] = 1 // 1 个服务器
	data[1] = 0 // IPv4
	copy(data[2:6], []byte{10, 0, 0, 1})
	binary.BigEndian.PutUint16(data[6:8], 9090)
	data[8] = 0 // 额外字节，确保长度 > 8

	result, err := CheckAndParseServerInfo(data)
	if err != nil {
		t.Fatalf("CheckAndParseServerInfo 失败: %v", err)
	}
	if result.Servers.ServerNum != 1 {
		t.Errorf("ServerNum 应为 1, 实际为 %d", result.Servers.ServerNum)
	}
	if len(result.Servers.Server) != 1 {
		t.Errorf("Server 列表长度应为 1, 实际为 %d", len(result.Servers.Server))
	}
	if result.PrefixLen != 8 {
		t.Errorf("PrefixLen 应为 8, 实际为 %d", result.PrefixLen)
	}
}

func TestCheckAndParseServerInfo_MultipleServers(t *testing.T) {
	data := make([]byte, 15)
	data[0] = 2 // 2 个服务器

	// 服务器 1: IPv4
	data[1] = 0
	copy(data[2:6], []byte{192, 168, 1, 1})
	binary.BigEndian.PutUint16(data[6:8], 80)

	// 服务器 2: IPv4
	data[8] = 0
	copy(data[9:13], []byte{10, 0, 0, 2})
	binary.BigEndian.PutUint16(data[13:15], 443)

	result, err := CheckAndParseServerInfo(data)
	if err != nil {
		t.Fatalf("CheckAndParseServerInfo 失败: %v", err)
	}
	if result.Servers.ServerNum != 2 {
		t.Errorf("ServerNum 应为 2, 实际为 %d", result.Servers.ServerNum)
	}
	if len(result.Servers.Server) != 2 {
		t.Errorf("Server 列表长度应为 2, 实际为 %d", len(result.Servers.Server))
	}
}

func TestCheckAndParseServerInfo_InvalidData(t *testing.T) {
	data := []byte{0x01, 0x02} // 数据太短
	_, err := CheckAndParseServerInfo(data)
	if err == nil {
		t.Error("数据太短时应返回错误")
	}
}

// ============== KLVParser 测试 ==============

func TestNewKLVParser(t *testing.T) {
	data := make([]byte, 100)
	parser := NewKLVParser(data)

	if parser == nil {
		t.Fatal("NewKLVParser 返回 nil")
	}
	if parser.formatDef == nil {
		t.Error("formatDef 不应为 nil")
	}
	if parser.fieldMap == nil {
		t.Error("fieldMap 不应为 nil")
	}
	if parser.reader == nil {
		t.Error("reader 不应为 nil")
	}
}

func TestNewKLVParserWithRules(t *testing.T) {
	data := make([]byte, 100)
	sheetConfig := parser.SheetConfig{
		SheetName: "test",
		FieldRules: []parser.FieldRule{
			{FieldName: "CommandID", Required: "必填", Type: "string"},
			{FieldName: "SrcIp", Required: "必填", Type: "ip"},
		},
	}

	p := NewKLVParserWithRules(data, sheetConfig)
	if p == nil {
		t.Fatal("NewKLVParserWithRules 返回 nil")
	}

	// 验证规则是否正确关联
	for _, fieldDef := range p.formatDef {
		if fieldDef.Name == "CommandID" && fieldDef.FieldRule == nil {
			t.Error("CommandID 应有关联的 FieldRule")
		}
		if fieldDef.Name == "SrcIp" && fieldDef.FieldRule == nil {
			t.Error("SrcIp 应有关联的 FieldRule")
		}
	}
}

func TestKLVParser_ResetForNewLog(t *testing.T) {
	data := make([]byte, 100)
	p := NewKLVParser(data)

	p.fieldMap["test"] = "value"
	p.ResetForNewLog()

	if len(p.fieldMap) != 0 {
		t.Errorf("ResetForNewLog 后 fieldMap 应为空, 实际长度为 %d", len(p.fieldMap))
	}
}

func TestKLVParser_ParseDone(t *testing.T) {
	data := make([]byte, 10)
	p := NewKLVParser(data)

	if p.ParseDone() {
		t.Error("有数据时 ParseDone 应返回 false")
	}

	p.reader.offset = 10
	if !p.ParseDone() {
		t.Error("数据读完时 ParseDone 应返回 true")
	}
}

func TestKLVParser_GetFields(t *testing.T) {
	data := make([]byte, 100)
	p := NewKLVParser(data)

	p.fields.AddField(BField{Name: "TestField", Len: 4, Value: "test", Type: "string"})
	fields := p.GetFields()

	if len(fields) != 1 {
		t.Errorf("字段列表长度应为 1, 实际为 %d", len(fields))
	}
	if fields[0].Name != "TestField" {
		t.Errorf("字段名应为 TestField, 实际为 %s", fields[0].Name)
	}
}

// ============== 类型转换测试 ==============

func TestConvertToInt_Bytes(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected int
	}{
		{"空字节", []byte{}, 0},
		{"单字节", []byte{0x05}, 5},
		{"双字节", []byte{0x01, 0x00}, 256},
		{"四字节", []byte{0x00, 0x00, 0x01, 0x00}, 256},
	}

	p := &KLVParser{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.convertToInt(tt.data)
			if result != tt.expected {
				t.Errorf("convertToInt(%v) = %d, want %d", tt.data, result, tt.expected)
			}
		})
	}
}

func TestConvertToInt_Interface(t *testing.T) {
	p := &KLVParser{}

	if p.convertToInt(int(100)) != 100 {
		t.Error("int 转换错误")
	}
	if p.convertToInt(uint8(50)) != 50 {
		t.Error("uint8 转换错误")
	}
	if p.convertToInt(uint16(1000)) != 1000 {
		t.Error("uint16 转换错误")
	}
	if p.convertToInt(uint32(50000)) != 50000 {
		t.Error("uint32 转换错误")
	}
	if p.convertToInt("invalid") != 0 {
		t.Error("无效类型应返回 0")
	}
}

func TestConvertFieldValue(t *testing.T) {
	tests := []struct {
		name      string
		valueType string
		data      []byte
		expected  string
	}{
		{"uint8", "uint8", []byte{0x05}, "5"},
		{"uint16", "uint16", []byte{0x01, 0x00}, "256"},
		{"string", "string", []byte("hello"), "hello"},
		{"ip", "ip", []byte{192, 168, 1, 1}, "192.168.1.1"},
		{"cmd", "cmd", []byte("CMD001"), "CMD001"},
	}

	p := &KLVParser{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.convertFieldValue(tt.valueType, tt.data)
			if result != tt.expected {
				t.Errorf("convertFieldValue(%s, %v) = %s, want %s",
					tt.valueType, tt.data, result, tt.expected)
			}
		})
	}
}

func TestConvertToIP(t *testing.T) {
	p := &KLVParser{}

	ipv4 := p.convertToIP([]byte{192, 168, 1, 1})
	if ipv4 != "192.168.1.1" {
		t.Errorf("IPv4 转换错误: %s", ipv4)
	}

	nonIP := p.convertToIP([]byte("hello"))
	if nonIP != "hello" {
		t.Errorf("非 IP 数据应返回字符串: %s", nonIP)
	}
}

func TestConvertToBase64(t *testing.T) {
	p := &KLVParser{}
	data := []byte("test base64 data")
	result := p.convertToBase64(data)
	if result != "test base64 data" {
		t.Errorf("Base64 转换错误: %s", result)
	}
}

// ============== 条件字段判断测试 ==============

func TestShouldIncludeField(t *testing.T) {
	p := &KLVParser{
		fieldMap: map[string]string{
			"Flag": string([]byte{0x00}),
		},
	}

	field := FieldDef{
		ConditionField: "Flag",
		ConditionValue: 0,
	}

	if !p.shouldIncludeField(field) {
		t.Error("条件满足时应返回 true")
	}

	field.ConditionValue = 1
	if p.shouldIncludeField(field) {
		t.Error("条件不满足时应返回 false")
	}
}

func TestShouldIncludeField_MissingCondition(t *testing.T) {
	p := &KLVParser{
		fieldMap: map[string]string{},
	}

	field := FieldDef{
		ConditionField: "Missing",
		ConditionValue: 1,
	}

	if p.shouldIncludeField(field) {
		t.Error("条件字段不存在时应返回 false")
	}
}

func TestShouldIncludeFieldExclude(t *testing.T) {
	p := &KLVParser{
		fieldMap: map[string]string{
			"Flag": string([]byte{0x00}),
		},
	}

	field := FieldDef{
		ConditionField: "Flag",
		ConditionValue: 1,
	}

	if !p.shouldIncludeFieldExclude(field) {
		t.Error("条件不满足时应返回 true (exclude)")
	}

	field.ConditionValue = 0
	if p.shouldIncludeFieldExclude(field) {
		t.Error("条件满足时应返回 false (exclude)")
	}
}

// ============== ToLogString 测试 ==============

func TestToLogString(t *testing.T) {
	p := &KLVParser{
		fields: &BinaryLogFields{
			Fields: []BField{},
		},
	}
	p.fields.AddField(BField{Name: "F1", Value: "value1", Type: "string"})
	p.fields.AddField(BField{Name: "F2", Value: "value2", Type: "string"})
	p.fields.AddField(BField{Name: "F3", Value: "value3", Type: "string"})

	result := p.ToLogString()
	expected := "value1|value2|value3"
	if result != expected {
		t.Errorf("ToLogString() = %s, want %s", result, expected)
	}
}

func TestToLogString_Empty(t *testing.T) {
	p := &KLVParser{
		fields: &BinaryLogFields{
			Fields: []BField{},
		},
	}
	result := p.ToLogString()
	if result != "" {
		t.Errorf("空字段时 ToLogString() 应为空, 实际为 %s", result)
	}
}

func TestToLogString_SingleField(t *testing.T) {
	p := &KLVParser{
		fields: &BinaryLogFields{
			Fields: []BField{},
		},
	}
	p.fields.AddField(BField{Name: "F1", Value: "only", Type: "string"})

	result := p.ToLogString()
	if result != "only" {
		t.Errorf("单字段时 ToLogString() = %s, want 'only'", result)
	}
}

// ============== validateFieldRules 测试 ==============

func TestValidateFieldRules_RequiredNotEmpty(t *testing.T) {
	field := FieldDef{
		Name: "TestField",
		FieldRule: &parser.FieldRule{
			FieldName: "TestField",
			Required:  "必填",
		},
	}
	var errors []ValidationError

	err := validateFieldRules(field, "value", &errors)
	if err != nil {
		t.Errorf("必填字段有值时不应报错: %v", err)
	}
	if len(errors) != 0 {
		t.Errorf("必填字段有值时不应有错误, 但有 %d 个错误", len(errors))
	}
}

func TestValidateFieldRules_RequiredEmpty(t *testing.T) {
	field := FieldDef{
		Name: "TestField",
		FieldRule: &parser.FieldRule{
			FieldName: "TestField",
			Required:  "必填",
		},
	}
	var errors []ValidationError

	err := validateFieldRules(field, "", &errors)
	if err == nil {
		t.Error("必填字段为空时应报错")
	}
	if len(errors) != 1 {
		t.Errorf("应有 1 个错误, 实际有 %d 个", len(errors))
	}
	if errors[0].ErrorType != "required" {
		t.Errorf("错误类型应为 required, 实际为 %s", errors[0].ErrorType)
	}
}

func TestValidateFieldRules_OptionalEmpty(t *testing.T) {
	field := FieldDef{
		Name: "TestField",
		FieldRule: &parser.FieldRule{
			FieldName: "TestField",
			Required:  "选填",
			Type:      "int",
		},
	}
	var errors []ValidationError

	err := validateFieldRules(field, "", &errors)
	if err != nil {
		t.Errorf("选填字段为空时不应报错: %v", err)
	}
	if len(errors) != 0 {
		t.Errorf("选填字段为空时不应有错误, 但有 %d 个错误", len(errors))
	}
}

func TestValidateFieldRules_TypeValidation(t *testing.T) {
	field := FieldDef{
		Name: "TestField",
		FieldRule: &parser.FieldRule{
			FieldName: "TestField",
			Required:  "必填",
			Type:      "int",
		},
	}
	var errors []ValidationError

	err := validateFieldRules(field, "not_int", &errors)
	if err == nil {
		t.Error("类型校验失败时应报错")
	}
	if len(errors) != 1 {
		t.Errorf("应有 1 个错误, 实际有 %d 个", len(errors))
	}
	if errors[0].ErrorType != "type" {
		t.Errorf("错误类型应为 type, 实际为 %s", errors[0].ErrorType)
	}
}

func TestValidateFieldRules_NoRule(t *testing.T) {
	field := FieldDef{
		Name:      "TestField",
		FieldRule: nil,
	}
	var errors []ValidationError

	err := validateFieldRules(field, "any_value", &errors)
	if err != nil {
		t.Errorf("无规则时不应报错: %v", err)
	}
}

// ============== 集成测试 ==============

func TestProcessDatFile_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.dat")

	// 构建有效的二进制数据
	data := buildValidDatFile()
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		t.Fatalf("写入测试文件失败: %v", err)
	}

	sheetConfig := parser.SheetConfig{
		SheetName: "test",
		FieldRules: []parser.FieldRule{
			{FieldName: "CommandID", Required: "必填", Type: "string"},
		},
	}

	var errors []ValidationError
	result, err := ProcessDatFile(filePath, sheetConfig, &errors)
	if err != nil {
		t.Fatalf("ProcessDatFile 失败: %v", err)
	}
	if result.PrefixServer.Servers.ServerNum != 1 {
		t.Errorf("ServerNum 应为 1, 实际为 %d", result.PrefixServer.Servers.ServerNum)
	}
}

func TestProcessDatFile_NonExistentFile(t *testing.T) {
	var errors []ValidationError
	_, err := ProcessDatFile("/nonexistent/file.dat", parser.SheetConfig{}, &errors)
	if err == nil {
		t.Error("文件不存在时应报错")
	}
}

func TestProcessDatFile_InvalidData(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "invalid.dat")

	// 写入无效数据
	if err := os.WriteFile(filePath, []byte("invalid"), 0644); err != nil {
		t.Fatalf("写入测试文件失败: %v", err)
	}

	var errors []ValidationError
	_, err := ProcessDatFile(filePath, parser.SheetConfig{}, &errors)
	if err == nil {
		t.Error("无效数据时应报错")
	}
}

func TestBinaryFileRecord_GetFilePrefix(t *testing.T) {
	br := BinaryFileRecord{
		PrefixServer: FilePrefixServerInfo{
			Servers: Servers{
				ServerNum: 1,
				Server: []Server{
					{
						ServerIPType: 0,
						ServerIP:     []byte{192, 168, 1, 1},
						ServerPort:   8080,
					},
				},
			},
			PrefixLen: 8,
		},
	}

	result := br.GetFilePrefix()
	if result == "" {
		t.Error("GetFilePrefix() 不应返回空字符串")
	}
}

func TestBinaryFileRecord_GetAllFields(t *testing.T) {
	br := BinaryFileRecord{
		PrefixServer: FilePrefixServerInfo{
			Servers: Servers{
				ServerNum: 1,
				Server: []Server{
					{
						ServerIPType: 0,
						ServerIP:     []byte{10, 0, 0, 1},
						ServerPort:   443,
					},
				},
			},
			PrefixLen: 8,
		},
		parser: &KLVParser{
			fields: &BinaryLogFields{
				Fields: []BField{},
			},
		},
	}

	result := br.GetAllFields()
	if result == "" {
		t.Error("GetAllFields() 不应返回空字符串")
	}
}

// ============== 辅助函数 ==============

// buildValidDatFile 构建有效的测试二进制数据
func buildValidDatFile() []byte {
	var buf bytes.Buffer

	// 服务器信息前缀
	buf.WriteByte(1)                  // 1 个服务器
	buf.WriteByte(0)                  // IPv4
	buf.Write([]byte{192, 168, 1, 1}) // IP 地址
	buf.Write([]byte{0x1F, 0x90})     // 端口 8080

	// 日志前缀头 (16 字节)
	buf.Write(make([]byte, 16))

	// CommandID (13 字节)
	buf.Write([]byte("CMD0012345678"))

	// House_ID_Length (1 字节)
	buf.WriteByte(4)
	// House_ID (4 字节)
	buf.Write([]byte("H001"))

	// SourceIP_Length (1 字节)
	buf.WriteByte(4)
	// SrcIp (4 字节)
	buf.Write([]byte{10, 0, 0, 1})

	// DestinationIP_Length (1 字节)
	buf.WriteByte(4)
	// DestIp (4 字节)
	buf.Write([]byte{10, 0, 0, 2})

	// SrcPort (2 字节)
	binary.Write(&buf, binary.BigEndian, uint16(12345))
	// DestPort (2 字节)
	binary.Write(&buf, binary.BigEndian, uint16(80))

	// DomainName_Length (2 字节)
	binary.Write(&buf, binary.BigEndian, uint16(9))
	// DomainName (9 字节)
	buf.Write([]byte("test.com\x00"))

	// ProxyType_Flag (2 字节) - 值为 0 表示无代理
	binary.Write(&buf, binary.BigEndian, uint16(0))

	// Title_Length (2 字节) - 值为 0 表示无标题
	binary.Write(&buf, binary.BigEndian, uint16(0))

	// Content_Length (4 字节) - 值为 0 表示无内容
	binary.Write(&buf, binary.BigEndian, uint32(0))

	// Url_Length (2 字节) - 值为 0 表示无 URL
	binary.Write(&buf, binary.BigEndian, uint16(0))

	// Attachmentfile_Num (1 字节) - 0 个附件
	buf.WriteByte(0)

	// GatherTime (4 字节)
	binary.Write(&buf, binary.BigEndian, uint32(1609459200))
	// TrafficType (1 字节)
	buf.WriteByte(1)
	// ProtocolType (1 字节)
	buf.WriteByte(6) // TCP
	// ApplicationProtocol (2 字节)
	binary.Write(&buf, binary.BigEndian, uint16(80))
	// BusinessProtocol (2 字节)
	binary.Write(&buf, binary.BigEndian, uint16(1))

	return buf.Bytes()
}
