package config

import (
	"flag"
	"io/ioutil"
	"os"

	"github.com/BurntSushi/toml"
)

// FIXME: 32 bit compabilty is a problem for configuration items

const DefaultSystemConfigPath = "data/gauss.conf"

type Config struct {
	SystemPath  string
	ShowHelp    bool
	ShowVersion bool
	Server      ServerInfo
}

type ServerInfo struct {
	ListenIp    string `toml:"listenIp"`
	BroadcastIp string `toml:"broadcastIp"`
	Port        int    `toml:"port"`
	JoinIp      string `toml:"joinIp"`
	JoinPort    int    `toml:"joinPort"`
	LogDir      string `toml:"logDir"`
}

func (c *Config) Load(arguments []string) error {
	var path string
	f := flag.NewFlagSet("gauss", -1)
	f.SetOutput(ioutil.Discard)
	f.StringVar(&path, "config", "", "path to config file")
	f.Parse(arguments)

	// Load from system file.
	if err := c.LoadSystemFile(); err != nil {
		return err
	}

	// Load from config file specified in arguments.
	if path != "" {
		if err := c.LoadFile(path); err != nil {
			return err
		}
	}

	// Load from command line flags.
	if err := c.LoadFlags(arguments); err != nil {
		return err
	}

	return nil
}

// Loads from the system newton configuration file if it exists.
func (c *Config) LoadSystemFile() error {
	if _, err := os.Stat(c.SystemPath); os.IsNotExist(err) {
		return nil
	}
	return c.LoadFile(c.SystemPath)
}

// Loads configuration from command line flags.
func (c *Config) LoadFlags(arguments []string) error {
	f := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	f.SetOutput(ioutil.Discard)

	/* Generic configuration  parameters */
	f.BoolVar(&c.ShowHelp, "h", false, "")
	f.BoolVar(&c.ShowHelp, "help", false, "")
	f.BoolVar(&c.ShowVersion, "version", false, "")

	/* Server server parameters */
	f.StringVar(&c.Server.ListenIp, "database-listen-ip", c.Server.ListenIp, "")
	f.StringVar(&c.Server.BroadcastIp, "database-broadcast-ip", c.Server.BroadcastIp, "")
	f.IntVar(&c.Server.Port, "database-port", c.Server.Port, "")
	f.StringVar(&c.Server.JoinIp, "database-join-ip", c.Server.JoinIp, "")
	f.IntVar(&c.Server.JoinPort, "database-join-port", c.Server.JoinPort, "")
	f.StringVar(&c.Server.LogDir, "database-log-dir", c.Server.LogDir, "")

	if err := f.Parse(arguments); err != nil {
		return err
	}

	return nil
}

// Loads configuration from a file.
func (c *Config) LoadFile(path string) error {
	_, err := toml.DecodeFile(path, &c)
	return err
}

// Creates a new configuration
func New() *Config {
	c := new(Config)
	c.SystemPath = DefaultSystemConfigPath
	return c
}
