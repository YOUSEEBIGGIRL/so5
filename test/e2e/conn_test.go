package e2e

import (
	"io"
	"log"
	"net"
	"testing"

	"zz.io/cargo/so5/client"
	"zz.io/cargo/so5/server"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func TestServer(t *testing.T) {
	if err := server.ListenAndServer("127.0.0.1:8080"); err != nil {
		t.Fatal(err)
	}
}

func TestClient(t *testing.T) {
	if err := client.ListenAndServer("127.0.0.1:8081", "127.0.0.1:8080", "127.0.0.1:8083"); err != nil {
		t.Fatal(err)
	}
}

func TestTargetServer(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:8083")
	if err != nil {
		t.Fatal(err)
	}
	defer lis.Close()

	for {
		conn, err := lis.Accept()
		if err != nil {
			t.Error(err)
			continue
		}

		go func() {
			_, err := io.Copy(conn, conn)
			if err != nil {
				t.Error(err)
				return
			}
		}()
	}
}
