package utils

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"time"
)

func IsFoundHost(host string, port uint16) bool {
	target := fmt.Sprintf("%s:%d", host, port)

	_, err := net.DialTimeout("tcp", target, 1*time.Second)

	if err != nil {
		fmt.Printf("%s %s \n", target, err)
		return false
	}

	return true
}

var PATTERN = regexp.MustCompile(`((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?\.){3})(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)`)

func FindNeighbors(myHost string, myPort uint16, startIp uint8, endIp uint8, startPort uint16, endPort uint16) []string {
	address := fmt.Sprintf("%s:%d", myHost, myPort)

	m := PATTERN.FindStringSubmatch(myHost)
	if m == nil {
		return nil
	}

	prefixHost := m[1]
	lastIp, _ := strconv.Atoi(m[len(m)-1])

	neighbors := make([]string, 0)

	for port := startPort; port <= endPort; port += 1 {
		for ip := startIp; ip <= endIp; ip += 1 {
			guessHost := fmt.Sprintf("%s%d", prefixHost, lastIp+int(ip))
			guessTarget := fmt.Sprintf("%s:%d", guessHost, port)

			if guessTarget != address && IsFoundHost(guessHost, port) {
				neighbors = append(neighbors, guessTarget)
			}
		}
	}

	return neighbors
}

func GetHost() string {

	return "127.0.0.1"

	/*
		TODO: need to repair the hosts based on valid hosts
			  today, this implementation is not working correctly,
			  addresses are changing in the array, and its not resolving correctly

			hostname, err := os.Hostname()

			if err != nil {
				fmt.Println(err)
				return "127.0.0.1"
			}

			address, err := net.LookupHost(hostname)

			if err != nil {
				fmt.Println(err)
				return "127.0.0.1"
			}

			fmt.Println(address[1])

			return address[1]

	*/
}
