package sshr

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

type config struct {
	ListenAddr      string   `toml:"listen_addr"`
	RemoteAddr      string   `toml:"remote_addr"`
	DestinationPort string   `toml:"destination_port"`
	HostKeyPath     []string `toml:"server_hostkey_path"`
	UseMasterKey    bool     `toml:"use_master_key"`
	MasterKeyPath   string   `toml:"master_key_path"`
}

func loadConfig(path string) (*config, error) {
	var c config
	defaultConfig(&c)

	_, err := toml.DecodeFile(path, &c)
	if err != nil {
		return nil, err
	}

	if err := validateAndLogConfig(&c); err != nil {
		return nil, err
	}

	return &c, nil
}

func validateAndLogConfig(c *config) error {
	logrus.Infof("Config: listen_addr=%s", c.ListenAddr)
	logrus.Infof("Config: destination_port=%s", c.DestinationPort)
	logrus.Infof("Config: server_hostkey_path=%v", c.HostKeyPath)
	logrus.Infof("Config: use_master_key=%v", c.UseMasterKey)

	for _, kp := range c.HostKeyPath {
		if _, err := os.Stat(kp); err != nil {
			return fmt.Errorf("server_hostkey_path %q not found: %v", kp, err)
		}
	}

	if c.UseMasterKey {
		logrus.Infof("Config: master_key_path=%q", c.MasterKeyPath)
		if c.MasterKeyPath == "" {
			return fmt.Errorf("use_master_key is true but master_key_path is empty — check toml key spelling (must be master_key_path)")
		}
		if _, err := os.Stat(c.MasterKeyPath); err != nil {
			return fmt.Errorf("master_key_path %q not found: %v", c.MasterKeyPath, err)
		}
	}

	return nil
}

func newServerConfig(c *config) (*ssh.ServerConfig, error) {
	serverConfig := &ssh.ServerConfig{}

	for _, k := range c.HostKeyPath {
		privateKeyBytes, err := os.ReadFile(k)
		if err != nil {
			return nil, err
		}
		privateKey, err := ssh.ParsePrivateKey(privateKeyBytes)
		if err != nil {
			return nil, err
		}
		serverConfig.AddHostKey(privateKey)
	}

	return serverConfig, nil
}

func defaultConfig(config *config) {
	config.ListenAddr = "0.0.0.0:2222"
	config.UseMasterKey = false
}
