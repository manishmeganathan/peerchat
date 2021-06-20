package src

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	discovery "github.com/libp2p/go-libp2p-discovery"
	host "github.com/libp2p/go-libp2p-host"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/mr-tron/base58/base58"
	"github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multihash"
	"github.com/sirupsen/logrus"
)

const service = "manishmeganathan/peerchat"

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

	// Represents the PubSub Handler
	PubSub *pubsub.PubSub
}

/*
A constructor function that generates and returns a P2P object for a given context object.

Constructs a libp2p host with TLS encrypted secure transportation that works over a TCP
transport connection using a Yamux Stream Multiplexer and uses UPnP for the NAT traversal.

A Kademlia DHT is then bootstrapped on this host using the default peers offered by libp2p.
A Peer Discovery service is created from this Kademlia DHT. The PubSub handler is then
created on the host using the peer discovery service created prior.
*/
func NewP2P(ctx context.Context) *P2P {

	// Set up the host identity options
	prvkey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, rand.Reader)
	//identity := libp2p.Identity(prvkey)
	// Handle any potential error
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalln("P2P Identity Generation Failed!", prvkey)
	}

	// Debug log
	logrus.Debugln("Created Identity Configurations for the P2P Host.")

	// Set up TLS secured transport options
	//tlstransport, err := tls.New(prvkey)
	//security := libp2p.Security(tls.ID, tlstransport)
	// Handle any potential error
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalln("P2P Secure Transport Option Generation Failed!")
	}

	// Debug log
	logrus.Debugln("Created Security Configurations for the P2P Host.")

	// Set up host listener address options
	sourcemultiaddr, err := multiaddr.NewMultiaddr("/ip4/0.0.0.0/tcp/0")
	listen := libp2p.ListenAddrs(sourcemultiaddr)
	// Handle any potential error
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalln("P2P Listener Address Option Generation Failed!")
	}

	// Debug log
	logrus.Debugln("Created Port Listening Address Configurations for the P2P Host.")

	// Set up the transport, stream mux and NAT options
	//transport := libp2p.Transport(tcp.NewTCPTransport)
	//muxer := libp2p.Muxer("/yamux/1.0.0", yamux.DefaultTransport)
	nat := libp2p.NATPortMap()

	// Debug log
	logrus.Debugln("Created Transport, Stream Multiplexer and NAT Configurations for the P2P Host.")

	// Construct a new LibP2P host with the options
	libhost, err := libp2p.New(
		ctx,
		listen,
		//security,
		//transport,
		//muxer,
		//identity,
		nat,
	)
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
			if err := libhost.Connect(ctx, *peerinfo); err != nil {
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
	logrus.Infof("Success! Connected to %d out of %d Bootstrap Peers", connectedbootpeers, totalbootpeers)

	// Create a peer discovery service using the Kad DHT
	routingdiscovery := discovery.NewRoutingDiscovery(kaddht)

	// Debug log
	logrus.Debugln("Created Peer Discovery Service.")

	// Create a new PubSub service which uses a GossipSub router
	gossipsub, err := pubsub.NewGossipSub(ctx, libhost) //, pubsub.WithDiscovery(routingdiscovery))
	// Handle any potential error
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalln("GossipSub Handler Creation Failed!")
	}

	// Debug log
	logrus.Debugln("Created GossipSub Handler.")

	// Return the P2P object
	return &P2P{
		Ctx:       ctx,
		Host:      libhost,
		KadDHT:    kaddht,
		Discovery: routingdiscovery,
		PubSub:    gossipsub,
	}
}

// A method of P2P that advertises the peerchat service's
// availabilty on this node and then discovers all peers
// advertising the same service starts event handler to
// connects to new peers as they are discovered
func (p2p *P2P) Connect() {

	// Advertise the availabilty of the service on this node
	// discovery.Advertise(p2p.Ctx, p2p.Discovery, service)

	timed, err := p2p.Discovery.Advertise(p2p.Ctx, service)
	time.Sleep(time.Second * 5)

	// Debug log
	logrus.Debugln("Advertised Peerchat Service.", timed)

	// Find all peers advertising the same service
	peerchan, err := p2p.Discovery.FindPeers(p2p.Ctx, service)
	// Handle any potential error
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalln("P2P Peer Discovery Failed!")
	}

	// Debug log
	logrus.Debugln("Discovered Peerchat Service Peers.")

	// Connect to all peers being discovered by the peer discovery service
	go func() {
		// Iterate over the peer channel
		for peer := range peerchan {
			// Ignore if the discovered peer is the host itself
			if peer.ID == p2p.Host.ID() {
				continue
			}

			// Connect to the peer
			p2p.Host.Connect(context.Background(), peer)
		}
	}()

	// Debug log
	logrus.Debugln("Started Peer Connection Handler.")
}

func (p2p *P2P) Connect2() {

	hash := sha256.Sum256([]byte(service))
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

	// Find the other providers for the service CID
	peers, err := p2p.KadDHT.FindProviders(p2p.Ctx, cidvalue)
	// Log any potential error
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalln("Provider Discovery Failed!")
	}

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

	logrus.Infof("Connected to %d Peers", peercount)
}
