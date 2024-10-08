package utils

import (
	"os"
	"strings"
)

func GetEnvOrDefault(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}

func GetActionInputAsSlice(input string) []string {
	result := make([]string, 0, 5)
	s := strings.Split(input, "\n")
	for _, i := range s {
		if c := strings.TrimSpace(i); c != "" {
			result = append(result, c)
		}
	}
	return result
}
