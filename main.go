package main

import (
	"errors"
	"flag"
	"github.com/Gurpartap/logrus-stack"
	"github.com/sirupsen/logrus"
	"github.com/tsurubee/sshr/sshr"
	"strings"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	stackLevels := []logrus.Level{logrus.PanicLevel, logrus.FatalLevel}
	logrus.AddHook(logrus_stack.NewHook(stackLevels, stackLevels))
}

var separator string

func main() {

	flagConfigFile := flag.String("config", "example.toml", "path to config file")
	flagSeparator := flag.String("separator", "_", "separator for host spec in username")

	flag.Parse()
	confFile := *flagConfigFile
	separator = *flagSeparator

	sshServer, err := sshr.NewSSHServer(confFile)
	if err != nil {
		logrus.Fatal(err)
	}

	sshServer.ProxyConfig.FindUpstreamHook = FindUpstreamByUsername
	if err := sshServer.Run(); err != nil {
		logrus.Fatal(err)
	}
}

func FindUpstreamByUsername(username string) (string, error) {
	parts := strings.Split(username, separator)
	if len(parts) == 2 {
		host := parts[1]
		if host != "localhost" {
			return host, nil
		}
	}
	return "", errors.New("access denied")
}
