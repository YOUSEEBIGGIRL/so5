package client

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"github.com/zengh1/socks5"
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

func WriteRequestIP4(conn net.Conn,
	targetIP4 []byte, targetPort uint16) error {
	// +----+-----+-------+------+----------+----------+
	// |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
	// +----+-----+-------+------+----------+----------+
	// | 1  |  1  | X'00' |  1   | Variable |    2     |
	// +----+-----+-------+------+----------+----------+

	var b bytes.Buffer
	b.WriteByte(Version)
	b.WriteByte(CmdConnect)
	b.WriteByte(RSV)
	b.WriteByte(AtypIPv4)
	// 目的地址为 127.0.0.1:8888
	b.Write(targetIP4) // DST.ADDR

	pp := make([]byte, 2)
	// 以大端的方式将 8888 转换为 2 字节
	binary.BigEndian.PutUint16(pp, targetPort)
	b.Write(pp)

	_, err := conn.Write(b.Bytes())
	if err != nil {
		return fmt.Errorf("write request to conn error: %+v", err)
	}

	return nil
}

func ReadResponse(conn net.Conn) (addr, port string, err error) {
	buf := make([]byte, 255)

	// +-----+-----+-------+------+----------+----------+
	// | VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
	// +-----+-----+-------+------+----------+----------+
	// |  1  |  1  | X'00' |  1   | Variable |    2     |
	// +-----+-----+-------+------+----------+----------+

	// VER
	_, err = io.ReadFull(conn, buf[:1])
	if err != nil {
		return "", "", fmt.Errorf("read VER error")
	}
	ver := buf[0]
	//log.Printf("ver: %v \n", ver)

	// REP
	_, err = io.ReadFull(conn, buf[:1])
	if err != nil {
		return "", "", fmt.Errorf("read REP error")
	}
	rep := buf[0]
	if rep != RepSuccess {
		return "", "", fmt.Errorf("create conn to target addr error")
	}

	// RSV
	_, err = io.ReadFull(conn, buf[:1])
	if err != nil {
		return "", "", fmt.Errorf("read RSV error")
	}
	rsv := buf[0]
	//fmt.Printf("rsv: %v\n", rsv)

	// ATYP
	_, err = io.ReadFull(conn, buf[:1])
	if err != nil {
		return "", "", fmt.Errorf("read RSV error")
	}
	atyp := buf[0]
	//fmt.Printf("atyp: %v\n", atyp)

	// BND.ADDR
	addr, err = socks5.ParseAddr(atyp, conn)
	if err != nil {
		return
	}
	//fmt.Printf("addr: %v\n", addr)

	port, err = socks5.ParsePort(conn)
	if err != nil {
		return
	}
	//fmt.Printf("port: %v\n", port)

	log.Printf("ver: %v, rep: %v, rsv: %v, atyp: %v addr: %v, port: %v \n",
		ver, rep, rsv, atyp, addr, port)
	return
}


