package config

/**
* There will be additional configurations
 */

type Config struct {
	ServerAddr string
}

func createConfig() *Config {
	c := Config{ServerAddr: ":8888"}
	return &c
}

// Get returns a quickstarts service configuration
func Get() *Config {
	return createConfig()
}
