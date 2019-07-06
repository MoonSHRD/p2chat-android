package pkg

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"

	"github.com/MoonSHRD/p2chat-android/pkg/utils"
	"github.com/MoonSHRD/p2chat/api"
	p2chat "github.com/MoonSHRD/p2chat/pkg"
	mapset "github.com/deckarep/golang-set"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p-core/protocol"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/multiformats/go-multiaddr"
)

type Config struct {
	RendezvousString string // Unique string to identify group of nodes. Share this with your friends to let them connect with you
	ProtocolID       string // Sets a protocol id for stream headers
	ListenHost       string // The bootstrap node host listen address
	ListenPort       int    // Node listen port
}

var myself host.Host

var globalCtx context.Context
var globalCtxCancel context.CancelFunc

var Pb *pubsub.PubSub
var networkTopics = mapset.NewSet()
var messageQueue utils.Queue
var handler p2chat.Handler
var serviceTopic string

// this function get new messages from subscribed topic
func readSub(subscription *pubsub.Subscription, incomingMessagesChan chan pubsub.Message) {
	ctx := globalCtx
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		msg, err := subscription.Next(context.Background())
		if err != nil {
			fmt.Println("Error reading from buffer")
			panic(err)
		}

		if string(msg.Data) == "" {
			return
		}
		if string(msg.Data) != "\n" {
			addr, err := peer.IDFromBytes(msg.From)
			if err != nil {
				fmt.Println("Error occurred when reading message From field...")
				panic(err)
			}

			// This checks if sender address of incoming message is ours. It is need because we get our messages when subscribed to the same topic.
			if addr == myself.ID() {
				continue
			}
			incomingMessagesChan <- *msg
		}

	}
}

// Publish message into some topic
func PublishMessage(topic string, text string) {
	message := &api.BaseMessage{
		Body: text,
		Flag: api.FLAG_GENERIC_MESSAGE,
	}

	sendData, err := json.Marshal(message)
	if err != nil {
		fmt.Println("Error occurred when marshalling message object")
		return
	}

	handler.PbMutex.Lock()
	err = Pb.Publish(topic, sendData)
	handler.PbMutex.Unlock()
	if err != nil {
		fmt.Println("Error occurred when publishing")
		return
	}
}

func Start(rendezvous string, pid string, listenHost string, port int) {
	cfg := GetConfig(&rendezvous, &pid, &listenHost, &port)
	serviceTopic = cfg.RendezvousString

	fmt.Printf("[*] Listening on: %s with port: %d\n", cfg.ListenHost, cfg.ListenPort)

	ctx, ctxCancel := context.WithCancel(context.Background())
	globalCtx = ctx
	globalCtxCancel = ctxCancel
	r := rand.Reader

	// Creates a new RSA key pair for this host.
	prvKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		panic(err)
	}

	// 0.0.0.0 will listen on any interface device.
	sourceMultiAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", cfg.ListenHost, cfg.ListenPort))

	// libp2p.New constructs a new libp2p Host.
	// Other options can be added here.
	host, err := libp2p.New(
		ctx,
		libp2p.ListenAddrs(sourceMultiAddr),
		libp2p.Identity(prvKey),
	)

	if err != nil {
		panic(err)
	}

	// Set a function as stream handler.
	// This function is called when a peer initiates a connection and starts a stream with this peer. (Handle incoming connections)
	//	host.SetStreamHandler(protocol.ID(cfg.ProtocolID), handleStream)

	fmt.Printf("\n[*] Your Multiaddress Is: /ip4/%s/tcp/%v/p2p/%s\n", cfg.ListenHost, cfg.ListenPort, host.ID().Pretty())

	myself = host

	// Initialize pubsub object
	pb, err := pubsub.NewFloodsubWithProtocols(context.Background(), host, []protocol.ID{protocol.ID(cfg.ProtocolID)}, pubsub.WithMessageSigning(false))
	if err != nil {
		fmt.Println("Error occurred when create PubSub")
		panic(err)
	}

	Pb = pb

	handler = p2chat.NewHandler(pb, serviceTopic, &networkTopics)

	peerChan := p2chat.InitMDNS(ctx, host, serviceTopic)

	//Subscription should go BEFORE connections
	// NOTE:  here we use Randezvous string as 'topic' by default .. topic != service tag
	subscription, err := pb.Subscribe(serviceTopic)
	if err != nil {
		fmt.Println("Error occurred when subscribing to topic")
		panic(err)
	}

	incomingMessages := make(chan pubsub.Message)
	go readSub(subscription, incomingMessages)
	go getNetworkTopics()

MainLoop:
	for {
		select {
		case <-ctx.Done():
			break MainLoop
		case msg := <-incomingMessages:
			{
				handler.HandleIncomingMessage(msg, func(textMessage p2chat.TextMessage) {
					messageQueue.PushFront(textMessage)
				})
			}
		case newPeer := <-peerChan:
			{
				fmt.Println("\nFound peer:", newPeer, ", add address to peerstore")

				// Adding peer addresses to local peerstore
				host.Peerstore().AddAddr(newPeer.ID, newPeer.Addrs[0], peerstore.PermanentAddrTTL)
				// Connect to the peer
				if err := host.Connect(ctx, newPeer); err != nil {
					fmt.Println("Connection failed:", err)
				}
				fmt.Println("\nConnected to:", newPeer)
			}
		}
	}
}

// TODO: get this part to separate file (flags or whatever). all defaults parameters and their parsing should be done from separate file
func GetConfig(rendezvous *string, pid *string, host *string, port *int) *Config {
	c := &Config{}

	if *rendezvous != "" && rendezvous != nil {
		c.RendezvousString = *rendezvous
	} else {
		c.RendezvousString = "moonshard"
	}

	if *pid != "" && pid != nil {
		c.ProtocolID = *pid
	} else {
		c.ProtocolID = "/chat/1.1.0"
	}

	if *host != "" && host != nil {
		c.ListenHost = *host
	} else {
		c.ListenHost = "0.0.0.0"
	}

	if *port != 0 && port != nil && !(*port < 0) && !(*port > 65535) {
		c.ListenPort = *port
	} else {
		c.ListenPort = 4001
	}

	return c
}

func getNetworkTopics() {
	ctx := globalCtx
	handler.RequestNetworkTopics(ctx)
}

func GetMessages() string {
	textMessage := messageQueue.PopBack()
	if textMessage != nil {
		jsonData, err := json.Marshal(textMessage)
		if err != nil {
			return ""
		}
		return string(jsonData)
	}
	return ""
}
