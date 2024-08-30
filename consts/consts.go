package consts

const (
	AtypIPv4   = 0x01
	AtypDomain = 0x03
	AtypIpv6   = 0x04
)

const (
	RepSuccess = 0x00 // 代理服务器到目的服务器的连接建立成功
	RepFailed  = 0x01 // 代理服务器到目的服务器的连接建立失败,这里粗略的用 1 代表所有错误情况，实际细分了很多种
	CmdConnect = 0x01
	CmdBind    = 0x02 // not support
	CmdUdp     = 0x03 // not support
	RSV        = 0x00 // 保留字段
)

const (
	Version      = 0x05 // socket5 ver 的默认值
	AuthUserOk   = 0x00 // 用户验证成功
	AuthUserFail = 0x01 // 用户验证失败（非 0）
)

// 服务端支持的认证方式
const (
	AuthTypeNoRequired   byte = 0x00 // 无需认证
	AuthTypeUnamePwd          = 0x02 // 使用用户名/密码进行认证
	AuthTypeNoAcceptable      = 0xff // 客户端不支持服务端的认证方法
)
