package socks5

import (
	"bytes"
	"encoding/binary"
	//"encoding/binary"
	"io"
	"log"
	"net"
	"testing"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

const (
	proxyPort            = ":8080"
	targetPortInt uint16 = 8888
	targetPort           = ":8888"
)

func TestGetTargetConn(t *testing.T) {
	l, err := net.Listen("tcp", proxyPort)
	if err != nil {
		log.Fatalln(err)
	}

	for {
		cliConn, err := l.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		tarConn, err := getTargetConn(cliConn)
		if err != nil {
			// 写入失败的响应信息
			writeIPv4FailedResponse(cliConn)
			log.Println(err)
			continue
		}
		writeIPv4SuccessResponse(cliConn, tarConn)
		log.Println(tarConn.LocalAddr())
	}
}

func TestClient(t *testing.T) {
	conn, err := net.Dial("tcp", proxyPort)
	if err != nil {
		log.Fatalln(err)
	}

	// +----+-----+-------+------+----------+----------+
	// |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
	// +----+-----+-------+------+----------+----------+
	// | 1  |  1  | X'00' |  1   | Variable |    2     |
	// +----+-----+-------+------+----------+----------+

	var b bytes.Buffer
	b.WriteByte(Version)
	b.WriteByte(CmdConnect)
	b.WriteByte(RSV)
	b.WriteByte(AtypIPv4)
	// 目的地址为 127.0.0.1:8888
	b.Write([]byte{127, 0, 0, 1}) // DST.ADDR

	pp := make([]byte, 2)
	// 以大端的方式将 8888 转换为 2 字节
	binary.BigEndian.PutUint16(pp, targetPortInt)
	b.Write(pp)

	_, err = conn.Write(b.Bytes())
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("write ok")

	err = __readResponse__(conn)
	if err != nil {
		log.Fatalln(err)
	}
}

// +-----+-----+-------+------+----------+----------+
// | VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
// +-----+-----+-------+------+----------+----------+
// |  1  |  1  | X'00' |  1   | Variable |    2     |
// +-----+-----+-------+------+----------+----------+
func __readResponse__(conn net.Conn) error {
	buf := make([]byte, 255)

	_, err := io.ReadFull(conn, buf[:1])
	if err != nil {
		return err
	}
	ver := buf[0]
	_ = ver

	_, err = io.ReadFull(conn, buf[:1])
	if err != nil {
		return err
	}
	rep := buf[0]
	log.Println("rep: ", rep) // 0x00 成功 其他代表失败

	return nil
}

// 目的服务器，对客户端进行回声处理
func TestTargetServer(t *testing.T) {
	l, err := net.Listen("tcp", targetPort)
	if err != nil {
		log.Fatalln(err)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		add := conn.RemoteAddr()
		log.Printf("[%v] is coming", add)

		b := make([]byte, 1024)

		// echo
		n, err := conn.Read(b)
		if err != nil {
			log.Println(err)
			continue
		}
		log.Printf("read from [%v]: %v \n", add, string(b))

		_, err = conn.Write(b[:n])
		if err != nil {
			log.Println(err)
			continue
		}
	}
}
