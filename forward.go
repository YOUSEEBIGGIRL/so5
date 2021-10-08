package socks5

import (
	"io"
	"net"
)

func Forward(cliConn, targetConn net.Conn) (err error) {
	fn := func(dst, src net.Conn) error {
		_, err := io.Copy(dst, src)
		return err
	}

	go func() {
		err = fn(targetConn, cliConn) // 将客户端数据发送到目的服务器
	}()

	go func() {
		err = fn(cliConn, targetConn) // 将目的服务器响应发送给客户端
	}()

	return
}
