package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
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
	Host string `json:"host"`
	Port int    `json:"port"`
}

type oracleHost struct {
	Host string `json:"host"`
	Port int    `json:"port"`
	Scan []host `json:"scan"`
}

func main() {
	dataBytes, err := os.ReadFile("/var/run/onprem-firewall.yaml")
	if err != nil {
		errLog.Fatal(err)
	}

	var hostMap map[string][]any
	if err := yaml.Unmarshal(dataBytes, &hostMap); err != nil {
		errLog.Fatal(err)
	}

	for hostType, hosts := range hostMap {
		// fmt.Println(hostType)
		switch hostType {
		case "oracle":
			checkUpForOracleHosts(hosts)
		default:
			checkUpForHosts(hosts)
		}
	}
}

func checkUpForOracleHosts(hosts []any) {
	for _, h := range hosts {
		current := oracleHost{}
		mapstructure.Decode(h, &current)
		checkUp(current.Host, current.Port)

		if len(current.Scan) > 0 {
			for _, sh := range current.Scan {
				checkUp(sh.Host, sh.Port)
			}
		}
	}
}

func checkUpForHosts(hosts []any) {
	for _, h := range hosts {
		current := host{}
		mapstructure.Decode(h, &current)
		checkUp(current.Host, current.Port)
	}
}

func checkUp(host string, port int) {
	// bruk up tjenesten for å teste wildcard hosts
	if strings.Contains(host, "*.") {
		host = strings.Replace(host, "*", "up", 1)
	}

	if err := dialWithRetry(fmt.Sprintf("%v:%v", host, port)); err != nil {
		errLog.Errorf("Host %v unreachable on port %v: err %v", host, port, err)
	}
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
