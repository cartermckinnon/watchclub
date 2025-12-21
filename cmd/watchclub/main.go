package main

import (
	"github.com/cartermckinnon/watchclub/cmd/watchclub/server"
	"github.com/cartermckinnon/watchclub/internal/cli"
)

func main() {
	m := cli.Main{
		Name:           "watchclub",
		Description:    "Watch things together with your friends",
		AdditionalHelp: "http://github.com/cartermckinnon/watchclub",
		Commands: []cli.Command{
			server.NewServerCommand(),
		},
	}
	m.Run()
}
