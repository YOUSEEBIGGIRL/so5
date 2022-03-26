package main

import (
	"bytes"
	"io"
	"log"
	"net"
	"sync"
	"testing"
	"time"
)

// =============== test for no auth required ===============
func TestAuthNoAuthRequired(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(2)
	// server
	go func() {
		defer wg.Done()
		l, err := net.Listen("tcp", ":8080")
		if err != nil {
			log.Fatalln(err)
		}

		for {
			conn, err := l.Accept()
			if err != nil {
				log.Fatalln(err)
			}

			if err := AuthClient(conn, NoAuthRequired); err != nil {
				log.Fatalln(err)
			}
		}
	}()

	time.Sleep(time.Second)

	// client
	go func() {
		defer wg.Done()
		conn, err := net.Dial("tcp", ":8080")
		if err != nil {
			log.Fatalln(err)
		}

		_, err = conn.Write([]byte{Version, 1, NoAuthRequired})
		if err != nil {
			log.Fatalln(err)
		}

		b := make([]byte, 2)
		_, err = conn.Read(b)
		if err != nil {
			log.Fatalln(err)
		}
		if b[0] != Version || b[1] != NoAuthRequired {
			log.Fatalf("test error: want return 0x5(sock5 version), 0x0(no auth), get %v, %v\n", b[0], b[1])
		}
		log.Println("test ok, result: ", b)
	}()

	wg.Wait()
}

// =============== test for username/password ===============
func TestAuthUnamePwd(t *testing.T) {
	l, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalln(err)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatalln(err)
		}

		if err := AuthClient(conn, UnamePwd); err != nil {
			log.Fatalln(err)
		}
	}
}

func TestClientUnamePwd(t *testing.T) {
	conn, err := net.Dial("tcp", ":8080")
	if err != nil {
		log.Fatalln(err)
	}

	_, err = conn.Write([]byte{Version, 2, NoAuthRequired, UnamePwd})
	if err != nil {
		log.Fatalln(err)
	}

	b := make([]byte, 2)
	_, err = io.ReadFull(conn, b)
	if err != nil {
		log.Fatalln(err)
	}
	//log.Println(b)

	needMethod := b[1]

	if needMethod == UnamePwd {
		uname := "abc"
		pwd := "123" // true pwd
		ulen := (byte)(len(uname))
		plen := (byte)(len(pwd))

		// 写入数据格式如下：
		// +----+------+----------+------+----------+
		// |VER | ULEN |  UNAME   | PLEN |  PASSWD  |
		// +----+------+----------+------+----------+
		// | 1  |  1   | 1 to 255 |  1   | 1 to 255 |
		// +----+------+----------+------+----------+
		var buf bytes.Buffer
		buf.WriteByte(Version)
		buf.WriteByte(ulen)
		buf.WriteString(uname)
		buf.WriteByte(plen)
		buf.WriteString(pwd)

		_, err = conn.Write(buf.Bytes())
		if err != nil {
			log.Fatalln(err)
		}

		b := make([]byte, 2)
		_, err := io.ReadFull(conn, b)
		if err != nil {
			log.Fatalln(err)
		}
		if b[0] != Version {
			log.Fatalf("test error: server return version is %v, not 0x5", b[0])
		}
		if b[0] != AuthUserOk {
			log.Fatalf("test error: username or password wrong.")
		}
		log.Println("test ok")
	}
}
