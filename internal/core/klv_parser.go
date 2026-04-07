package core

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strconv"
	"xdrCheck/internal/parser"
	"xdrCheck/internal/validator"
)

// 字段类型标识（可根据需要扩展）
const (
	TypeFixed              = iota // 固定长度
	TypeLengthPrefixed            // 前导长度字段
	TypeConditional               // 有条件出现
	TypeConditionalExclude        // 有条件出现，排除该条件
	TypeRepeated                  // 可能重复出现
)

// 字段描述
type FieldDef struct {
	Name           string
	Length         int               // 固定长度时使用；变长时为0
	LenField       string            // 前导长度字段名（变长时使用）
	Type           int               // TypeFixed 或 TypeLengthPrefixed
	ValueType      string            // 字段值类型  比如"int64" "string" "byte" 等
	ConditionField string            // 有条件出现时的条件字段名（TypeConditional时使用）
	ConditionValue int               // 有条件出现时的条件值（TypeConditional时使用）
	RepeatedField  string            // 可能重复出现时的重复字段名（TypeRepeated时使用）
	FieldRule      *parser.FieldRule // 校验规则结构
}

var formatDef01e0 = []FieldDef{
	{"CommandID", 13, "", TypeFixed, "cmd", "", 0, "", nil},
	{"House_ID_Length", 1, "", TypeFixed, "uint8", "", 0, "", nil},
	{"House_ID", 0, "House_ID_Length", TypeLengthPrefixed, "string", "", 0, "", nil},
	{"SourceIP_Length", 1, "", TypeFixed, "uint8", "", 0, "", nil},
	{"SrcIp", 0, "SourceIP_Length", TypeLengthPrefixed, "ip", "", 0, "", nil},
	{"DestinationIP_Length", 1, "", TypeFixed, "uint8", "", 0, "", nil},
	{"DestIp", 0, "DestinationIP_Length", TypeLengthPrefixed, "ip", "", 0, "", nil},
	{"SrcPort", 2, "", TypeFixed, "uint16", "", 0, "", nil},
	{"DestPort", 2, "", TypeFixed, "uint16", "", 0, "", nil},
	{"DomainName_Length", 2, "", TypeFixed, "uint16", "", 0, "", nil},
	{"DomainName", 0, "DomainName_Length", TypeLengthPrefixed, "string", "", 0, "", nil},

	// 代理相关（有条件出现）
	{"ProxyType_Flag", 2, "", TypeFixed, "uint8", "", 0, "", nil},
	{"ProxyType", 2, "", TypeConditional, "uint16", "ProxyType_Flag", 1, "", nil},
	{"ProxyIp_Length", 1, "", TypeConditional, "uint8", "ProxyType_Flag", 1, "", nil},
	{"ProxyIp", 0, "ProxyIp_Length", TypeConditional, "ip", "ProxyType_Flag", 1, "", nil},
	{"ProxyPort", 2, "", TypeConditional, "uint16", "ProxyType_Flag", 1, "", nil},

	{"Title_Length", 2, "", TypeFixed, "uint16", "", 0, "", nil},
	{"Title", 0, "Title_Length", TypeConditionalExclude, "string", "Title_Length", 0, "", nil},
	{"Content_Length", 4, "", TypeFixed, "uint32", "", 0, "", nil},
	{"Content", 0, "Content_Length", TypeConditionalExclude, "string", "Content_Length", 0, "", nil},
	{"Url_Length", 2, "", TypeFixed, "uint16", "", 0, "", nil},
	{"Url", 0, "Url_Length", TypeConditionalExclude, "base64", "Url_Length", 0, "", nil},

	// 附件相关（可能重复出现）
	{"Attachmentfile_Num", 1, "", TypeFixed, "uint8", "", 0, "", nil},
	{"AttachmentfileName_Length", 2, "", TypeRepeated, "uint16", "", 0, "Attachmentfile_Num", nil},
	{"AttachmentfileName", 0, "AttachmentfileName_Length", TypeRepeated, "string", "", 0, "Attachmentfile_Num", nil},

	// 采集时间和协议信息
	{"GatherTime", 4, "", TypeFixed, "uint32", "", 0, "", nil},
	{"TrafficType", 1, "", TypeFixed, "uint8", "", 0, "", nil},
	{"ProtocolType", 1, "", TypeFixed, "uint8", "", 0, "", nil},
	{"ApplicationProtocol", 2, "", TypeFixed, "uint16", "", 0, "", nil},
	{"BusinessProtocol", 2, "", TypeFixed, "uint16", "", 0, "", nil},
}

