package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"testing"
)

func TestForward(t *testing.T) {
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

		tarConn, err := CreateTargetConn(cliConn)
		if err != nil {
			// 写入失败的响应信息
			WriteIPv4FailedResponse(cliConn)
			log.Println(err)
			continue
		}
		WriteIPv4SuccessResponse(cliConn, tarConn)
		log.Println(tarConn.LocalAddr())

		// forward 数据
		if err := Forward(cliConn, tarConn); err != nil {
			log.Println(err)
			continue
		}
	}
}

func TestForwardClient(t *testing.T) {
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

	rep, err := __readResponse1__(conn)
	if err != nil {
		log.Fatalln(err)
	}

	if rep != 0 {
		log.Println("rep fail")
		return
	}

	// rep == 0，可以开始发送数据了
	msg := "123"
	_, err = conn.Write([]byte(msg))
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("send: ", msg)

	bb := make([]byte, 1024)
	conn.Read(bb)
	log.Println("recv: ", string(bb))
}

func __readResponse1__(conn net.Conn) (rep byte, err error) {
	buf := make([]byte, 255)

	// +-----+-----+-------+------+----------+----------+
	// | VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
	// +-----+-----+-------+------+----------+----------+
	// |  1  |  1  | X'00' |  1   | Variable |    2     |
	// +-----+-----+-------+------+----------+----------+

	// VER
	_, err = io.ReadFull(conn, buf[:1])
	if err != nil {
		return
	}
	ver := buf[0]
	log.Printf("ver: %v \n", ver)

	// REP
	_, err = io.ReadFull(conn, buf[:1])
	if err != nil {
		return
	}
	rep = buf[0]
	log.Println("rep: ", rep) // 0x00 成功 其他代表失败

	// RSV
	_, err = io.ReadFull(conn, buf[:1])
	if err != nil {
		return
	}
	rsv := buf[0]
	fmt.Printf("rsv: %v\n", rsv)

	// ATYP
	_, err = io.ReadFull(conn, buf[:1])
	if err != nil {
		return
	}
	atyp := buf[0]
	fmt.Printf("atyp: %v\n", atyp)

	// BND.ADDR
	addr, err := ParseAddr(atyp, conn)
	if err != nil {
		return
	}
	fmt.Printf("addr: %v\n", addr)

	port, err := ParsePort(conn)
	if err != nil {
		return
	}
	fmt.Printf("port: %v\n", port)

	log.Printf("ver: %v, rep: %v, rsv: %v, atyp: %v addr: %v, port: %v \n",
		ver, rep, rsv, atyp, addr, port)
	return
}
