package e2e

import (
	"net"
	"testing"

	"zz.io/cargo/so5/client"
	"zz.io/cargo/so5/consts"
	"zz.io/cargo/so5/server"
)

var (
	serverSupport   = []byte{consts.AuthTypeNoRequired, consts.AuthTypeUnamePwd}
	serverNoSupport = []byte{0x10}
)

func TestAuthClient1(t *testing.T) {
	runClient(t, serverNoSupport)
}

func TestAuthClient2(t *testing.T) {
	runClient(t, serverSupport)
}

func TestAuthServerNoRequired(t *testing.T) {
	runServer(t, consts.AuthTypeNoRequired)
}

func TestAuthServerUnamePwd(t *testing.T) {
	runServer(t, consts.AuthTypeUnamePwd)
}

func runClient(t *testing.T, supportAuthMethods []byte) {
	conn, err := net.Dial("tcp", "127.0.0.1:8080")
	if err != nil {
		t.Fatal(err)
	}

	authMethod, err := client.NegotiationAuth(conn, supportAuthMethods)
	if err != nil {
		t.Fatal(err)
	}

	switch authMethod {
	case consts.AuthTypeNoRequired:
		t.Log("NegotiationAuthMethod: NoRequired")
	case consts.AuthTypeUnamePwd:
		if err := client.AuthUseUnamePwd(conn, server.Username, server.Password); err != nil {
			t.Error(err)
			return
		}
	case consts.AuthTypeNoAcceptable:
		t.Log("NegotiationAuthMethod: NoAcceptable")
	}
}

func runServer(t *testing.T, authMethod byte) {
	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		t.Fatal(err)
	}
	defer lis.Close()

	conn, err := lis.Accept()
	if err != nil {
		t.Error(err)
		return
	}
	defer conn.Close()

	if err := server.NegotiationAuth(conn, authMethod); err != nil {
		t.Error(err)
		return
	}
}
