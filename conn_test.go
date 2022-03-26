package main

import (
	"bytes"
	"encoding/binary"
	"sync"
	"time"

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

func TestConn(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(3)
	go func() { defer wg.Done(); createProxyServer() }()
	time.Sleep(time.Second)
	go func() { defer wg.Done(); createTargetServer() }()
	time.Sleep(time.Second)
	go func() { defer wg.Done(); createClient() }()
	time.Sleep(time.Second)
	wg.Wait()
}

// 该服务作为代理(proxy)，客户链接到该服务，该服务再将客户请求转发到目的服务(target)
func createProxyServer() {
	l, err := net.Listen("tcp", proxyPort)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("[proxyServer] now started, addr: %v\n", proxyPort)

	for {
		cliConn, err := l.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		tarConn, err := CreateTargetConn(cliConn)
		if err != nil {
			log.Println(err)
			// 写入失败的响应信息
			if err := WriteIPv4FailedResponse(cliConn); err != nil {
				log.Println(err)
			}
			continue
		}
		if err := WriteIPv4SuccessResponse(cliConn, tarConn); err != nil {
			log.Fatalln(err)
		}
		log.Println("[proxyServer]target connect addr: ", tarConn.RemoteAddr())
		// 建立连接
		if err := Forward(cliConn, tarConn); err != nil {
			log.Println("forward error: ", err)
		}
	}
}

func createClient() {
	conn, err := net.Dial("tcp", proxyPort)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("[client] now started, addr: %v\n", conn.LocalAddr())

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
	log.Println("[client] write request ok")

	err = readCreateConnectResponse(conn)
	if err != nil {
		log.Fatalln(err)
	}
	//log.Printf("bind addr: %v, bind port: %v\n", addr, port)

	conn.Write([]byte("123"))
	bb := make([]byte, 1024)
	n, _ := conn.Read(bb)
	log.Println("[client] read from proxyServer: ", string(bb[:n]))
}

// +-----+-----+-------+------+----------+----------+
// | VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
// +-----+-----+-------+------+----------+----------+
// |  1  |  1  | X'00' |  1   | Variable |    2     |
// +-----+-----+-------+------+----------+----------+
func readCreateConnectResponse(conn net.Conn) error {
	buf := make([]byte, 255)

	_, err := io.ReadFull(conn, buf[:1])
	if err != nil {
		return err
	}
	ver := buf[0]

	_, err = io.ReadFull(conn, buf[:1])
	if err != nil {
		return err
	}
	rep := buf[0]

	_, err = io.ReadFull(conn, buf[:1])
	if err != nil {
		return err
	}
	rsv := buf[0]

	_, err = io.ReadFull(conn, buf[:1])
	if err != nil {
		return err
	}
	atyp := buf[0]

	addr, err := ParseAddr(atyp, conn)
	if err != nil {
		return err
	}

	port, err := ParsePort(conn)
	if err != nil {
		return err
	}

	log.Printf("ver: %v, rep: %v, rsv: %v, atyp: %v, addr: %v, port: %v\n", ver, rep, rsv, atyp, addr, port)
	return nil
}

// 目的服务器，对客户端进行回声处理
func createTargetServer() {
	l, err := net.Listen("tcp", targetPort)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("[targetServer] now started, addr: %v\n", targetPort)

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		add := conn.RemoteAddr()
		log.Printf("[targetServer][%v] is coming", add)

		b := make([]byte, 1024)

		// echo
		n, err := conn.Read(b)
		if err != nil {
			log.Println(err)
			continue
		}
		log.Printf("[targetServer]read from [%v]: %v \n", add, string(b))

		_, err = conn.Write(b[:n])
		if err != nil {
			log.Println(err)
			continue
		}
	}
}