// BField 二进制字段结构体
type BField struct {
	Name  string      `json:"name"`  // 字段名称
	Len   uint32      `json:"len"`   // 字段长度
	Data  []byte      `json:"data"`  // 字段原始数据
	Value interface{} `json:"value"` // 字段值
	Type  string      `json:"type"`  // uint8/uint16/uint32/uint64/string/[]byte
}

// LogPrefixHeader 日志前缀头结构体
type LogPrefixHeader struct {
	PrefixLog [16]byte
}

// BinaryLogFields 二进制日志字段集合
type BinaryLogFields struct {
	PrefixHeader LogPrefixHeader
	FieldsNum    uint16
	Fields       []BField // 字段列表
}

// SetHeader 设置前缀头
func (bf *BinaryLogFields) SetHeader(header LogPrefixHeader) {
	bf.PrefixHeader = header
}

// GetHeader 获取前缀头
func (bf *BinaryLogFields) GetHeader() LogPrefixHeader {
	return bf.PrefixHeader
}

// ResetHeader 重置前缀头
func (bf *BinaryLogFields) ResetHeader() {
	bf.PrefixHeader = LogPrefixHeader{}
}

// GetFields 获取字段列表
func (bf *BinaryLogFields) GetFields() []BField {
	return bf.Fields
}

// AddField 添加字段
func (bf *BinaryLogFields) AddField(field BField) {
	bf.FieldsNum++
	bf.Fields = append(bf.Fields, field)
}

// Reset 重置字段集合
func (bf *BinaryLogFields) Reset() {
	bf.PrefixHeader = LogPrefixHeader{}
	bf.FieldsNum = 0
	bf.Fields = nil
}

// ByteReader 字节读取器
type ByteReader struct {
	data   []byte
	offset int
}

// NewByteReader 创建字节读取器
func NewByteReader(data []byte) *ByteReader {
	return &ByteReader{
		data:   data,
		offset: 0,
	}
}

// ReadBytes 读取指定长度的字节
func (r *ByteReader) ReadBytes(length int) ([]byte, error) {
	if r.offset+length > len(r.data) {
		return nil, io.EOF
	}

	result := r.data[r.offset : r.offset+length]
	r.offset += length
	return result, nil
}

// Remaining 获取剩余字节数
func (r *ByteReader) Remaining() int {
	return len(r.data) - r.offset
}

// ==================== 文件头服务器信息解析 ====================

// Server 服务器信息结构体
type Server struct {
	ServerIPType uint8  // 服务器IP类型，0-ipv4 4字节, 1-ipv6 16字节
	ServerIP     []byte // 服务器IP地址
	ServerPort   uint16 // 服务器端口号
}

// Servers 服务器集合
type Servers struct {
	ServerNum uint8 // 后续的字段重复次数
	Server    []Server
}

// FilePrefixServerInfo 文件前缀服务器信息
type FilePrefixServerInfo struct {
	Servers   Servers
	PrefixLen uint32
	Prefix    []byte
}

// BinaryFileRecord 二进制文件记录结构
type BinaryFileRecord struct {
	PrefixServer FilePrefixServerInfo
	LogFields    []BinaryLogFields
	Lines        []string
	parser       *KLVParser
}

// ExtractMode 提取模式
type ExtractMode int

