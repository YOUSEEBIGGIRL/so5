package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
)

var (
	proxyAddr  = flag.String("p", "", "proxy server addr")
	targetAddr = flag.String("t", "", "target server addr")
	//username   = flag.String("u", "", "username(if need)")
	//password   = flag.String("pwd", "", "password(if need)")
)

func main() {
	flag.Parse()
	if *proxyAddr == "" || *targetAddr == "" {
		fmt.Printf("usage: ./main -p [host:port] -t [host:port] -u [username] -pwd [passowrd]")
		return
	}
	conn, err := net.Dial("tcp", *proxyAddr)
	if err != nil {
		log.Println(err)
		return
	}

	allowMethod, err := Auth(conn)
	if err != nil {
		log.Println("handshake with the server error: ", err)
		return
	}

	switch allowMethod {
	case AuthMethodNoNeed:
	case AuthMethodUnamePwd:
		var uname, pwd string
		fmt.Println("this server need auth username")
		fmt.Println("please input username: ")
		if _, err := fmt.Scan(&uname); err != nil {
			fmt.Println("username scan error: ", err)
			return
		}
		fmt.Println("please input password: ")
		if _, err := fmt.Scan(&pwd); err != nil {
			fmt.Println("password scan error: ", err)
			return
		}
		if err := AuthUseUnamePwd(conn, uname, pwd); err != nil {
			log.Println("auth username error: ", err)
		}
		log.Println("auth success")
	}

	s := strings.Split(*targetAddr, ":")
	if len(s) != 2 {
		log.Fatalln("targetAddr format error, correct format: 127.0.0.1:8080")
	}
	host := s[0]
	port := s[1]
	portuint16, _ := strconv.ParseUint(port, 10, 16)

	if err := WriteRequestIP4(conn, []byte(host), uint16(portuint16)); err != nil {
		log.Println("send request error: ", err)
	}

	addr, port, err := ReadResponse(conn)
	if err != nil {
		log.Println(err)
		return
	}
	_, _ = addr, port

	for {
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
