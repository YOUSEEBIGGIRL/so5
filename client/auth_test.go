package client

import (
	"log"
	"net"
	"github.com/zengh1/socks5"
	"testing"
)

func TestAuthServer(t *testing.T) {
	l, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalln(err)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatalln(err)
		}

		if err := socks5.AuthClient(conn, socks5.UnamePwd); err != nil {
			log.Fatalln(err)
		}
	}
}

func TestAuthCli(t *testing.T) {
	conn, err := net.Dial("tcp", ":8080")
	if err != nil {
		log.Fatalln(err)
	}

	trueUname := "abc"
	truePwd := "123"

	//wrongPwd := "1234"

	if err := AuthUseUnamePwd(conn, trueUname, truePwd); err != nil {
		log.Fatalln(err)
	}
}
