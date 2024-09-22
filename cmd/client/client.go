package client

import (
	"fmt"
	"log"
	"zz.io/cargo/so5/client"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var cliOpts = &ClientOptions{}

type ClientOptions struct {
	listenAddr string
	proxyAddr  string
	targetAddr string
}

func (c *ClientOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.listenAddr, "listen-addr", "", "listen address")
	fs.StringVar(&c.proxyAddr, "proxy-addr", "", "proxy address")
	fs.StringVar(&c.targetAddr, "target-addr", "", "target server addr")
}

var ClientCmd = &cobra.Command{
	Use: "client",
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Printf("%+v", cliOpts)
		if cliOpts.listenAddr == "" || cliOpts.proxyAddr == "" || cliOpts.targetAddr == "" {
			return fmt.Errorf("usage: so5 client --listen-addr=<> --proxy-addr=<> --target-addr=<>")
		}
		return client.ListenAndServer(cliOpts.listenAddr, cliOpts.proxyAddr, cliOpts.targetAddr)
	},
}

func InitCmd() {
	clientFs := pflag.NewFlagSet("client", pflag.ExitOnError)
	cliOpts.AddFlags(clientFs)
	ClientCmd.Flags().AddFlagSet(clientFs)
}
