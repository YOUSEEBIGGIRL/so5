package server

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"

	"zz.io/cargo/so5/consts"
)

const (
	Username = "root"
	Password = "root"
)

// RFC 1928, https://www.ietf.org/rfc/rfc1928.txt

// NegotiationAuth 会对客户的身份进行验证，客户发送的内容格式为：
// +------+------------+-----------+
// | VER  | NMETHODS   | METHODS   |	字段名
// +----- +------------+-----------+
// |  1   |     1      | 1 to 255  |	字节数
// +------+------------+-----------+
//
// VER		本次请求的协议版本号，取固定值 0x05（表示 socks 5）
// NMETHODS	客户端支持的认证方式数量，可取值 1~255
// METHODS	可用的认证方式列表
//
// 目前支持的验证方式一共有:
// 0x00 无验证需求
// 0x01 通用安全服务应用程序接口(GSSAPI)
// 0x02 用户名/密码(USERNAME/PASSWORD)
// 0x03 至 X’7F’ IANA 分配(IANA ASSIGNED)
// 0x80 至 X’FE’ 私人方法保留(RESERVED FOR PRIVATE METHODS)
// 0xFF 无可接受方法(NO ACCEPTABLE METHODS)
// method 由服务提供者自行定义
func NegotiationAuth(conn net.Conn, method byte) error {
	buf := make([]byte, 255)

	// 使用 ReadFull 保证读满 2 字节的数据，否则返回错误
	_, err := io.ReadFull(conn, buf[:2])
	if err != nil {
		return errors.New("read header[ver, nmethods] error: " + err.Error())
	}

	ver := buf[0]
	nmethods := buf[1]
	log.Printf("ver: %v, nmethods: %v\n", ver, nmethods)

	// socket 版本必须为 5
	if ver != consts.Version {
		errMsg := "invalid version"
		log.Println(errMsg)
		return errors.New(errMsg)
	}

	// 将用户支持的验证方法全部读出来
	_, err = io.ReadFull(conn, buf[:nmethods])
	if err != nil {
		errMsg := "read methods error: " + err.Error()
		log.Println(errMsg)
		return errors.New(errMsg)
	}

	// 将用户支持的验证方法保存到 map 中，主要用于服务端回复检测
	clientSupportAutoMethod := make(map[byte]struct{})
	for i := 0; i < int(nmethods); i++ {
		clientSupportAutoMethod[buf[i]] = struct{}{}
	}

	// 服务端提供的验证方法不被客户端接受，客户端收到该报文后应当断开连接
	if _, ok := clientSupportAutoMethod[method]; !ok {
		_, err := conn.Write([]byte{consts.Version, consts.AuthTypeNoAcceptable})
		if err != nil {
			log.Println(err)
			return err
		}
	}

	switch method {
	case consts.AuthTypeNoRequired:
		if err := NoAuthRequireHandler(conn); err != nil {
			return err
		}
	case consts.AuthTypeUnamePwd:
		if err := UnamePwdHandler(conn); err != nil {
			return err
		}
	}

	return nil
}

// 客户身份验证通过后，服务端会查看客户支持的认证方式，从中选择一种发送给客户，
// 表示需要客户使用此方式进行验证，回复的报文格式为：
// +-----+--------+
// | VER | METHOD |
// +-----+--------+
// |  1  |   1    |
// +-----+--------+
//
// VER		本次请求的协议版本号，取固定值 0x05（表示 socks 5）
// METHOD	服务端选定的验证方式
//

// NoAuthRequireHandler 回复客户端，连接不需要进行验证
func NoAuthRequireHandler(conn net.Conn) error {
	_, err := conn.Write([]byte{consts.Version, consts.AuthTypeNoRequired})
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

// UnamePwdHandler 回复客户端，连接需要通过 用户名/密码 方式进行验证
func UnamePwdHandler(conn net.Conn) error {
	_, err := conn.Write([]byte{consts.Version, consts.AuthTypeUnamePwd})
	if err != nil {
		log.Println("write support method to client error: ", err)
		return err
	}

	uname, pwd, err := getUnamePwd(conn)
	if err != nil {
		//log.Println("get username and password error: ", err)
		return err
	}

	// +----+--------+
	// |VER | STATUS |
	// +----+--------+
	// | 1  |   1    |
	// +----+--------+
	// 服务端将验证结果发送给客户，如果验证成功则返回状态 0x00,否则返回任何非 0x00 的值。
	// 客户端收到未成功验的状态必须关闭当前连接。
	ok := authUser(uname, pwd)
	if ok {
		_, err = conn.Write([]byte{consts.Version, consts.AuthUserOk})
		if err != nil {
			log.Println("write auth result response error: ", err)
			return err
		}
		return nil
	}

	_, err = conn.Write([]byte{consts.Version, consts.AuthUserFail})
	if err != nil {
		log.Println("write auth result response error: ", err)
		return err
	}
	return nil
}

// 如果客户选择了 用户名/密码 协议，那么客户将会发送如下报文：
// 详见 RFC 1929, https://www.ietf.org/rfc/rfc1929.txt
//
// +----+------+----------+------+----------+
// |VER | ULEN |  UNAME   | PLEN |  PASSWD  |
// +----+------+----------+------+----------+
// | 1  |  1   | 1 to 255 |  1   | 1 to 255 |
// +----+------+----------+------+----------+
//
// VER 		协议版本号
// ULEN 	用户名长度
// UNAME 	用户名
// PLEN 	密码长度
// PASSWD 	密码
//
// getUnamePwd 从 conn 中读取客户发送的请求报文，并得到 username 和 password
func getUnamePwd(conn net.Conn) (uname, pwd string, err error) {
	buf := make([]byte, 255)

	// 读取 VER 和 ULEN
	_, err = io.ReadFull(conn, buf[:2])
	if err != nil {
		return "", "", err
	}

	ver := buf[0]
	ulen := buf[1]

	if ver != consts.Version {
		errMsg := fmt.Errorf("invalid version")
		log.Println(errMsg)
		return "", "", errMsg
	}

	// 读取 USERNAME
	_, err = io.ReadFull(conn, buf[:ulen])
	if err != nil {
		errMsg := fmt.Errorf("read USERNAME error: %+v\n", err)
		log.Println(errMsg)
		return "", "", errMsg
	}

	uname = string(buf[:ulen])

	// 读取 PLEN
	_, err = io.ReadFull(conn, buf[:1])
	if err != nil {
		log.Printf("read PLEN error: %+v\n", err)
		return "", "", fmt.Errorf("read PLEN error: %+v ", err)
	}
	plen := int(buf[0])

	// 读取 PASSWORD
	_, err = io.ReadFull(conn, buf[:plen])
	if err != nil {
		log.Printf("read PLEN error: %+v\n", err)
		return "", "", fmt.Errorf("read PLEN error: %+v ", err)
	}
	pwd = string(buf[:plen])

	return uname, pwd, nil
}

// authUser 对用户名和密码进行验证，TODO 使用者自行实现
func authUser(uname, pwd string) bool {
	b := uname == Username && pwd == Password
	if b {
		log.Printf("user %v auth success", uname)
	} else {
		log.Printf("user %v auth fail", uname)
	}
	return b
}
