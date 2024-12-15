package common

import (
	"strconv"
	"strings"
)

func SplitBindAddr(addr string) (string, int) {
	parts := strings.Split(addr, ":")
	if len(parts) == 1 {
		return parts[0], 0
	}
	port, _ := strconv.Atoi(parts[1])
	return parts[0], port
}
