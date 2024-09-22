package util

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/netip"
	"strconv"

	"zz.io/cargo/so5/consts"
)

// ParseAddrFromConn 从连接中获取 ATYP
func ParseAddrFromConn(atyp byte, conn net.Conn) (addr string, err error) {
	b := make([]byte, 255)
	switch atyp {
	case consts.AtypIPv4: // IPv4
		_, err = io.ReadFull(conn, b[:4])
		if err != nil {
			return "", fmt.Errorf("parse atyp 0x01 [IPv4] addr error: %+v", err)
		}
		addr = fmt.Sprintf("%v.%v.%v.%v", b[0], b[1], b[2], b[3])
		return
	case consts.AtypDomain: // 域名
		// DST.ADDR 部分第一个字节为域名长度，DST.ADDR 剩余的内容为域名，
		// 没有 0 结尾。
		_, err = io.ReadFull(conn, b[:1])
		if err != nil {
			return "",
				fmt.Errorf("read atyp 0x03 [domain] len error: %+v", err)
		}
		domainLen := b[0]

		_, err = io.ReadFull(conn, b[:domainLen])
		if err != nil {
			return "", fmt.Errorf("parse atyp 0x03 [domain] addr error: %+v", err)
		}
		addr = string(b[:domainLen])
		return
	case consts.AtypIpv6: // IPv6，长度为 16 字节
		// TODO:
		return "", fmt.Errorf("IPv6 not support yet")
	default:
		return "", fmt.Errorf("invalid atyp")
	}
}

// ParsePortFromConn 从连接中解析出 DST.PORT
func ParsePortFromConn(conn net.Conn) (port string, err error) {
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

func ConvPortStrToBigEndianByte(p string) (uint16, error) {
	// 将字符串转换为整数
	portInt, err := strconv.Atoi(p)
	if err != nil {
		return 0, fmt.Errorf("invalid port string: %v", err)
	}

	// 将整数转换为 uint16 类型
	portUint16 := uint16(portInt)

	// 创建一个长度为 2 的 byte 切片
	b := make([]byte, 2)

	// 将 uint16 类型的数值以大端序写入 byte 切片
	binary.BigEndian.PutUint16(b, portUint16)

	// 将转换后的大端序结果返回
	return binary.BigEndian.Uint16(b), nil
}

// ParseAddr 根据 addr 获取对应的信息从而构造 requests 报文
func ParseAddr(addr string) (atyp byte, adr []byte, port uint16, err error) {
	host, p, err := net.SplitHostPort(addr)
	if err != nil {

	}

	if IsDomainName(host) {
		pp, er := ConvPortStrToBigEndianByte(p)
		if er != nil {
			err = er
			return
		}
		return consts.AtypDomain, []byte(host), pp, nil
	}

	addrPort, err := netip.ParseAddrPort(addr)
	if err != nil {
		return 0, nil, 0, err
	}

	switch {
	case addrPort.Addr().Is4():
		atyp = consts.AtypIPv4
	case addrPort.Addr().Is6():
		atyp = consts.AtypIpv6
	}

	return atyp, addrPort.Addr().AsSlice(), addrPort.Port(), nil
}

// IsDomainName checks if a string is a presentation-format domain name
// (currently restricted to hostname-compatible "preferred name" LDH labels and
// SRV-like "underscore labels"; see golang.org/issue/12421).
func IsDomainName(s string) bool {
	// The root domain name is valid. See golang.org/issue/45715.
	if s == "." {
		return true
	}

	// See RFC 1035, RFC 3696.
	// Presentation format has dots before every label except the first, and the
	// terminal empty label is optional here because we assume fully-qualified
	// (absolute) input. We must therefore reserve space for the first and last
	// labels' length octets in wire format, where they are necessary and the
	// maximum total length is 255.
	// So our _effective_ maximum is 253, but 254 is not rejected if the last
	// character is a dot.
	l := len(s)
	if l == 0 || l > 254 || l == 254 && s[l-1] != '.' {
		return false
	}

	last := byte('.')
	nonNumeric := false // true once we've seen a letter or hyphen
	partlen := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		default:
			return false
		case 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || c == '_':
			nonNumeric = true
			partlen++
		case '0' <= c && c <= '9':
			// fine
			partlen++
		case c == '-':
			// Byte before dash cannot be dot.
			if last == '.' {
				return false
			}
			partlen++
			nonNumeric = true
		case c == '.':
			// Byte before dot cannot be dot, dash.
			if last == '.' || last == '-' {
				return false
			}
			if partlen > 63 || partlen == 0 {
				return false
			}
			partlen = 0
		}
		last = c
	}
	if last == '-' || partlen > 63 {
		return false
	}

	return nonNumeric
}
