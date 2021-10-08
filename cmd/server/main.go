package main

import (
	"log"
	"net"
	"github.com/zengh1/socks5"
)

func main() {
	l, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalln(err)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		log.Println(conn.RemoteAddr())

		if err := socks5.AuthClient(conn, socks5.UnamePwd); err != nil {
			log.Println(err)
			continue
		}

		targetConn, err := socks5.CreateTargetConn(conn)
		if err != nil {
			// 写入失败的响应信息
			socks5.WriteIPv4FailedResponse(conn)
			log.Println(err)
			continue
		}
		// 写入成功响应信息
		socks5.WriteIPv4SuccessResponse(conn, targetConn)

		// 连接建立成功，开始转发消息
		if err := socks5.Forward(conn, targetConn); err != nil {
			log.Println(err)
			continue
		}
	}
}