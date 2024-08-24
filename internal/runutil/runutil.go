package runutil

import (
	"os"
	"strings"
)

func GetenvDefault(varName, defaultValue string) string {
	value, isSet := os.LookupEnv(varName)
	if !isSet {
		return defaultValue
	}
	return value
}

func MustGetenv(varName string) string {
	value, isSet := os.LookupEnv(varName)
	if !isSet {
		panic("the environment variable " + varName + " is not set")
	}
	return value
}

func GetenvBool(varName string) bool {
	return strings.ToLower(os.Getenv(varName)) == "true"
}
