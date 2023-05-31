package environment

import (
	"os"
)

const (
	VARIABLE_NAME = "PP_ENV"

	DEV_ENVIRONMENT        = "dev"
	PRODUCTION_ENVIRONMENT = "production"
)

var environment string

func GetEnvironment() string {
	if environment == "" {
		SetEnvironment(os.Getenv(VARIABLE_NAME))
	}

	return environment
}

func SetEnvironment(newEnvironment string) {
	switch newEnvironment {
	case DEV_ENVIRONMENT:
		environment = DEV_ENVIRONMENT
	default:
		environment = PRODUCTION_ENVIRONMENT
	}
}

func IsDev() bool {
	return GetEnvironment() == DEV_ENVIRONMENT
}