const (
	ExtractFirst ExtractMode = iota
	ExtractAll
)

var (
	ErrServerInfoInvalid = fmt.Errorf("server info is invalid")
)

// parseServer 解析单个服务器信息
func parseServer(data []byte, offset *uint32) (Server, error) {
	var ser Server

	ser.ServerIPType = uint8(data[*offset])
	*offset += 1
	if ser.ServerIPType == 0 {
		ser.ServerIP = data[*offset : *offset+4]
		*offset += 4
	} else {
		ser.ServerIP = data[*offset : *offset+16]
		*offset += 16
	}
	ser.ServerPort = binary.BigEndian.Uint16(data[*offset : *offset+2])
	*offset += 2
	return ser, nil
}

// CheckAndParseServerInfo 检查并解析服务器信息
func CheckAndParseServerInfo(data []byte) (FilePrefixServerInfo, error) {
	if len(data) <= 8 {
		return FilePrefixServerInfo{}, ErrServerInfoInvalid
	}

	var offset uint32
	var ps FilePrefixServerInfo

	ps.Servers.ServerNum = uint8(data[0])
	offset += 1
	for i := 0; i < int(ps.Servers.ServerNum); i++ {
		ser, err := parseServer(data, &offset)
		if err != nil {
			return FilePrefixServerInfo{}, fmt.Errorf("failed to parse server: %w", err)
		}
		ps.Servers.Server = append(ps.Servers.Server, ser)
	}

	ps.Prefix = data[:offset]
	ps.PrefixLen = offset

	return ps, nil
}

// ProcessDatFile 处理二进制数据文件
func ProcessDatFile(filePath string, sheetConfig parser.SheetConfig, errors *[]ValidationError) (BinaryFileRecord, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return BinaryFileRecord{}, fmt.Errorf("failed to read binary file: %w", err)
	}

	ps, err := CheckAndParseServerInfo(data)
	if err != nil {
		return BinaryFileRecord{}, fmt.Errorf("failed to parse server info: %w", err)
	}

	var br BinaryFileRecord
	// 解析FilePrefixServerInfo
	br.PrefixServer = ps
	data = data[ps.PrefixLen:]

	if len(data) == 0 {
		return BinaryFileRecord{}, nil
	}

	var lines []string

	// 创建带规则的KLV解析器
	br.parser = NewKLVParserWithRules(data, sheetConfig)
	for !br.parser.ParseDone() {
		br.parser.ResetForNewLog()
		err = br.parser.Parse(errors)
		if err != nil {
			return BinaryFileRecord{}, fmt.Errorf("failed to parse fields: %w", err)
		}
		logLine := br.GetLogLine()
		lines = append(lines, logLine)
	}

	br.Lines = lines

	return br, nil
}

// GetFilePrefixData 获取文件前缀数据
func (br BinaryFileRecord) GetFilePrefixData() []byte {
	return br.PrefixServer.Prefix
}

// GetFilePrefix 获取文件前缀字符串
func (br BinaryFileRecord) GetFilePrefix() string {
	var b bytes.Buffer

	b.WriteString(fmt.Sprintf("Byte Length: %d\n", br.PrefixServer.PrefixLen))

	b.WriteString("Server: ")
	for _, ser := range br.PrefixServer.Servers.Server {
		b.WriteString(fmt.Sprintf("%s:%d\n", ser.ServerIP, ser.ServerPort))
	}
	b.WriteString("\n")

	return b.String()
}

// GetLogLine 获取日志行
func (br BinaryFileRecord) GetLogLine() string {
	return br.parser.ToLogString()
}

// GetAllFields 获取所有字段信息
func (br BinaryFileRecord) GetAllFields() string {
	var b bytes.Buffer

	b.WriteString(fmt.Sprintf("Byte Length: %d\n", br.PrefixServer.PrefixLen))

	b.WriteString("Server: ")
	for _, ser := range br.PrefixServer.Servers.Server {
		b.WriteString(fmt.Sprintf("%s:%d\n", ser.ServerIP, ser.ServerPort))
	}
	b.WriteString("\n")

	b.WriteString("log: ")
	b.WriteString(br.GetLogLine())
	b.WriteString("\n")

	return b.String()
}

