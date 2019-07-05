package utils

import (
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
)

type Configuration struct {
	RendezvousString string `yaml:"RendezvousString" env:"RENDEZVOUS_STRING"` // Unique string to identify group of nodes. Share this with your friends to let them connect with you
	ProtocolID       string `yaml:"ProtocolID" env:"PROTOCOL_ID"` // Sets a protocol id for stream headers
	ListenHost       string `yaml:"ListenHost" env:"LISTEN_HOST"` // The bootstrap node host listen address
	ListenPort       int    `yaml:"ListenPort" env:"LISTEN_PORT"` // Node listen port
}

// Config contains the default values
var Config = Configuration{
	RendezvousString: "moonshard",
	ProtocolID: "/chat/1.1.0",
	ListenHost: "0.0.0.0",
	ListenPort: 4001,
}

func GetConfig() Configuration {
	return Config
}

func SetConfig(newConfig *Configuration) {

	if newConfig.RendezvousString != "" {
		Config.RendezvousString = newConfig.RendezvousString
	}

	if newConfig.ProtocolID != "" {
		Config.ProtocolID = newConfig.ProtocolID
	}

	if newConfig.ListenHost != "" {
		Config.ListenHost = newConfig.ListenHost
	}

	if newConfig.ListenPort != 0 && (newConfig.ListenPort < 0 && newConfig.ListenPort > 65535) {
		Config.ListenPort = newConfig.ListenPort
	}
}

// ReadFromFile loads the Configuration from yaml file if needed
func ReadFromFile() error {

	file, err := ioutil.ReadFile("config/config.yaml")
	if err == nil {
		if err := yaml.Unmarshal(file, &Config); err != nil {
			return errors.Wrap(err, "could not unmarshal yaml file")
		}
	} else if !os.IsNotExist(err) {
		return errors.Wrap(err, "could not read config file")
	} else {
		log.Println("No configuration file found, using defaults with environment variable overrides.")
	}

	return nil
}