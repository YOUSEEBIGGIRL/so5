package server

import (
	"log"
	"net"
	"net/netip"

	"zz.io/cargo/so5/consts"
)

func ListenAndServer(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Println(err)
		return err
	}
	defer lis.Close()

	// 走到这里说明连接建立成功，这代表 addr 没有问题
	addrPort, err := netip.ParseAddrPort(addr)
	if err != nil {
		log.Println(err)
		return err
	}

	atypeFunc := func(addr string) byte {
		switch {
		case addrPort.Addr().Is4():
			return consts.AtypIPv4
		case addrPort.Addr().Is6():
			return consts.AtypIpv6
		default: // 因为 addr 没有问题，所以这里会是域名
			return consts.AtypDomain
		}
	}

	writeReplyPayloadFunc := func(conn net.Conn, dialTargetErr error) error {
		var rep byte
		// TODO: 暂时只支持 0x00 和 0x01
		switch dialTargetErr {
		case nil:
			rep = consts.RepSuccess
		default:
			rep = consts.RepFailed
		}

		payload, err := replyPayload(rep, atypeFunc(addr), addrPort.Addr().AsSlice(), addrPort.Port())
		if err != nil {
			return err
		}

		_, err = conn.Write(payload)
		if err != nil {
			return err
		}

		return dialTargetErr
	}

	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		go func() {
			cmd, addr, port, er := getRequest(conn)
			if er != nil {
				log.Println(er)
				return
			}

			switch cmd {
			case consts.CmdConnect:
				if err := handlerConnectCmd(conn, addr, port, writeReplyPayloadFunc); err != nil {
					log.Println(err)
					return
				}
			case consts.CmdBind:
			case consts.CmdUdp:

			}
		}()
	}
}