// ==================== KLV解析器 ====================

// KLVParser KLV格式解析器
type KLVParser struct {
	formatDef  []FieldDef
	fieldMap   map[string]string
	fieldSlice []*BinaryLogFields
	fields     *BinaryLogFields
	reader     *ByteReader
}

// NewKLVParser 创建KLV解析器
func NewKLVParser(data []byte) *KLVParser {
	return &KLVParser{
		formatDef:  formatDef01e0,
		fieldMap:   make(map[string]string),
		fieldSlice: []*BinaryLogFields{},
		fields:     &BinaryLogFields{},
		reader:     NewByteReader(data),
	}
}

// NewKLVParserWithRules 创建带规则的KLV解析器
func NewKLVParserWithRules(data []byte, sheetConfig parser.SheetConfig) *KLVParser {
	// 复制基础格式定义
	customFormatDef := make([]FieldDef, len(formatDef01e0))
	copy(customFormatDef, formatDef01e0)

	// 将sheetConfig中的规则赋值到formatDef上，通过字段名关联
	for i := range customFormatDef {
		fieldName := customFormatDef[i].Name
		// 在sheetConfig中查找对应的字段规则
		for _, fieldRule := range sheetConfig.FieldRules {
			if fieldRule.FieldName == fieldName {
				// 复制规则到字段定义
				customFormatDef[i].FieldRule = &fieldRule
				break
			}
		}
	}

	return &KLVParser{
		formatDef:  customFormatDef,
		fieldMap:   make(map[string]string),
		fieldSlice: []*BinaryLogFields{},
		fields:     &BinaryLogFields{},
		reader:     NewByteReader(data),
	}
}

// ResetForNewLog 重置解析器以解析新的日志
func (p *KLVParser) ResetForNewLog() {
	p.fieldMap = make(map[string]string)
}

// Parse 解析KLV二进制数据
func (p *KLVParser) Parse(errors *[]ValidationError) error {
	reader := p.reader

	var header LogPrefixHeader
	h, err := reader.ReadBytes(16)
	if err != nil {
		return fmt.Errorf("failed to read prefix header: %w", err)
	}
	copy(header.PrefixLog[:], h)
	p.fields.SetHeader(header)

	// 使用klvparser进行解析
	for i := 0; i < len(p.formatDef); i++ {
		field := p.formatDef[i]

		// 处理有条件出现的字段
		if field.Type == TypeConditional {
			if !p.shouldIncludeField(field) {
				continue
			}
		}

		// 处理有条件出现的字段
		if field.Type == TypeConditionalExclude {
			if !p.shouldIncludeFieldExclude(field) {
				continue
			}
		}

		// 处理重复字段
		if field.Type == TypeRepeated {
			if err := p.parseRepeatedField(reader, field, errors); err != nil {
				return err
			}
			continue
		}

		// 解析单个字段
		if err := p.parseField(reader, field, errors); err != nil {
			return err
		}
	}

	return nil
}

// parseField 解析单个字段
func (p *KLVParser) parseField(reader *ByteReader, field FieldDef, errors *[]ValidationError) error {
	if field.Length == 0 {
		return p.parseLengthPrefixedField(reader, field, errors)
	} else {
		return p.parseFixedField(reader, field, errors)
	}
}

