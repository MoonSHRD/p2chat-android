package utils

type Configuration struct {
	RendezvousString string // Unique string to identify group of nodes. Share this with your friends to let them connect with you
	ProtocolID       string // Sets a protocol id for stream headers
	ListenHost       string // The bootstrap node host listen address
	ListenPort       int    // Node listen port
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