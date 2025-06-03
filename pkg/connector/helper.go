package connector

import (
	"strconv"
)

func parsePageToken(token string) (uint, error) {
	if token == "" {
		return 0, nil
	}
	num, err := strconv.ParseUint(token, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint(num), nil
}

func createPageToken(pageNumber *uint) string {
	if pageNumber == nil {
		return ""
	}
	return strconv.FormatUint(uint64(*pageNumber), 10)
}
