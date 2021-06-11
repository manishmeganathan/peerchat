package src

import (
	"context"
	"crypto/sha256"
	"sync"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-host"
	"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-pubsub"
	"github.com/mr-tron/base58/base58"
	"github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multihash"
	"github.com/sirupsen/logrus"
)

const serviceCID = "manishmeganathan/peerchat"

// A structure that represents P2P Host
type P2PHost struct {
	// Represents the host context layer
	Ctx context.Context

	// Represents the libp2p host
	Host host.Host

	// Represents the DHT routing table
	KadDHT *dht.IpfsDHT

	// Represents the PubSub router
	PubSubRouter *pubsub.PubSub
}

/*
A constructor function that generates and returns a P2PHost for a given context object.

Constructs a libp2p host with a multiaddr on 0.0.0.0/0 IPV4 address and configure it
with NATPortMap to open a port in the firewall using UPnP. A GossipSub pubsub router
is initialized for transport and a Kademlia DHT for peer discovery
*/
func NewP2PHost(ctx context.Context) *P2PHost {
	// Create a new multiaddr object
	sourcemultiaddr, _ := multiaddr.NewMultiaddr("/ip4/0.0.0.0/tcp/0")

	// Construct a new LibP2P host with the multiaddr and the NAT Port Map
	libhost, err := libp2p.New(
		ctx,
		libp2p.ListenAddrs(sourcemultiaddr),
		libp2p.NATPortMap(),
	)
	// Handle any potential error
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalln("P2P Host Creation Failed!")
	}

	// Create a new PubSub service which uses a GossipSub router
	gossip, err := pubsub.NewGossipSub(ctx, libhost)
	// Handle any potential error
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalln("GossipSub Router Creation Failed!")
	}

	// Bind the LibP2P host to a Kademlia DHT peer
	kaddht, err := dht.New(ctx, libhost, dht.Mode(dht.ModeServer))
	// Handle any potential error
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalln("Kademlia DHT Creation Failed!")
	}

	// Return the P2PHost
	return &P2PHost{
		Ctx:          ctx,
		Host:         libhost,
		KadDHT:       kaddht,
		PubSubRouter: gossip,
	}
}

// A method of P2PHost that bootstraps the Kad-DHT
// and connects to the default bootstrap peers.
func (p2p *P2PHost) Bootstrap() {
	// Bootstrap the DHT
	if err := p2p.KadDHT.Bootstrap(p2p.Ctx); err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalln("Kademlia DHT Bootstrapping Failed!")
	}
	logrus.Infoln("DHT Bootstrapped Succesfully! Connecting to Bootstrap Peers...")

	// Declare a WaitGroup
	var wg sync.WaitGroup
	// Declare counters for the number of bootstrap peers
	var connectedbootpeers int
	var totalbootpeers int

	// Iterate over the default bootstrap peers provided by libp2p
	for _, peeraddr := range dht.DefaultBootstrapPeers {
		// Retrieve the peer address information
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peeraddr)

		// Incremenent waitgroup counter
		wg.Add(1)
		// Start a goroutine to connect to each bootstrap peer
		go func() {
			// Defer the waitgroup decrement
			defer wg.Done()
			// Attempt to connect to the bootstrap peer
			if err := p2p.Host.Connect(p2p.Ctx, *peerinfo); err != nil {
				// Increment the total bootstrap peer count
				totalbootpeers++
			} else {
				// Increment the connected bootstrap peer count
				connectedbootpeers++
				// Increment the total bootstrap peer count
				totalbootpeers++
			}
		}()
	}

	// Wait for the waitgroup to complete
	wg.Wait()
	// Log the number of bootstrap peers connected
	logrus.Infof("Connected to %d out of %d Bootstrap Peers", connectedbootpeers, totalbootpeers)
}

// A method of P2PHost that generates a service CID and
// announces its ability to provide it to the network.
func (p2p *P2PHost) Provide() {
	// Hash the service content ID with SHA256
	hash := sha256.Sum256([]byte(serviceCID))
	// Append the hash with the encoding format for SHA2-256 (0x12),
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

	// Log the successful announcement
	logrus.Infoln("Succesfully Announced Service Content ID!")
}
