package helper

import "strings"

func SplitEndpoint(endpoint string) (network string, address string) {
	if strings.HasPrefix(endpoint, "unix:") {
		ss := strings.SplitN(endpoint, ":", 2)
		network = ss[0]
		address = ss[1]
	} else {
		network = "tcp"
		address = endpoint
	}
	return
}
