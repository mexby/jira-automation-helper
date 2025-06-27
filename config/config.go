package config

import (
	"github.com/jinzhu/configor"
)

type Configuration struct {
	APIKey string `yaml:"APIKey"`
}

func configFiles() []string {
	return []string{"config.yml"}
}

// Get returns the configuration extracted from env variables or config file.
func Get() *Configuration {
	conf := new(Configuration)
	err := configor.New(&configor.Config{ENVPrefix: "MEXBY_JIRA", Silent: true}).Load(conf, configFiles()...)
	if err != nil {
		panic(err)
	}
	return conf
}
