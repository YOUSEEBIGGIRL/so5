package main

import (
	"flag"
	"fmt"
	"log"
	"net"
)

var addr = flag.String("addr", "", "listen addr")
var auth = flag.String("auth", "n",
	`If you enter n, the server will not authenticate, if you enter y, 
			the server will authenticate with username/password`)

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	flag.Parse()
	if *addr == "" {
		fmt.Printf("usage: ./main -addr [host:port]\n")
		return
	}
	var m Method
	switch *auth {
	case "n":
		m = NoAuthRequired
	case "y":
		m = UnamePwd
	default:
		fmt.Printf("-m input error, please input NoAuth or UnamePwd")
		return
	}

	l, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("listen %v error: %v\n", *addr, err)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println("accept error: ", err)
			continue
		}
		log.Printf("a new connect from [%v]\n", conn.RemoteAddr())

		if err := AuthClient(conn, m); err != nil {
			//log.Println("auth error: ", err)
			continue
		}

		targetConn, err := CreateTargetConn(conn)
		if err != nil {
			// 写入失败的响应信息
			if err := WriteIPv4FailedResponse(conn); err != nil {
				//log.Println("write response to client conn error: ", err)
				continue
			}
			log.Println("create target conn error: ", err)
			continue
		}
		// 写入成功响应信息
		if err := WriteIPv4SuccessResponse(conn, targetConn); err != nil {
			//log.Println("write response to client conn error: ", err)
			continue
		}

		// 连接建立成功，开始转发消息
		if err := Forward(conn, targetConn); err != nil {
			log.Println("forward error: ", err)
			continue
		}
	}
}
