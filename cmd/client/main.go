package main

import (
	"log"
	"net"
	"github.com/zengh1/socks5/client"
)

func main() {
	conn, err := net.Dial("tcp", ":8080")
	if err != nil {
		log.Println(err)
		return
	}

	for {
		if err := client.AuthUseUnamePwd(conn, "abc", "123"); err != nil {
			log.Println(err)
			continue
		}
		log.Println("auth success")

		if err := client.WriteRequestIP4(conn, []byte{127, 0, 0, 1}, 8888); err != nil {
			log.Println(err)
			continue
		}

		addr, port, err := client.ReadResponse(conn)
		if err != nil {
			log.Println(err)
			continue
		}
		_, _ = addr, port

		// test
		_, err = conn.Write([]byte("123"))
		if err != nil {
			log.Println(err)
			continue
		}

		b := make([]byte, 1024)
		_, err = conn.Read(b)
		if err != nil {
			log.Println(err)
			continue
		}
		log.Println(string(b))
	}
}