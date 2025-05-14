package connector

import "strconv"

func parsePageToken(token string) (uint, error) {
	if token != "" {
		num, err := strconv.ParseUint(token, 10, 32)
		if err != nil {
			return uint(num), err
		}
	}
	return 0, nil
}

func createPageToken(pageNumber *uint) string {
	if pageNumber == nil {
		return ""
	}
	return strconv.FormatUint(uint64(*pageNumber), 10)
}
