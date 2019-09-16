package p2mobile

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/MoonSHRD/p2chat-android/pkg/match"
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

const (
	// Defines the timeout when new peer was found
	peerlistConnectionTimeout = time.Millisecond * 300
)

var (
	// Pb is main object for accessing the pubsub system
	Pb *pubsub.PubSub

	// matchProcessor is object to work with matches.
	// Get all matches, get new match, add new match, etc.
	matchProcessor match.MatchProcessor

	myself          host.Host
	globalCtx       context.Context
	globalCtxCancel context.CancelFunc
	networkTopics   = mapset.NewSet()
	messageQueue    utils.Queue
	handler         p2chat.Handler
	serviceTopic    string

	// Pair "Topic-CancelFunc", function for stopping listening to topic and unsubscribing
	subscribedTopics map[string]context.CancelFunc
)

// this function get new messages from subscribed topic
func readSub(subscription *pubsub.Subscription, incomingMessagesChan chan pubsub.Message, ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			{
				close(incomingMessagesChan)
				subscription.Cancel()
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

// PublishMessage publishes message into some topic
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

// Start launches main p2chat process
func Start(rendezvous string, pid string, listenHost string, port int) {
	subscribedTopics = make(map[string]context.CancelFunc)
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

	handler = p2chat.NewHandler(pb, serviceTopic, host.ID(), &networkTopics)

	peerChan, err := p2chat.InitMDNS(ctx, host, serviceTopic)
	if err != nil {
		panic(err)
	}

	subscribeToTopic(serviceTopic)
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

				time.Sleep(peerlistConnectionTimeout)
				getMatchResponse(newPeer.ID)
			}
		}
	}
}

// GetMatchResponse collects a list of topics to which the peer is subscribed,
// collects a list of peers from these topics,
// requests to its matrixIDs and then marshals them to json
func getMatchResponse(newPeerID peer.ID) {
	// Send request for peers identity to fills up the identity map
	GetPeersIdentity()

	var peerTopics []string
	peerMatrixID := getMatrixIDFromPeerID(newPeerID)

	// Get topics this node is subscribed to check new node inclusiveness to them
	topics := handler.GetTopics()
	for _, topic := range topics {
		// Get peer list of subscribed peers to specific topic
		topicPeers := handler.GetPeers(topic)

		for _, peerID := range topicPeers {
			// Check if new peer is included to the peer list of the specific topic
			if peerID == newPeerID {
				peerTopics = append(peerTopics, topic)
			}
		}
	}

	matchProcessor.AddNewMatch(peerMatrixID, peerTopics)
}

// Returns the peer matrixID from identity map by its peerID
func getMatrixIDFromPeerID(peerID peer.ID) string {
	idenityMap := handler.GetIdentityMap()
	return idenityMap[peerID]
}

// SetMatrixID sets the matrixID of a current peer
func SetMatrixID(mxID string) {
	handler.SetMatrixID(mxID)
}

// GetNetworkTopics requests network topics from other peers
func GetNetworkTopics() {
	ctx := globalCtx
	handler.RequestNetworkTopics(ctx)
}

// GetPeersIdentity requests MatrixID from other peers
func GetPeersIdentity() {
	ctx := globalCtx
	handler.RequestPeersIdentity(ctx)
}

// GetTopics is method for getting subcribed topics of current peer
func GetTopics() string {
	var topics []string
	for key := range subscribedTopics {
		topics = append(topics, key)
	}
	return utils.ObjectToJSON(topics)
}

// GetPeers is method for getting peer ids by topic
func GetPeers(topic string) string {
	var peersStrings []string

	for _, peer := range handler.GetPeers(topic) {
		peersStrings = append(peersStrings, string(peer))
	}

	return utils.ObjectToJSON(peersStrings)
}

// BlacklistPeer blacklists peer by its peer.ID
func BlacklistPeer(pid string) {
	handler.BlacklistPeer(peer.ID(pid))
}

// GetMessages returns json message string from the message-queue
func GetMessages() string {
	textMessage := messageQueue.PopBack()
	if textMessage != nil {
		return utils.ObjectToJSON(textMessage)
	}
	return ""
}

// SubscribeToTopic allows to subscribe to specific topic. This is public API.
func SubscribeToTopic(topic string) {
	if topic == serviceTopic {
		log.Println("Manual subscription to service topic is not allowed!")
		return
	}
	if _, ok := subscribedTopics[topic]; ok {
		log.Println("You are already subscribed to the topic!")
		return
	}

	subscribeToTopic(topic)
}

func subscribeToTopic(topic string) {
	incomingMessages := make(chan pubsub.Message)
	subscription, err := Pb.Subscribe(topic)
	if err != nil {
		panic(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	subscribedTopics[topic] = cancel
	go readSub(subscription, incomingMessages, ctx)

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

// UnsubscribeFromTopic allows to unsubscribe from specific topic
func UnsubscribeFromTopic(topic string) {
	if subscribedTopics[topic] != nil {
		subscribedTopics[topic]() // cancel context
		delete(subscribedTopics, topic)
	}
}

func GetAllMatches() string {
	return matchProcessor.GetAllMatches()
}

func GetNewMatch() string {
	return matchProcessor.GetNewMatch()
}
