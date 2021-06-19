package src

import (
	"context"
	"crypto/rand"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	discovery "github.com/libp2p/go-libp2p-discovery"
	host "github.com/libp2p/go-libp2p-host"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	tls "github.com/libp2p/go-libp2p-tls"
	yamux "github.com/libp2p/go-libp2p-yamux"
	"github.com/libp2p/go-tcp-transport"
	"github.com/multiformats/go-multiaddr"
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
	identity := libp2p.Identity(prvkey)
	// Handle any potential error
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalln("P2P Identity Generation Failed!")
	}

	// Debug log
	logrus.Debugln("Created Identity Configurations for the P2P Host.")

	// Set up TLS secured transport options
	tlstransport, err := tls.New(prvkey)
	security := libp2p.Security(tls.ID, tlstransport)
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

	// Create a new PubSub service which uses a GossipSub router
	gossipsub, err := pubsub.NewGossipSub(ctx, libhost, pubsub.WithDiscovery(routingdiscovery))
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
	discovery.Advertise(p2p.Ctx, p2p.Discovery, service)

	// Debug log
	logrus.Debugln("Advertised Peerchat Service.")

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
			if err := p2p.Host.Connect(p2p.Ctx, peer); err != nil {
				// Handle any potential error
				logrus.WithFields(logrus.Fields{
					"error": err.Error(),
				}).Fatalln("P2P Peer Connection Failed!")
			}
		}
	}()

	// Debug log
	logrus.Debugln("Started Peer Connection Handler.")
}
