package src

import (
	"context"
	"crypto/rand"
	"crypto/sha256"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	discovery "github.com/libp2p/go-libp2p-discovery"
	host "github.com/libp2p/go-libp2p-host"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	tls "github.com/libp2p/go-libp2p-tls"
	yamux "github.com/libp2p/go-libp2p-yamux"
	"github.com/libp2p/go-tcp-transport"
	"github.com/mr-tron/base58/base58"
	"github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multihash"
	"github.com/sirupsen/logrus"
)

const serviceCID = "manishmeganathan/peerchat"

// A structure that represents a P2P Host
type P2P struct {
	// Represents the host context layer
	Ctx context.Context

	// Represents the libp2p host
	Host host.Host

	// Represents the DHT routing table
	KadDHT *dht.IpfsDHT

	// Represents the peer discovery service
	Discovery *discovery.RoutingDiscovery

	// Represents the Gossip router
	GossipRouter *pubsub.GossipSubRouter

	// Represents the PubSub Handler
	PubSub *pubsub.PubSub
}

/*
A constructor function that generates and returns a P2P object for a given context object.

Constructs a libp2p host with TLS encrypted secure transportation that works over a TCP
transport connection using a Yamux Stream Multiplexer and uses UPnP for the NAT traversal.

A Kademlia DHT is then bootstrapped on this host using the default peers offered by libp2p.
A Peer Discovery service is created from this Kademlia DHT
*/
func NewP2P(ctx context.Context) *P2P {

	// Set up the host identity options
	prvkey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, rand.Reader)
	identity := libp2p.Identity(prvkey)
	// Set up TLS secured transport options
	tlstransport, err := tls.New(prvkey)
	security := libp2p.Security(tls.ID, tlstransport)

	// Debug log
	logrus.Debugln("Created Identity and Security Configurations for the P2P Host.")

	// Set up host listener address options
	sourcemultiaddr, _ := multiaddr.NewMultiaddr("/ip4/0.0.0.0/tcp/0")
	listen := libp2p.ListenAddrs(sourcemultiaddr)

	// Debug log
	logrus.Debugln("Created Port Listening Address Configurations for the P2P Host.")

	// Set up the transport, stream mux and NAT options
	transport := libp2p.Transport(tcp.NewTCPTransport)
	muxer := libp2p.Muxer("/yamux/1.0.0", yamux.DefaultTransport)
	nat := libp2p.NATPortMap()

	// Debug log
	logrus.Debugln("Created Transport, Stream Multiplexer and NAT Configurations for the P2P Host.")

	// Construct a new LibP2P host with the options
	libhost, err := libp2p.New(ctx, listen, security, transport, muxer, identity, nat)
	// Handle any potential error
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalln("P2P Host Creation Failed!")
	}

	// Create DHT server mode option
	dhtmode := dht.Mode(dht.ModeServer)
	// Create the DHT bootstrap peers option
	dhtpeers := dht.BootstrapPeers(dht.GetDefaultBootstrapPeerAddrInfos()...)

	// Debug log
	logrus.Debugln("Created DHT Configuration Options.")

	// Start a Kademlia DHT on the host in server mode
	kaddht, err := dht.New(ctx, libhost, dhtmode, dhtpeers)
	// Handle any potential error
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalln("Kademlia DHT Creation Failed!")
	}

	// Debug log
	logrus.Debugln("Created Kademlia DHT on Host.")

	// Bootstrap the DHT
	if err := kaddht.Bootstrap(ctx); err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalln("Kademlia DHT Bootstrapping Failed!")
	}

	// Debug log
	logrus.Debugln("Bootstrapped Kademlia DHT.")

	// Create a peer discovery service using the Kad DHT
	routingdiscovery := discovery.NewRoutingDiscovery(kaddht)

	// Debug log
	logrus.Debugln("Created Peer Discovery Service.")

	// Return the P2PHost
	return &P2P{
		Ctx:          ctx,
		Host:         libhost,
		KadDHT:       kaddht,
		Discovery:    routingdiscovery,
		PubSub:       nil,
		GossipRouter: nil,
	}
}

// A method of P2PHost that generates a service CID and
// announces its ability to provide it to the network.
func (p2p *P2PHost) Announce() {
	// Log the start of the announce runtime
	logrus.Infof("Announcing Service Content ID...")

	// Hash the service content ID with SHA256
	hash := sha256.Sum256([]byte(serviceCID))
	// Append the hash with the hashing codec ID for SHA2-256 (0x12),
	// the digest size (0x20) and the hash of the service content ID
	finalhash := append([]byte{0x12, 0x20}, hash[:]...)
	// Encode the fullhash to Base58
	b58string := base58.Encode(finalhash)

	// Generate a Multihash from the base58 string
	mulhash, err := multihash.FromB58String(string(b58string))
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalln("Multihash Generation for Service Content ID Failed!")
	}

	// Generate a CID from the Multihash
	cidvalue := cid.NewCidV1(12, mulhash)

	// Announce that this host can provide the service CID
	err = p2p.KadDHT.Provide(p2p.Ctx, cidvalue, true)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalln("Service Content ID Announcement Failed!")
	}

	// Assign the CID value
	p2p.CIDValue = cidvalue
	// Log the successful announcement
	logrus.Infoln("Success! Service Content ID Announced!")
}

// A method of P2PHost that connects to peers that
// provide the same service CID in the network
func (p2p *P2PHost) Connect() {
	// Log the start of the connect runtime
	logrus.Infof("Discovering Other Service Content ID Providers...")

	// Find the other providers for the service CID
	peers, err := p2p.KadDHT.FindProviders(p2p.Ctx, p2p.CIDValue)
	// Log any potential error
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalln("Provider Discovery Failed!")
	}

	// Log the discovered peer count
	logrus.Infof("Discovered %d Peers", len(peers))
	// Declare a peer counter
	var peercount int

	// Iterate over the discovered peers
	for _, peer := range peers {
		// Ignore if the discovered peer
		if peer.ID == p2p.Host.ID() {
			continue
		}
		// Connect to the peer
		if err := p2p.Host.Connect(p2p.Ctx, peer); err == nil {
			// Increment peer count
			peercount++
		}
	}

	// Log the succesful connection
	logrus.Infof("Connected to %d Peers", peercount)
}

// // Create a new PubSub service which uses a GossipSub router
// gossip, err := pubsub.NewGossipSub(ctx, libhost)
// // Handle any potential error
// if err != nil {
// 	logrus.WithFields(logrus.Fields{
// 		"error": err.Error(),
// 	}).Fatalln("GossipSub Router Creation Failed!")
// }
