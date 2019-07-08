package utils

const maxPort = 65535

type Configuration struct {
	RendezvousString string // Unique string to identify group of nodes. Share this with your friends to let them connect with you
	ProtocolID       string // Sets a protocol id for stream headers
	ListenHost       string // The bootstrap node host listen address
	ListenPort       int    // Node listen port
}

// Config contains the default values
var config = Configuration{
	RendezvousString: "moonshard",
	ProtocolID: "/chat/1.1.0",
	ListenHost: "0.0.0.0",
	ListenPort: 4001,
}

func GetConfig() *Configuration {
	return &config
}

func SetConfig(newConfig *Configuration) {

	if newConfig.RendezvousString != "" {
		config.RendezvousString = newConfig.RendezvousString
	}

	if newConfig.ProtocolID != "" {
		config.ProtocolID = newConfig.ProtocolID
	}

	if newConfig.ListenHost != "" {
		config.ListenHost = newConfig.ListenHost
	}

	if newConfig.ListenPort != 0 && (newConfig.ListenPort < 0 && newConfig.ListenPort > maxPort) {
		config.ListenPort = newConfig.ListenPort
	}
}