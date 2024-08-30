package main

import (
	"github.com/spf13/cobra"

	"zz.io/cargo/so5/cmd/client"
	"zz.io/cargo/so5/cmd/server"
)

var rootCmd = &cobra.Command{
	Use: "so5",
	Run: func(cmd *cobra.Command, args []string) {},
}

// ./so5 server --listen-addr=127.0.0.1:8081
// ./so5 client --listen-addr=127.0.0.1:8080 --proxy-addr=127.0.0.1:8081 --target-addr=127.0.0.1:8083
func main() {
	client.InitCmd()
	server.InitCmd()

	rootCmd.AddCommand(client.ClientCmd, server.ServerCmd)
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
