package socks5

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
)

const (
	RepSuccess = 0x00 // 代理服务器到目的服务器的连接建立成功
	RepFailed  = 0x01 // 代理服务器到目的服务器的连接建立失败,这里粗略的用 1 代表所有错误情况，实际细分了很多种
	AtypIPv4   = 0x01
	AtypIpv6   = 0x04
	AtypDomain = 0x03
	CmdConnect = 0x00
	CmdBind    = 0x02 // not support
	CmdUdp     = 0x03 // not support
	RSV        = 0x00 // 保留字段
)

// CreateTargetConn 从 cliConn 中读取数据，并据此建立到目的地址服务器的连接，之后应该将结果返回给 cliConn
func CreateTargetConn(cliConn net.Conn) (targetConn net.Conn, err error) {
	cmd, addr, port, err := getRequest(cliConn)
	if err != nil {
		return
	}

	// 获取目的服务器的连接
	targetConn, err = dialConn(cliConn, cmd, addr, port)
	return
}

// SOCKS 的请求构成如下：（参见 RFC 1928，4. Requests）
// +----+-----+-------+------+----------+----------+
// |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
// +----+-----+-------+------+----------+----------+
// | 1  |  1  | X'00' |  1   | Variable |    2     |
// +----+-----+-------+------+----------+----------+
// VER		0x05，协议版本号
// CMD		连接方式，0x01=CONNECT, 0x02=BIND, 0x03=UDP ASSOCIATE
// RSV		保留字段，目前没用
// ATYP		地址类型，0x01=IPv4，0x03=域名，0x04=IPv6
// DST.ADDR	目标地址
// DST.PORT	目标端口，2字节，网络字节序（network octec order）
func getRequest(conn net.Conn) (cmd byte, addr, port string, err error) {
	b := make([]byte, 255)

	_, err = io.ReadFull(conn, b[:4])
	if err != nil {
		err = fmt.Errorf("read header[VER, CMD, RSV, ATYP] error: %+v", err)
		return
	}

	ver, cmd, rsv, atyp := b[0], b[1], b[2], b[3]
	if ver != 5 {
		err = fmt.Errorf("invalid version")
		return
	}

	_ = rsv // 忽略保留字段

	addr, err = ParseAddr(atyp, conn)
	if err != nil {
		return
	}

	port, err = ParsePort(conn)
	if err != nil {
		return
	}

	return
}

// parseAddr 根据 ATYP 获得客户要连接的地址
func ParseAddr(atyp byte, conn net.Conn) (addr string, err error) {
	b := make([]byte, 255)
	switch atyp {

	case AtypIPv4: // IPv4
		_, err = io.ReadFull(conn, b[:4])
		if err != nil {
			return "",
				fmt.Errorf("parse atyp 0x01 [IPv4] addr error: %+v", err)
		}
		addr = fmt.Sprintf("%v.%v.%v.%v", b[0], b[1], b[2], b[3])
		return
	case AtypDomain: // 域名
		// DST.ADDR 部分第一个字节为域名长度，DST.ADDR剩余的内容为域名，
		// 没有 0 结尾。
		_, err = io.ReadFull(conn, b[:1])
		if err != nil {
			return "",
				fmt.Errorf("read atyp 0x03 [domain] len error: %+v", err)
		}
		domainLen := b[0]

		_, err = io.ReadFull(conn, b[:domainLen])
		if err != nil {
			return "",
				fmt.Errorf("parse atyp 0x03 [domain] addr error: %+v", err)
		}
		addr = string(b[:domainLen])
		return
	case AtypIpv6: // IPv6
		// TODO:
		return "", fmt.Errorf("IPv6 not support yet")

	default:
		return "", fmt.Errorf("invalid atyp")
	}
}

// parsePort 解析出 DST.PORT
func ParsePort(conn net.Conn) (port string, err error) {
	b := make([]byte, 2)

	_, err = io.ReadFull(conn, b)
	if err != nil {
		return "", fmt.Errorf("parse port error: %+v", err)
	}

	// 通过大端获取
	p := binary.BigEndian.Uint16(b)
	// port = string(p)
	// conversion from int to string yields a string of one rune,
	// not a string of digits (did you mean fmt.Sprint(x)?)
	port = fmt.Sprint(p)

	return
}

// dialConn 根据客户提供的 cmd 参数，向目的服务器建立对应的连接
// 0x01=CONNECT, 0x02=BIND, 0x03=UDP ASSOCIATE
// 之后会将响应返回给客户（写入到 cliConn），返回服务端和
// 目的地址的连接（targetConn）
func dialConn(
	cliConn net.Conn, cmd byte,
	addr, port string) (targetConn net.Conn, err error) {
	address := fmt.Sprintf("%s:%s", addr, port)
	switch cmd {
	case CmdConnect:
		// 建立到目的地址的 tcp 连接
		targetConn, err = net.Dial("tcp", address)
		if err != nil {
			return
		}
		log.Printf("[CmdConnect] dial tcp conn [%v] \n", address)
		return

	default: // 只支持 CONNECT
		err = fmt.Errorf("only CONNECT be able to accept")
		return
	}
}

// +-----+-----+-------+------+----------+----------+
// | VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
// +-----+-----+-------+------+----------+----------+
// |  1  |  1  | X'00' |  1   | Variable |    2     |
// +-----+-----+-------+------+----------+----------+
// ATYP	地址类型，0x01=IPv4，0x03=域名，0x04=IPv6
func writeResponse(conn net.Conn,
	rep, atyp byte, bndPort, bndAddr []byte) error {

	var b bytes.Buffer

	b.WriteByte(Version)
	b.WriteByte(rep)
	b.WriteByte(0x00) // RSV
	b.WriteByte(atyp)
	b.Write(bndAddr)
	b.Write(bndPort)

	_, err := conn.Write(b.Bytes())
	if err != nil {
		return fmt.Errorf("write response error: %+v", err)
	}

	return nil
}

// WriteIPv4SuccessResponse 向 cliConn 中写入 “连接建立成功” 的响应
func WriteIPv4SuccessResponse(cliConn, targetConn net.Conn) (err error) {
	addr, port, err := getTargetConnMessage(targetConn)
	if err != nil {
		return
	}

	ip4 := net.ParseIP(addr).To4()

	b := make([]byte, 2)
	uport, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		return err
	}
	binary.BigEndian.PutUint16(b, uint16(uport))

	// 写入 resp
	err = writeResponse(cliConn, RepSuccess, AtypIPv4, b, ip4)
	return
}

// WriteIPv4SuccessResponse 向 cliConn 中写入 “连接建立失败” 的响应
func WriteIPv4FailedResponse(cliConn net.Conn) (err error) {
	// 写入 resp
	err = writeResponse(cliConn, RepFailed, AtypIPv4, []byte{0, 0}, []byte{0})
	return
}

// getTargetConnMessage 从 targetConn 中获取 addr 和 port 信息，用于填充响应报文的 BND.ADDR 和
// BND.PORT 字段
func getTargetConnMessage(targetConn net.Conn) (addr, port string, err error) {
	if targetConn == nil {
		err = fmt.Errorf("targetConn is nil")
		return
	}
	taddr := targetConn.LocalAddr().String()
	s := strings.Split(taddr, ":")
	if len(s) < 2 {
		err = fmt.Errorf("get BND.ADDR error, need [host:port]")
		return
	}

	addr = s[0]
	port = s[1]

	return
}
