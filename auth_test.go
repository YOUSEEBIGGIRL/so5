package socket5

import (
	"bytes"
	"io"
	"log"
	"net"
	"testing"
)

// =============== test for no auth required ===============
func TestAuthNoAuthRequired(t *testing.T) {
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

}

func TestClientNoAuthRequired(t *testing.T) {
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
	log.Println(b)
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
		pwd := "123"	// true pwd
		ulen := (byte)(len(uname))
		plen := (byte)(len(pwd))

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
		_, err := conn.Read(b)
		if err != nil {
			log.Fatalln(err)
		}
		log.Println(b)
	}
}