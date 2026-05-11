package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"github.com/Gurpartap/logrus-stack"
	"github.com/sirupsen/logrus"
	"github.com/tsurubee/sshr/sshr"
	"net"
	"strings"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	stackLevels := []logrus.Level{logrus.PanicLevel, logrus.FatalLevel}
	logrus.AddHook(logrus_stack.NewHook(stackLevels, stackLevels))
}

var separator string
var suffix string

func main() {

	flagConfigFile := flag.String("config", "example.toml", "path to config file")
	flagSeparator := flag.String("separator", "_", "separator for host spec in username")
	flagSuffix := flag.String("suffix", ".blue.lan", "valid suffix for hosts")

	flag.Parse()
	confFile := *flagConfigFile
	separator = *flagSeparator
	suffix = *flagSuffix

	sshServer, err := sshr.NewSSHServer(confFile)
	if err != nil {
		logrus.Fatal(err)
	}

	sshServer.ProxyConfig.FindUpstreamHook = FindUpstreamByUsername
	sshServer.ProxyConfig.FetchAuthorizedKeysHook = FetchAuthorizedKeysWithLogging
	if err := sshServer.Run(); err != nil {
		logrus.Fatal(err)
	}
}

func FindUpstreamByUsername(username string) (string, error) {
	parts := strings.Split(username, separator)
	if len(parts) == 2 {
		host := parts[1]
		fqdn := host + suffix
		_, err := net.LookupHost(fqdn)
		if err == nil && host != "localhost" {
			logrus.Infof("[upstream] user=%s resolved host=%s (fqdn=%s)", username, host, fqdn)
			return host, nil
		}
		logrus.Warnf("[upstream] user=%s host lookup failed for %s: %v", username, fqdn, err)
	} else {
		logrus.Warnf("[upstream] user=%s bad username format (separator=%q, parts=%d)", username, separator, len(parts))
	}
	return "", errors.New("access denied")
}

func FetchAuthorizedKeysWithLogging(username string, host string) ([]byte, error) {
	path := fmt.Sprintf("/home/%s/.ssh/authorized_keys", username)
	data, err := os.ReadFile(path)
	if err != nil {
		logrus.Errorf("[auth] authorized_keys read FAILED user=%s host=%s path=%s err=%v", username, host, path, err)
		return nil, err
	}
	// count non-empty, non-comment lines as a proxy for key count
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	keyCount := 0
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" && !strings.HasPrefix(l, "#") {
			keyCount++
		}
	}
	logrus.Infof("[auth] authorized_keys OK user=%s host=%s path=%s size=%d bytes keys~%d", username, host, path, len(data), keyCount)
	return data, nil
}
