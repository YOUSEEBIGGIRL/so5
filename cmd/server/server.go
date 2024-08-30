package server

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"log"
	"zz.io/cargo/so5/server"
)

var svrOpts = &ServerOptions{}

type ServerOptions struct {
	ListenAddr string
}

func (c *ServerOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.ListenAddr, "listen-addr", "", "")
}

var ServerCmd = &cobra.Command{
	Use: "server",
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Printf("%+v", svrOpts)
		if svrOpts.ListenAddr == "" {
			return fmt.Errorf("usage: so5 server --listen-addr=<> ")
		}
		return server.ListenAndServer(svrOpts.ListenAddr)
	},
}

func InitCmd() {
	svrFs := pflag.NewFlagSet("server", pflag.ExitOnError)
	svrOpts.AddFlags(svrFs)
	ServerCmd.Flags().AddFlagSet(svrFs)
}
