package server

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"zz.io/cargo/so5/consts"
	"zz.io/cargo/so5/util"
)

func handlerConnectCmd(conn net.Conn, addr, port string, f func(conn net.Conn, err error) error) (err error) {
	// 获取目的服务器的连接
	targetConn, err := net.Dial("tcp", net.JoinHostPort(addr, port))
	// write reply to client
	if err := f(conn, err); err != nil {
		return err
	}

	defer conn.Close()
	defer targetConn.Close()

	go func() {
		_, er := io.Copy(targetConn, conn)
		if er != nil {
			err = er
			return
		}
	}()

	if _, err := io.Copy(conn, targetConn); err != nil {
		return err
	}

	return nil
}

// SOCKS 的请求构成如下：（参见 RFC 1928，4. Requests）
// +----+-----+-------+------+----------+----------+
// |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
// +----+-----+-------+------+----------+----------+
// | 1  |  1  | X'00' |  1   | Variable |    2     |
// +----+-----+-------+------+----------+----------+
// VER		0x05，协议版本号
// CMD		连接方式，0x01=CONNECT, 0x02=BIND, 0x03=UDP ASSOCIATE
// RSV		保留字段，目前没用，值为 0x00
// ATYP		地址类型，0x01=IPv4，0x03=域名，0x04=IPv6
// DST.ADDR	目标地址
// DST.PORT	目标端口，2 字节，网络字节序（network octec order）
func getRequest(conn net.Conn) (cmd byte, addr, port string, err error) {
	b := make([]byte, 255)

	_, err = io.ReadFull(conn, b[:4])
	if err != nil {
		err = fmt.Errorf("read header[VER, CMD, RSV, ATYP] error: %+v", err)
		return
	}

	ver, cmd, rsv, atyp := b[0], b[1], b[2], b[3]
	if ver != consts.Version {
		err = fmt.Errorf("invalid version")
		return
	}

	_ = rsv // 忽略保留字段

	addr, err = util.ParseAddrFromConn(atyp, conn)
	if err != nil {
		return
	}

	port, err = util.ParsePortFromConn(conn)
	if err != nil {
		return
	}

	return
}

// +-----+-----+-------+------+----------+----------+
// | VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
// +-----+-----+-------+------+----------+----------+
// |  1  |  1  | X'00' |  1   | Variable |    2     |
// +-----+-----+-------+------+----------+----------+
//
// VER  	固定为 0x05，协议版本号
// REP
//
//	· X’00’ 成功
//	· X’01’ 普通的SOCKS服务器请求失败
//	· X’02’ 现有的规则不允许的连接
//	· X’03’ 网络不可达
//	· X’04’ 主机不可达
//	· X’05’ 连接被拒
//	· X’06’ TTL超时
//	· X’07’ 不支持的命令
//	· X’08’ 不支持的地址类型
//	· X’09’ – X’FF’ 未定义
//
// RSV		保留位
// ATYP		地址类型，0x01=IPv4，0x03=域名，0x04=IPv6
// BND.ADDR 服务器绑定地址
// BND.PORT	服务器绑定的端口（以网络字节序表示）
//
// 传入的 bndPort 需要保证为大端序列
func replyPayload(rep, atyp byte, bndAddr []byte, bndPort uint16) ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte(consts.Version)
	b.WriteByte(rep)
	b.WriteByte(0x00) // RSV
	b.WriteByte(atyp)
	b.Write(bndAddr)
	// Write the BND.PORT
	err := binary.Write(&b, binary.BigEndian, bndPort)
	if err != nil {
		return nil, fmt.Errorf("failed to write port: %w", err)
	}

	return b.Bytes(), nil
}