// parseFixedField 解析固定长度字段
func (p *KLVParser) parseFixedField(reader *ByteReader, field FieldDef, errors *[]ValidationError) error {
	if field.Length <= 0 {
		return fmt.Errorf("invalid fixed length for field %s: %d", field.Name, field.Length)
	}

	data, err := reader.ReadBytes(field.Length)
	if err != nil {
		return fmt.Errorf("failed to read fixed field %s: %w", field.Name, err)
	}

	// 根据字段名进行适当的类型转换
	value := p.convertFieldValue(field.ValueType, data)

	// 根据条件结果确定字段的实际必填状态
	err = validateFieldRules(field, value, errors)
	if err != nil {
		// 错误已经通过指针追加到errors切片中
	}
	p.fieldMap[field.Name] = value

	p.fields.AddField(BField{
		Name:  field.Name,
		Len:   uint32(field.Length),
		Data:  data,
		Type:  field.ValueType,
		Value: value,
	})

	return nil
}

// parseLengthPrefixedField 解析前导长度字段
func (p *KLVParser) parseLengthPrefixedField(reader *ByteReader, field FieldDef, errors *[]ValidationError) error {
	// 获取长度字段的值
	lengthValue, exists := p.fieldMap[field.LenField]
	if !exists {
		return fmt.Errorf("length field %s not found for field %s", field.LenField, field.Name)
	}

	length, err := strconv.Atoi(lengthValue)
	if err != nil {
		return fmt.Errorf("invalid length value for field %s: %w", field.Name, err)
	}

	data, err := reader.ReadBytes(length)
	if err != nil {
		return fmt.Errorf("failed to read length-prefixed field %s: %w", field.Name, err)
	}
	value := p.convertFieldValue(field.ValueType, data)

	// 根据条件结果确定字段的实际必填状态
	err = validateFieldRules(field, value, errors)
	if err != nil {
		// 错误已经通过指针追加到errors切片中
	}
	p.fieldMap[field.Name] = value

	p.fields.AddField(BField{
		Name:  field.Name,
		Len:   uint32(length),
		Data:  data,
		Type:  field.ValueType,
		Value: value,
	})

	return nil
}

// parseRepeatedField 解析重复字段
func (p *KLVParser) parseRepeatedField(reader *ByteReader, field FieldDef, errors *[]ValidationError) error {
	// 获取重复次数
	repeatCountValue, exists := p.fieldMap[field.RepeatedField]
	if !exists {
		return fmt.Errorf("repeat count field %s not found for field %s", field.RepeatedField, field.Name)
	}

	repeatCount := p.convertToInt(repeatCountValue)
	if repeatCount < 0 {
		return fmt.Errorf("invalid repeat count %d for field %s", repeatCount, field.Name)
	}

	for i := 0; i < repeatCount; i++ {
		if err := p.parseField(reader, field, errors); err != nil {
			return err
		}
	}

	return nil
}

// shouldIncludeField 判断是否应该包含条件字段
func (p *KLVParser) shouldIncludeField(field FieldDef) bool {
	conditionValue, exists := p.fieldMap[field.ConditionField]
	if !exists {
		return false
	}
	actualValue := p.convertToInt(conditionValue)
	return actualValue == field.ConditionValue
}

// shouldIncludeFieldExclude 判断是否应该包含排除条件字段
func (p *KLVParser) shouldIncludeFieldExclude(field FieldDef) bool {
	conditionValue, exists := p.fieldMap[field.ConditionField]
	if !exists {
		return false
	}
	actualValue := p.convertToInt(conditionValue)
	return actualValue != field.ConditionValue
}

// convertFieldValue 转换字段值
func (p *KLVParser) convertFieldValue(valueType string, data []byte) string {
	switch valueType {
	case "bool", "int", "uint8", "uint16", "uint32":
		return fmt.Sprintf("%d", p.convertToInt(data))
	case "string":
		return string(data)
	case "ip":
		return p.convertToIP(data)
	case "base64":
		return p.convertToBase64(data)
	case "cmd":
		return string(data)
	default:
		return string(data)
	}
}

