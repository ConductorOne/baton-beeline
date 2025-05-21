package connector

import (
	"slices"
	"strconv"
	"testing"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
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

func AssertNoRatelimitAnnotations(
	t *testing.T,
	actualAnnotations annotations.Annotations,
) {
	if actualAnnotations != nil && len(actualAnnotations) == 0 {
		return
	}

	for _, annotation := range actualAnnotations {
		var ratelimitDescription v2.RateLimitDescription
		err := annotation.UnmarshalTo(&ratelimitDescription)
		if err != nil {
			continue
		}
		if slices.Contains(
			[]v2.RateLimitDescription_Status{
				v2.RateLimitDescription_STATUS_ERROR,
				v2.RateLimitDescription_STATUS_OVERLIMIT,
			},
			ratelimitDescription.Status,
		) {
			t.Fatal("request was ratelimited, expected not to be ratelimited")
		}
	}
}
