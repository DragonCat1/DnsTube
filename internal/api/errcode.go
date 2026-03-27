package api

// 错误码：稳定标识，供前后端与文档引用；中文文案见 msgZH，勿在 handler 中硬编码用户可见文案。

const (
	CodeInvalidJSON       = "INVALID_JSON"
	CodeBadID             = "BAD_ID"
	CodeNotFound          = "NOT_FOUND"
	CodeDuplicateKey      = "DUPLICATE_KEY"
	CodeDatabaseError     = "DATABASE_ERROR"
	CodeBadRequest        = "BAD_REQUEST"
	CodeMissingBearer     = "MISSING_BEARER_TOKEN"
	CodeInvalidToken      = "INVALID_TOKEN"
	CodeNoUserContext     = "NO_USER_CONTEXT"
	CodeInvalidCredentials = "INVALID_CREDENTIALS"
	CodeTokenSignFailed   = "TOKEN_SIGN_FAILED"
	CodePasswordHashFailed = "PASSWORD_HASH_FAILED"
	CodeInstanceFieldsRequired = "INSTANCE_NAME_OR_PORT_REQUIRED"
	CodeRuleFieldsRequired     = "FORWARD_RULE_FIELDS_REQUIRED"
	CodePatternURLFetchFailed  = "PATTERN_URL_FETCH_FAILED"
	CodeRuleModeInvalid        = "FORWARD_RULE_MODE_INVALID"
	CodeGroupNameRequired      = "UPSTREAM_GROUP_NAME_REQUIRED"
	CodeServerAddressRequired   = "UPSTREAM_SERVER_ADDRESS_REQUIRED"
	CodeUpstreamServerIPInvalid = "UPSTREAM_SERVER_IP_INVALID"
	CodeDNSRecordIPv4Invalid    = "DNS_RECORD_IPV4_INVALID"
	CodeDNSRecordIPv6Invalid    = "DNS_RECORD_IPV6_INVALID"
	CodeDNSReloadFailed         = "DNS_ENGINE_RELOAD_FAILED"
	CodeDNSBindFailed          = "DNS_UDP_BIND_FAILED"
	CodeTelegramSendFailed     = "TELEGRAM_SEND_FAILED"
)

var msgZH = map[string]string{
	CodeInvalidJSON:            "请求体 JSON 无效",
	CodeBadID:                  "路径中的 ID 无效",
	CodeNotFound:               "资源不存在",
	CodeDuplicateKey:           "与已有数据冲突（唯一约束）",
	CodeDatabaseError:          "数据库错误",
	CodeBadRequest:             "请求参数无效",
	CodeMissingBearer:          "缺少 Bearer Token",
	CodeInvalidToken:           "Token 无效或已过期",
	CodeNoUserContext:          "未找到当前用户",
	CodeInvalidCredentials:     "用户名或密码错误",
	CodeTokenSignFailed:        "签发 Token 失败",
	CodePasswordHashFailed:     "密码处理失败",
	CodeInstanceFieldsRequired: "实例名称与监听端口为必填",
	CodeRuleFieldsRequired:     "转发规则需填写域名正则与模式",
	CodePatternURLFetchFailed:  "拉取规则文件失败",
	CodeRuleModeInvalid:        "转发模式须为 sequential 或 parallel",
	CodeGroupNameRequired:      "转发组名称为必填",
	CodeServerAddressRequired:  "上游服务器地址为必填",
	CodeUpstreamServerIPInvalid: "上游服务器地址须为合法 IPv4 或 IPv6",
	CodeDNSRecordIPv4Invalid:   "A 记录正文须为合法 IPv4 地址",
	CodeDNSRecordIPv6Invalid:   "AAAA 记录正文须为合法 IPv6 地址",
	CodeDNSReloadFailed:        "DNS 引擎重载配置失败",
	CodeDNSBindFailed:          "UDP 监听地址绑定失败（端口可能被占用）",
	CodeTelegramSendFailed:     "Telegram 发送失败（请检查 Token、Chat ID、是否启用通知与网络）",
}

func msgForCode(code string) string {
	if m, ok := msgZH[code]; ok {
		return m
	}
	return code
}

// APICodeNum 将字符串错误码映射为 JSON 顶层非零整数 code；成功为 0。
func APICodeNum(code string) int {
	if n, ok := apiCodeNumByString[code]; ok {
		return n
	}
	return 1999
}

// apiCodeNumByString 与 errcode 常量一一对应，稳定不变。
var apiCodeNumByString = map[string]int{
	CodeInvalidJSON:        1001,
	CodeBadID:              1002,
	CodeNotFound:           1003,
	CodeDuplicateKey:       1004,
	CodeDatabaseError:      1005,
	CodeBadRequest:         1006,
	CodeMissingBearer:      1007,
	CodeInvalidToken:       1008,
	CodeNoUserContext:      1009,
	CodeInvalidCredentials: 1010,
	CodeTokenSignFailed:    1011,
	CodePasswordHashFailed: 1012,
	CodeInstanceFieldsRequired: 1013,
	CodeRuleFieldsRequired:     1014,
	CodePatternURLFetchFailed:  1024,
	CodeRuleModeInvalid:        1015,
	CodeGroupNameRequired:      1016,
	CodeServerAddressRequired: 1017,
	CodeUpstreamServerIPInvalid: 1023,
	CodeDNSReloadFailed:       1018,
	CodeDNSBindFailed:         1019,
	CodeTelegramSendFailed:    1020,
	CodeDNSRecordIPv4Invalid:  1021,
	CodeDNSRecordIPv6Invalid:  1022,
}
