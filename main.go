package main

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

func main() {
	ctx := context.Background()
	url := "https://raw.githubusercontent.com/navikt/pig/master/nada/doc/knada-gcp.md"
	token := os.Getenv("GITHUB_READ_TOKEN")

	file, err := getFile(ctx, url, token)
	if err != nil {
		log.Fatal(err)
	}

	hostMap, err := parseFile(file)
	if err != nil {
		log.Fatal(err)
	}

	for port, hosts := range hostMap {
		if err := checkUp(port, hosts); err != nil {
			log.Fatal(err)
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

	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	return string(bodyBytes), nil
}

func parseFile(file string) (map[string][]string, error) {
	hostMap := map[string][]string{}

	var port string
	for _, l := range strings.Split(file, "\n") {
		if strings.HasPrefix(l, "###") {
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
	parts := strings.Split(l, " ")
	for _, p := range parts {
		if p != "" {
			return p, nil
		}
	}

	return "", errors.New("host not found")
}

func checkUp(port string, hosts []string) error {
	timeout := 5 * time.Second
	for _, h := range hosts {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%v:%v", h, port), timeout)
		if err != nil {
			log.Warnf("Host %v unreachable on port %v, error: ", h, port, err)
			continue
		}
		conn.Close()
	}

	return nil
}
