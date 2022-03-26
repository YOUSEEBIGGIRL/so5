package util

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

const (
	AtypIPv4   = 0x01
	AtypDomain = 0x03
	AtypIpv6   = 0x04
)

// ParseAddr 根据 ATYP 获得客户要连接的地址
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

// ParsePort 解析出 DST.PORT
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
	// 使用 string(p) 得到的是 p 对应的 ASCII 字符，如果想转换为相同的 string 应该使用 fmt.Sprint(p)
	port = fmt.Sprint(p)
	return
}
