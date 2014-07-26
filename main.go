package main

import (
	"fmt"

	"github.com/purak/gauss/config"
	"github.com/purak/gauss/dhash"
	//"github.com/purak/newton/cstream"
	"os"
	"strings"
)

var version = "0.0.1"
var usage = `
gauss -- Fast. scalable, in-memory data structure server

Usage:
  gauss -addr <addr>
  gauss -h | -help
  gauss -version

Options:
  -h -help          Show this screen.
  --version         Show version.

Client Communication Options:
  -addr=<host:port>         The public host:port used for client communication.
`

var c *config.Config

func Usage() string {
	return strings.TrimSpace(usage)
}

func main() {
	c = config.New()
	if err := c.Load(os.Args[1:]); err != nil {
		fmt.Println(Usage() + "\n")
		fmt.Println(err.Error() + "\n")
		os.Exit(1)
	} else if c.ShowVersion {
		fmt.Println("gauss version")
		os.Exit(0)
	} else if c.ShowHelp {
		fmt.Println(Usage() + "\n")
		os.Exit(0)
	}
	startServer()
}

// Start a Gauss database node
func startServer() {
	// TODO: Verbose option
	listenAddr := fmt.Sprintf("%s:%d", c.Server.ListenIp, c.Server.Port)
	broadcastAddr := fmt.Sprintf("%s:%d", c.Server.BroadcastIp, c.Server.Port)
	s := dhash.NewNodeDir(listenAddr, broadcastAddr, c.Server.LogDir)
	// Start the database server
	s.MustStart()

	// Join a database cluster.
	if c.Server.JoinIp != "" {
		joinAddr := fmt.Sprintf("%s:%d", c.Server.JoinIp, c.Server.JoinPort)
		s.MustJoin(joinAddr)
	}

	select {}
}
