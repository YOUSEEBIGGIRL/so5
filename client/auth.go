package client

import (
	"bytes"
	"fmt"
	"io"
	"net"

	"zz.io/cargo/so5/consts"
)

// +------+------------+-----------+
// | VER  | NMETHODS   | METHODS   |
// +----- +------------+-----------+
// |  1   |     1      | 1 to 255  |
// +------+------------+-----------+

// NegotiationAuth 向服务器发送身份验证信息，服务器会查看客户支持的认证方式，从中选择一种发送给客户
// allowMethod 就是服务器指定的认证方式
func NegotiationAuth(conn net.Conn, supportAuthMethods []byte) (allowMethod byte, err error) {
	// +------+------------+-----------+
	// | VER  | NMETHODS   | METHODS   |
	// +----- +------------+-----------+
	// |  1   |     1      | 1 to 255  |
	// +------+------------+-----------+
	b := []byte{consts.Version, byte(len(supportAuthMethods))}
	for _, v := range supportAuthMethods {
		b = append(b, v)
	}

	_, err = conn.Write(b)
	if err != nil {
		return 0x00, err
	}

	// +-----+--------+
	// | VER | METHOD |
	// +-----+--------+
	// |  1  |   1    |
	// +-----+--------+
	b = make([]byte, 2)
	_, err = io.ReadFull(conn, b)
	if err != nil {
		return 0x00, fmt.Errorf("read server response error: %+v", err)
	}

	ver := b[0]
	if ver != consts.Version {
		return 0x00, fmt.Errorf("you connect server is not socks5")
	}

	allowMethod = b[1]
	return
}

// AuthUseUnamePwd 使用 用户名/密码 方式进行校验
func AuthUseUnamePwd(conn net.Conn, uname, pwd string) error {
	errch := make(chan error, 2)
	defer close(errch)

	go func() {
		errch <- writeUnameAndPwd(conn, uname, pwd)
	}()

	go func() {
		errch <- readAuthResponse(conn)
	}()

	for range 2 {
		if err := <-errch; err != nil {
			return err
		}
	}

	return nil
}

func writeUnameAndPwd(conn net.Conn, uname, pwd string) error {
	ulen := (byte)(len(uname))
	plen := (byte)(len(pwd))

	var buf bytes.Buffer

	buf.WriteByte(consts.Version)
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
	_, err := io.ReadFull(conn, b[:])
	//_, err := conn.Read(b)
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

	if ver != consts.Version {
		return fmt.Errorf("response version not socks5")
	}

	if status != consts.AuthUserOk {
		return fmt.Errorf("wrong user name or password")
	}

	return nil
}
