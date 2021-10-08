package client

import (
	"bytes"
	"fmt"
	"io"
	"net"
)

const (
	Version      = 0x05
	UnamePwdAuth = 0x02
	AuthUserOk   = 0x00 // 用户验证成功
	AuthUserFail = 0x01 // 用户验证失败（非 0）
)

// 使用 用户名/密码 方式进行校验
func AuthUseUnamePwd(conn net.Conn, uname, pwd string) error {
	// +------+------------+-----------+
	// | VER  | NMETHODS   | METHODS   |
	// +----- +------------+-----------+
	// |  1   |     1      | 1 to 255  |
	// +------+------------+-----------+
	_, err := conn.Write([]byte{Version, 1, UnamePwdAuth})
	if err != nil {
		return err
	}

	// +----+--------+
	// |VER | METHOD |
	// +----+--------+
	// | 1  |   1    |
	// +----+--------+
	b := make([]byte, 2)
	_, err = io.ReadFull(conn, b)
	if err != nil {
		return fmt.Errorf("read server response error: %+v", err)
	}

	ver := b[0]
	if ver != 5 {
		return fmt.Errorf("you connect server is not socks5")
	}

	method := b[1]
	if method == UnamePwdAuth {
		if err := writeUnameAndPwd(conn, uname, pwd); err != nil {
			return err
		}
	}

	if err := readAuthResponse(conn); err != nil {
		return err
	}

	return nil
}

func writeUnameAndPwd(conn net.Conn, uname, pwd string) error {
	ulen := (byte)(len(uname))
	plen := (byte)(len(pwd))

	var buf bytes.Buffer

	buf.WriteByte(Version)
	buf.WriteByte(ulen)
	buf.WriteString(uname)
	buf.WriteByte(plen)
	buf.WriteString(pwd)

	_, err := conn.Write(buf.Bytes())
	if err != nil {
		return err
	}

	return nil
}

func readAuthResponse(conn net.Conn) error {
	b := make([]byte, 2)
	_, err := conn.Read(b)
	if err != nil {
		return fmt.Errorf("read response error: %+v", err)
	}

	// +----+--------+
	// |VER | STATUS |
	// +----+--------+
	// | 1  |   1    |
	// +----+--------+
	ver := b[0]
	status := b[1]

	if ver != 5 {
		return fmt.Errorf("response version not socks5")
	}

	if status != AuthUserOk {
		return fmt.Errorf("wrong user name or password")
	}

	return nil
}
