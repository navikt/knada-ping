package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
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

func main() {
	ctx := context.Background()
	url := "https://raw.githubusercontent.com/navikt/nada-internal/main/brannmur.md"
	token := os.Getenv("GITHUB_READ_TOKEN")

	file, err := getFile(ctx, url, token)
	if err != nil {
		errLog.Fatal(err)
	}

	hostMap, err := parseFile(file)
	if err != nil {
		errLog.Fatal(err)
	}

	for port, hosts := range hostMap {
		if err := checkUp(port, hosts); err != nil {
			errLog.Fatal(err)
		}
	}
}

func getFile(ctx context.Context, url, token string) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("getting knada hosts file: %w", err)
	}
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %v", token))
	client := http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("performing http request, URL: %v: %w", url, err)
	}

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	return string(bodyBytes), nil
}

func parseFile(file string) (map[string][]string, error) {
	hostMap := map[string][]string{}

	var port string
	for _, l := range strings.Split(file, "\n") {
		if strings.HasPrefix(l, "##") {
			port = readPort(l)
			continue
		}
		if port != "" {
			h, err := readHost(l)
			if err != nil {
				continue
			}
			hostMap[port] = append(hostMap[port], h)
		}
	}

	return hostMap, nil
}

func readPort(l string) string {
	parts := strings.Split(l, " ")
	for i, val := range parts {
		if val == "port" {
			return parts[i+1]
		}
	}

	return ""
}

func readHost(l string) (string, error) {
	// ignorer det som kun gjelder for managed notebooks
	if strings.Contains(l, "(kun knada managed notebooks)") {
		return "", errors.New("ignore")
	}

	// bruk up tjenesten for å teste wildcard hosts
	if strings.Contains(l, "*.") {
		l = strings.Replace(l, "*", "up", 1)
	}

	parts := strings.Split(l, " ")
	for _, p := range parts {
		if p != "" {
			return p, nil
		}
	}

	return "", errors.New("host not found")
}

func checkUp(port string, hosts []string) error {
	for _, h := range hosts {
		if err := dialWithRetry(fmt.Sprintf("%v:%v", h, port)); err != nil {
			errLog.Errorf("Host %v unreachable on port %v: err %v", h, port, err)
			continue
		}
		infoLog.Infof("Host %v ok on port %v", h, port)
	}

	return nil
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
