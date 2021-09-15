package config

/**
* There will be additional configurations
 */

type Config struct {
	ServerAddr      string
	OpenApiSpecPath string
}

func createConfig() *Config {
	c := Config{ServerAddr: ":8000", OpenApiSpecPath: "./spec/openapi.json"}
	return &c
}

// Get returns a quickstarts service configuration
func Get() *Config {
	return createConfig()
}
