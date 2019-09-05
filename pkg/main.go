package p2mobile

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"strings"

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

var (
	myself           host.Host
	globalCtx        context.Context
	globalCtxCancel  context.CancelFunc
	Pb               *pubsub.PubSub
	networkTopics    = mapset.NewSet()
	messageQueue     utils.Queue
	handler          p2chat.Handler
	serviceTopic     string
	subscribedTopics map[string]chan struct{} // Pair "Topic-Channel", channel need for stopping listening
)

// this function get new messages from subscribed topic
func readSub(subscription *pubsub.Subscription, incomingMessagesChan chan pubsub.Message, stopChannel chan struct{}) {
	ctx := globalCtx
	for {
		select {
		case <-ctx.Done():
			return
		case <-stopChannel:
			{
				close(incomingMessagesChan)
				close(stopChannel)
				return
			}
		default:
		}
		msg, err := subscription.Next(context.Background())
		if err != nil {
			log.Println("Error reading from buffer")
			panic(err)
		}

		if string(msg.Data) == "" {
			return
		}
		if string(msg.Data) != "\n" {
			addr, err := peer.IDFromBytes(msg.From)
			if err != nil {
				log.Println("Error occurred when reading message From field...")
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
		Flag: api.FlagGenericMessage,
	}

	sendData, err := json.Marshal(message)
	if err != nil {
		log.Println("Error occurred when marshalling message object")
		return
	}

	handler.PbMutex.Lock()
	err = Pb.Publish(topic, sendData)
	handler.PbMutex.Unlock()
	if err != nil {
		log.Println("Error occurred when publishing")
		return
	}
}

func Start(rendezvous string, pid string, listenHost string, port int) {
	subscribedTopics = make(map[string]chan struct{})
	utils.SetConfig(&utils.Configuration{
		RendezvousString: rendezvous,
		ProtocolID:       pid,
		ListenHost:       listenHost,
		ListenPort:       port,
	})

	cfg := utils.GetConfig()

	serviceTopic = cfg.RendezvousString

	log.Printf("[*] Listening on: %s with port: %d\n", cfg.ListenHost, cfg.ListenPort)

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

	multiaddress := fmt.Sprintf("/ip4/%s/tcp/%v/p2p/%s\n", cfg.ListenHost, cfg.ListenPort, host.ID().Pretty())
	log.Printf("\n[*] Your Multiaddress Is: %s", multiaddress)

	myself = host

	// Initialize pubsub object
	pb, err := pubsub.NewFloodsubWithProtocols(context.Background(), host, []protocol.ID{protocol.ID(cfg.ProtocolID)}, pubsub.WithMessageSigning(true), pubsub.WithStrictSignatureVerification(true))
	if err != nil {
		log.Println("Error occurred when create PubSub")
		panic(err)
	}

	Pb = pb

	handler = p2chat.NewHandler(pb, serviceTopic, multiaddress, &networkTopics)

	peerChan, err := p2chat.InitMDNS(ctx, host, serviceTopic)
	if err != nil {
		panic(err)
	}

	SubscribeToTopic(serviceTopic)
	go GetNetworkTopics()

MainLoop:
	for {
		select {
		case <-ctx.Done():
			break MainLoop
		case newPeer := <-peerChan:
			{
				log.Println("\nFound peer:", newPeer, ", add address to peerstore")

				// Adding peer addresses to local peerstore
				host.Peerstore().AddAddr(newPeer.ID, newPeer.Addrs[0], peerstore.PermanentAddrTTL)
				// Connect to the peer
				if err := host.Connect(ctx, newPeer); err != nil {
					log.Println("Connection failed:", err)
				}
				log.Println("\nConnected to:", newPeer)
			}
		}
	}
}

func SetMatrixID(mxID string) {
	handler.SetMatrixID(mxID)
}

func GetNetworkTopics() {
	ctx := globalCtx
	handler.RequestNetworkTopics(ctx)
}

func GetPeersIdentity() {
	ctx := globalCtx
	handler.RequestPeersIdentity(ctx)
}

func GetTopics() []byte {
	topics := handler.GetTopics()
	return convertStringSliceToBytes(topics)
}

func GetPeers(topic string) []byte {
	var peersStrings []string

	for _, peer := range handler.GetPeers(topic) {
		peersStrings = append(peersStrings, string(peer))
	}

	return convertStringSliceToBytes(peersStrings)
}

func convertStringSliceToBytes(pids []string) []byte {
	return []byte(strings.Join(pids, " "))
}

func BlacklistPeer(pid string) {
	handler.BlacklistPeer(peer.ID(pid))
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

func SubscribeToTopic(topic string) {
	incomingMessages := make(chan pubsub.Message)
	subscription, err := Pb.Subscribe(topic)
	if err != nil {
		panic(err)
	}
	stopChan := make(chan struct{})
	subscribedTopics[topic] = stopChan
	go readSub(subscription, incomingMessages, stopChan)

ListenLoop:
	for {
		select {
		case <-globalCtx.Done():
			break ListenLoop
		case msg, ok := <-incomingMessages:
			{
				if ok {
					handler.HandleIncomingMessage(topic, msg, func(textMessage p2chat.TextMessage) {
						messageQueue.PushFront(textMessage)
					})
				} else {
					break ListenLoop
				}
			}
		}
	}
}

func UnsubscribeFromTopic(topic string) {
	if subscribedTopics[topic] != nil {
		close(subscribedTopics[topic])
		delete(subscribedTopics, topic)
	}
}
