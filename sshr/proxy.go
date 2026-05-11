package sshr

import (
	"net"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

func newSSHProxyConn(conn net.Conn, proxyConf *ssh.ProxyConfig) (proxyConn *ssh.ProxyConn, err error) {
	d, err := ssh.NewDownstreamConn(conn, proxyConf.ServerConfig)
	if err != nil {
		logrus.Warnf("[proxy] downstream handshake failed: %v", err)
		return nil, err
	}

	authRequestMsg, err := d.GetAuthRequestMsg()
	if err != nil {
		logrus.Warnf("[proxy] GetAuthRequestMsg failed: %v", err)
		return nil, err
	}

	username := authRequestMsg.User
	logrus.Infof("[proxy] user=%s method=%s", username, authRequestMsg.Method)
	p := &ssh.ProxyConn{
		User:       username,
		Downstream: d,
	}
	defer func() {
		if proxyConn == nil {
			d.Close()
		}
	}()

	upstreamHost, err := proxyConf.FindUpstreamHook(username)
	if err != nil {
		if err := p.SendFailureMsg(err.Error()); err != nil {
			return p, err
		}
		return p, err
	}
	p.DestinationHost = upstreamHost

	upstreamAddr := upstreamHost + ":" + proxyConf.DestinationPort
	upConn, err := net.Dial("tcp", upstreamAddr)
	if err != nil {
		logrus.Errorf("[proxy] user=%s TCP dial to upstream %s failed: %v", username, upstreamAddr, err)
		return p, err
	}
	logrus.Infof("[proxy] user=%s TCP connected to upstream %s", username, upstreamAddr)

	u, err := ssh.NewUpstreamConn(upConn, &ssh.ClientConfig{
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
	if err != nil {
		logrus.Errorf("[proxy] user=%s upstream SSH handshake failed: %v", username, err)
		return p, err
	}
	logrus.Infof("[proxy] user=%s upstream SSH handshake OK", username)
	defer func() {
		if proxyConn == nil {
			u.Close()
		}
	}()

	p.Upstream = u

	// Wrap the hook per-connection to detect whether pubkey auth was ever attempted.
	pubkeyAttempted := false
	connConf := *proxyConf
	origHook := proxyConf.FetchAuthorizedKeysHook
	connConf.FetchAuthorizedKeysHook = func(u string, h string) ([]byte, error) {
		pubkeyAttempted = true
		if origHook != nil {
			return origHook(u, h)
		}
		return nil, nil
	}

	logrus.Infof("[proxy] user=%s starting auth (initial method=%s)", username, authRequestMsg.Method)
	if err = p.AuthenticateProxyConn(authRequestMsg, &connConf); err != nil {
		if !pubkeyAttempted {
			logrus.Warnf("[proxy] user=%s auth failed: %v — client never offered a pubkey (no key in agent/identity, or password-only client)", username, err)
		} else {
			logrus.Errorf("[proxy] user=%s AuthenticateProxyConn failed: %v", username, err)
		}
		return p, err
	}

	return p, nil
}
