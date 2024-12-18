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

func DeleteItem[S comparable](s []S, i S) []S {
	for idx, item := range s {
		if item == i {
			return append(s[:idx], s[idx+1:]...)
		}
	}
	return s
}

//