// convertToInt 转换为整数
func (p *KLVParser) convertToInt(data interface{}) int {
	switch v := data.(type) {
	case []byte:
		if len(v) == 0 {
			return 0
		}
		var result int
		for i, b := range v {
			result += int(b) << (8 * (len(v) - i - 1))
		}
		return result
	case int:
		return v
	case uint8:
		return int(v)
	case uint16:
		return int(v)
	case uint32:
		return int(v)
	default:
		return 0
	}
}

// convertToIP 转换为IP地址字符串
func (p *KLVParser) convertToIP(data []byte) string {
	if len(data) == 4 {
		return fmt.Sprintf("%d.%d.%d.%d", data[0], data[1], data[2], data[3])
	}
	return string(data)
}

// convertToBase64 转换为Base64字符串
func (p *KLVParser) convertToBase64(data []byte) string {
	return string(data) // 简化处理，实际使用时可能需要真正的base64编码
}

// ToLogString 将解析结果转换为日志行格式
func (p *KLVParser) ToLogString() string {
	var result string
	fields := p.fields.GetFields()
	for i, field := range fields {
		result += fmt.Sprintf("%v", field.Value)
		if i < len(fields)-1 {
			result += "|"
		}
	}
	return result
}

// GetFields 获取解析后的字段列表
func (p *KLVParser) GetFields() []BField {
	return p.fields.GetFields()
}

// ParseDone 检查是否解析完成
func (p *KLVParser) ParseDone() bool {
	return p.reader.Remaining() == 0
}

// validateFieldRules 校验字段规则
func validateFieldRules(field FieldDef, data string, errors *[]ValidationError) error {
	// 校验字段
	if field.FieldRule == nil {
		return nil
	}

	actualRequired := field.FieldRule.Required

	// 然后校验类型
	if field.FieldRule.Type != "" {
		// 对于选填字段且为空的情况，跳过类型校验
		if actualRequired == "选填" && data == "" {
			// 选填字段为空时，跳过类型校验
		} else {
			typeValidator := validator.NewRuleValidator(data, 0, []string{field.Name}, nil)
			valid, msg := typeValidator.ValidateType(field.FieldRule.Type)
			if !valid {
				*errors = append(*errors, ValidationError{
					Filename:   "",
					LineNum:    1,
					FieldIndex: 0,
					FieldName:  field.FieldRule.FieldName,
					ErrorType:  "type",
					RuleOrType: field.FieldRule.Type,
					Message:    msg,
					FieldValue: data,
					FullLine:   "",
				})
				return fmt.Errorf("类型校验失败")
			}
		}
	}

	// 然后校验其他规则
	for _, rule := range field.FieldRule.Rules {
		// 对于选填字段且为空的情况，跳过规则校验
		if actualRequired == "选填" && data == "" {
			// 选填字段为空时，跳过规则校验
		} else {
			// 如果字段类型是base64，先进行base64解码
			ruleValue := data
			if field.FieldRule.Type == "base64" {
				decoded, err := decodeBase64(ruleValue)
				if err != nil {
					*errors = append(*errors, ValidationError{
						Filename:   "",
						LineNum:    1,
						FieldIndex: 0,
						FieldName:  field.FieldRule.FieldName,
						ErrorType:  "rule",
						RuleOrType: rule,
						Message:    fmt.Sprintf("base64解码失败: %v", err),
						FieldValue: ruleValue,
						FullLine:   "",
					})
					return err
				}
				ruleValue = decoded
			}

			// 使用解码后的值进行规则校验
			ruleValidator := validator.NewRuleValidator(ruleValue, 0, []string{field.Name}, nil)
			valid, msg := ruleValidator.ValidateRule(rule)
			if !valid {
				*errors = append(*errors, ValidationError{
					Filename:   "",
					LineNum:    1,
					FieldIndex: 0,
					FieldName:  field.FieldRule.FieldName,
					ErrorType:  "rule",
					RuleOrType: rule,
					Message:    msg,
					FieldValue: ruleValue,
					FullLine:   "",
				})
			}
			return fmt.Errorf("规则校验失败")
		}
	}

	return nil
}
