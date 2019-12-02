package dkg

import (
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/kyber/v3/pairing/bn256"
	dkg "github.com/annchain/kyber/v3/share/dkg/pedersen"
)

func GeneratePeers(suite *bn256.Suite, n int) []PartSec {
	signer := crypto.NewSigner(crypto.CryptoTypeSecp256k1)
	var peerInfos []PartSec
	for i := 0; i < n; i++ {
		pubKey, privKey := signer.RandomKeyPair()
		address := pubKey.Address()
		// dkg kyber pub/priv key
		dkgPrivKey, dkgPubKey := GenPartnerPair(suite)

		peerInfos = append(peerInfos, PartSec{
			PartPub: PartPub{
				Point: dkgPubKey,
				Peer: DkgPeer{
					Id:             i,
					PublicKey:      pubKey,
					Address:        address,
					PublicKeyBytes: nil,
				},
			},
			Scalar:     dkgPrivKey,
			PrivateKey: privKey,
		})
	}
	return peerInfos
}

// SetupAllDkgers generate all Dkgers locally (for genesis or test purporse).
func SetupAllDkgers(suite *bn256.Suite, numParts int, threshold int) (dkgers []*dkg.DistKeyGenerator, partSecs []PartSec, err error) {
	// generate PeerInfos
	partSecs = GeneratePeers(suite, numParts)
	var partPubs PartPubs = make([]PartPub, numParts)

	for i, peer := range partSecs {
		partPubs[i] = peer.PartPub
	}
	dkgers = make([]*dkg.DistKeyGenerator, numParts)
	for i := 0; i < numParts; i++ {
		dkger, err := dkg.NewDistKeyGenerator(suite, partSecs[i].Scalar, partPubs.Points(), threshold)
		if err != nil {
			panic(err)
		}
		dkgers[i] = dkger
	}
	return

	//var peerChans []chan DkgMessage
	//
	//// prepare incoming channels
	//for i := 0; i < numParts; i++ {
	//	peerChans = append(peerChans, make(chan DkgMessage, 5000))
	//}

	//var partners []*DefaultDkgPartner

	//for i := 0; i < numParts; i++ {
	//	communicator := NewDummyDkgPeerCommunicator(i, peerChans[i], peerChans)
	//	communicator.Run()
	//	partner, err := NewDefaultDkgPartner(suite, termId, numParts, threshold, partPubs, PartSecs[i],
	//		communicator, communicator)
	//	if err != nil {
	//		panic(err)
	//	}
	//
	//	partners = append(partners, partner)
	//}
	//return partners, PartSecs
}
