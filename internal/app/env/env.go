package env

import "os"

type Environment struct {
	name string
}

var (
	Dev = Environment{name: "dev"}
	Pro = Environment{name: "pro"}
)

func Detect() Environment {
	return detect(os.Getenv("ENVIRONMENT"))
}

func detect(value string) Environment {
	switch value {
	case Dev.name:
		return Dev
	case Pro.name:
		return Pro
	default:
		return Pro
	}
}
