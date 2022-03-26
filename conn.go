package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/zengh1/socks5/util"
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
	CmdConnect = 0x01
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

	addr, err = util.ParseAddr(atyp, conn)
	if err != nil {
		return
	}

	port, err = util.ParsePort(conn)
	if err != nil {
		return
	}

	return
}

// dialConn 根据客户提供的 cmd 参数，向目的服务器建立对应的连接
// 0x01=CONNECT, 0x02=BIND, 0x03=UDP ASSOCIATE
// 之后会将响应返回给客户（写入到 cliConn），返回服务端和
// 目的地址的连接（targetConn）
func dialConn(cliConn net.Conn, cmd byte, addr, port string) (targetConn net.Conn, err error) {
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
//
// VER  	固定为 0x05，协议版本号
// REP
//		· X’00’ 成功
//		· X’01’ 普通的SOCKS服务器请求失败
//		· X’02’ 现有的规则不允许的连接
//		· X’03’ 网络不可达
//		· X’04’ 主机不可达
//		· X’05’ 连接被拒
//		· X’06’ TTL超时
//		· X’07’ 不支持的命令
//		· X’08’ 不支持的地址类型
//		· X’09’ – X’FF’ 未定义
// RSV		保留位
// ATYP		地址类型，0x01=IPv4，0x03=域名，0x04=IPv6
// BND.ADDR 服务器绑定地址
// BND.PORT	服务器绑定的端口（以网络字节序表示）
//
// 传入的 bndPort 需要保证为大端序列
func writeResponse(conn net.Conn, rep, atyp byte, bndPort, bndAddr []byte) error {
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

// WriteIPv4FailedResponse 向 cliConn 中写入 “连接建立失败” 的响应
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
