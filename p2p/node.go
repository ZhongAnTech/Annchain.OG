package p2p

import (
	"context"
	"strings"
	"fmt"
	"log"
	host "github.com/libp2p/go-libp2p-host"
	peer "github.com/libp2p/go-libp2p-peer"
	ma "github.com/multiformats/go-multiaddr"
	mrand "math/rand"
	ds "github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-crypto"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	rhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
)

// node client version
const clientVersion = "go-p2p-node/0.0.1"

// Node type - a p2p host implementing one or more p2p protocols
type Node struct {
	host.Host        // lib-p2p host
	*AddPeerProtocol // addpeer protocol impl
	*RequestProtocol // for peers to request data

	number int
}

// NewNode creates a new node with its implemented protocols
func newNode(ctx context.Context, host host.Host, number int) *Node {
	node := &Node{Host: host, number: number}
	node.AddPeerProtocol = NewAddPeerProtocol(node)
	node.RequestProtocol = NewRequestProtocol(node)
	return node
}

func (n *Node) Name() string {
	id := n.ID().Pretty()
	return fmt.Sprintf("<Node %d %s>", n.number, id[2:8])
}

func (n *Node) GetFullAddr() string {
	hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", n.ID().Pretty()))

	// Now we can build a full multiaddress to reach this host
	// by encapsulating both addresses:
	addr := n.Addrs()[0]
	fullAddr := addr.Encapsulate(hostAddr)
	return fullAddr.String()
}

// TODO: should be changed to `Knows` and `HasConnections`
func (n *Node) IsPeer(peerID peer.ID) bool {
	for _, value := range n.Peerstore().Peers() {
		if value == peerID {
			return true
		}
	}
	return false
}

func makeKey(seed int64) (crypto.PrivKey, peer.ID, error) {
	// If the seed is zero, use real cryptographic randomness. Otherwise, use a
	// deterministic randomness source to make generated keys stay the same
	// across multiple runs
	r := mrand.New(mrand.NewSource(seed))
	// r := rand.Reader

	// Generate a key pair for this host. We will use it at least
	// to obtain a valid host ID.
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		return nil, "", err
	}

	// Get the peer id
	pid, err := peer.IDFromPrivateKey(priv)
	if err != nil {
		return nil, "", err
	}
	return priv, pid, nil
}

// makeNode creates a LibP2P host with a random peer ID listening on the
// given multiaddress. It will use secio if secio is true.
func makeNode(
	ctx context.Context,
	listenPort int,
	randseed int64,secio bool,
	doBootstrapping bool,
	bootstrapPeers []pstore.PeerInfo) (*Node, error) {
	// FIXME: should be set to localhost if we don't want to expose it to outside
	listenAddrString := fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort)

	priv, _, err := makeKey(randseed)
	if err != nil {
		return nil, err
	}
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(listenAddrString),
		libp2p.Identity(priv),
	}

	if !secio {
		opts = append(opts, libp2p.NoSecurity)
	}

	basicHost, err := libp2p.New(ctx, opts...)
	if err != nil {
		panic(err)
	}

	// Construct a datastore (needed by the DHT). This is just a simple, in-memory thread-safe datastore.
	dstore := dsync.MutexWrap(ds.NewMapDatastore())

	// Make the DHT
	dht := kaddht.NewDHT(ctx, basicHost, dstore)

	// Make the routed host
	routedHost := rhost.Wrap(basicHost, dht)

	if doBootstrapping {
		// try to connect to the chosen nodes
		bootstrapConnect(ctx, routedHost, bootstrapPeers)

		err = dht.Bootstrap(ctx)
		if err != nil {
			return nil, err
		}
	}

	// Make a host that listens on the given multiaddress
	node := newNode(ctx, routedHost, int(randseed))

	log.Printf("I am %s\n", node.GetFullAddr())

	return node, nil
}


func NewNode(listenPort int, seed int64,secio bool, bootnodesStr string, doBootstrapping bool) (node *Node,err error){
	var bootnodes []pstore.PeerInfo
	if bootnodesStr == "" {
		bootnodes = []pstore.PeerInfo{}
	} else {
		bootnodes = convertPeers(strings.Split(bootnodesStr, ","))
	}
	ctx := context.Background()
	node, err = makeNode(ctx, listenPort, seed,secio, doBootstrapping, bootnodes)
	if err != nil {
		log.Fatal(err)
	}
	return 
}
