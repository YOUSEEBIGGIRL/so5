package client

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"

	"zz.io/cargo/so5/consts"
	"zz.io/cargo/so5/util"
)

// ListenAndServer
// 客户端运行命令 example:
// so5 client --listen-addr=127.0.0.1:8080 --proxy-addr=127.0.0.1:8088 --target-addr=127.0.0.1:9090
// 其他应用通过 127.0.0.1 与 socks5 client 建立连接，然后 socks5 client 转发应用的请求
// 所以这里的 client 实际上即是 server（对应用而言），也是 client（对 socks5 server 而言）
func ListenAndServer(addr, proxyAddr, targetAddr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer lis.Close()

	for {
		conn, err := lis.Accept()
		if err != nil {
			return err
		}
		log.Printf("accepted new connection, addr %v", conn.RemoteAddr())

		go func() {
			if err := Dial(conn, proxyAddr, targetAddr); err != nil {
				io.WriteString(conn, err.Error())
				return
			}
		}()

	}
}

func Dial(conn net.Conn, addr, targetAddr string) error {
	if conn == nil {
		return nil
	}
	defer conn.Close()

	targetConn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	defer targetConn.Close()

	// TODO: Auth
	//authMethod, err := NegotiationAuth(conn, supportAuthMethods)
	//if err != nil {
	//	return err
	//}
	//
	//switch authMethod {
	//case consts.AuthTypeNoRequired:
	//	t.Log("NegotiationAuthMethod: NoRequired")
	//case consts.AuthTypeUnamePwd:
	//	if err := client.AuthUseUnamePwd(conn, server.Username, server.Password); err != nil {
	//		t.Error(err)
	//		return
	//	}
	//case consts.AuthTypeNoAcceptable:
	//	t.Log("NegotiationAuthMethod: NoAcceptable")
	//}

	atyp, adr, port, err := util.ParseAddr(targetAddr)
	if err != nil {
		return err
	}

	if err := WriteRequest(targetConn, atyp, adr, port); err != nil {
		return err
	}

	_, _, _, err = ReadReplyResponse(targetConn)
	if err != nil {
		return err
	}

	go func() {
		if _, er := io.Copy(targetConn, conn); er != nil {
			err = er
			log.Println(err)
			return
		}
	}()

	if _, err := io.Copy(conn, targetConn); err != nil {
		log.Println(err)
		return err
	}

	return err
}

func WriteRequest(conn net.Conn, atyp byte, addr []byte, targetPort uint16) error {
	// +----+-----+-------+------+----------+----------+
	// |VER | CMD |  RSV  | ATYP | DST.ADDR | DST.PORT |
	// +----+-----+-------+------+----------+----------+
	// | 1  |  1  | X'00' |  1   | Variable |    2     |
	// +----+-----+-------+------+----------+----------+

	var b bytes.Buffer
	b.WriteByte(consts.Version)
	b.WriteByte(consts.CmdConnect)
	b.WriteByte(consts.RSV)
	b.WriteByte(atyp)
	b.Write(addr) // DST.ADDR

	pp := make([]byte, 2)
	// 以大端的方式将 8888 转换为 2 字节
	binary.BigEndian.PutUint16(pp, targetPort)
	b.Write(pp)

	_, err := conn.Write(b.Bytes())
	if err != nil {
		errMsg := fmt.Errorf("write request to conn error: %+v", err)
		log.Println(errMsg)
		return errMsg
	}

	return nil
}

func ReadReplyResponse(conn net.Conn) (atyp byte, addr, port string, err error) {
	buf := make([]byte, 255)

	// +-----+-----+-------+------+----------+----------+
	// | VER | REP |  RSV  | ATYP | BND.ADDR | BND.PORT |
	// +-----+-----+-------+------+----------+----------+
	// |  1  |  1  | X'00' |  1   | Variable |    2     |
	// +-----+-----+-------+------+----------+----------+

	// VER
	_, err = io.ReadFull(conn, buf[:1])
	if err != nil {
		log.Println(err)
		if errors.Is(err, io.EOF) {
			return 0, "", "", nil
		}
		return 0, "", "", fmt.Errorf("read reply.VER error: %v", err)
	}
	ver := buf[0]

	// REP
	_, err = io.ReadFull(conn, buf[:1])
	if err != nil {
		return 0, "", "", fmt.Errorf("read reply.REP error")
	}
	rep := buf[0]
	if rep != consts.RepSuccess {
		return 0, "", "", fmt.Errorf("create conn to target addr error, REP: %d", rep)
	}

	// RSV
	_, err = io.ReadFull(conn, buf[:1])
	if err != nil {
		return 0, "", "", fmt.Errorf("read RSV error")
	}
	rsv := buf[0]
	//fmt.Printf("rsv: %v\n", rsv)

	// ATYP
	_, err = io.ReadFull(conn, buf[:1])
	if err != nil {
		return 0, "", "", fmt.Errorf("read ATYP error")
	}
	atyp = buf[0]
	//fmt.Printf("atyp: %v\n", atyp)

	// BND.ADDR
	addr, err = util.ParseAddrFromConn(atyp, conn)
	if err != nil {
		return
	}
	//fmt.Printf("addr: %v\n", addr)

	port, err = util.ParsePortFromConn(conn)
	if err != nil {
		return
	}
	//fmt.Printf("port: %v\n", port)

	log.Printf("ver: %v, rep: %v, rsv: %v, atyp: %v addr: %v, port: %v \n",
		ver, rep, rsv, atyp, addr, port)
	return
}
