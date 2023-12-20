package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/yaml"
)

var (
	infoLog *logrus.Logger
	errLog  *logrus.Logger
)

func init() {
	infoLog = logrus.New()
	infoLog.SetOutput(os.Stdout)
	errLog = logrus.New()
	errLog.SetOutput(os.Stderr)
}

type host struct {
	Port int `json:"port"`
}

func main() {
	dataBytes, err := os.ReadFile("/var/run/onprem-firewall.yaml")
	if err != nil {
		errLog.Fatal(err)
	}

	var hostMap map[string]host
	if err := yaml.Unmarshal(dataBytes, &hostMap); err != nil {
		errLog.Fatal(err)
	}

	for host, hostConfig := range hostMap {
		checkUp(host, hostConfig.Port)
	}
}

func checkUp(host string, port int) {
	// bruk up tjenesten for å teste wildcard hosts
	if strings.Contains(host, "*.") {
		host = strings.Replace(host, "*", "up", 1)
	}

	if err := dialWithRetry(fmt.Sprintf("%v:%v", host, port)); err != nil {
		errLog.Errorf("Host %v unreachable on port %v: err %v", host, port, err)
		return
	}

	infoLog.Infof("Host %v ok on port %v", host, port)
}

func dialWithRetry(host string) error {
	numRetries := 3
	timeout := 5 * time.Second
	retryDelay := 1 * time.Second

	var conn net.Conn
	var err error
	for i := 0; i < numRetries; i++ {
		conn, err = net.DialTimeout("tcp", host, timeout)
		if err != nil {
			time.Sleep(retryDelay)
			infoLog.Infof("Retrying host %v", host)
			continue
		}
		conn.Close()
		return nil
	}

	return err
}
