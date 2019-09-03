package annsensus_test

import (
	"github.com/annchain/OG/common/crypto"
	"github.com/annchain/OG/consensus/dkg"
	"github.com/annchain/kyber/v3/pairing/bn256"
)

func generatePeers(suite *bn256.Suite, n int) []dkg.PartSec {
	signer := crypto.NewSigner(crypto.CryptoTypeSecp256k1)
	var peerInfos []dkg.PartSec
	for i := 0; i < n; i++ {
		pubKey, privKey := signer.RandomKeyPair()
		address := pubKey.Address()
		// dkg kyber pub/priv key
		dkgPrivKey, dkgPubKey := dkg.GenPartnerPair(suite)

		peerInfos = append(peerInfos, dkg.PartSec{
			PartPub: dkg.PartPub{
				Point: dkgPubKey,
				Peer: dkg.PeerInfo{
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

func setupPartners(termId uint32, numParts int, threshold int) ([]*dkg.DkgPartner, []dkg.PartSec) {
	suite := bn256.NewSuiteG2()

	// generate PeerInfos
	PartSecs := generatePeers(suite, numParts)
	var partPubs []dkg.PartPub
	for _, peer := range PartSecs {
		partPubs = append(partPubs, peer.PartPub)
	}

	var peerChans []chan dkg.DkgMessage

	// prepare incoming channels
	for i := 0; i < numParts; i++ {
		peerChans = append(peerChans, make(chan dkg.DkgMessage, 5000))
	}

	var partners []*dkg.DkgPartner

	for i := 0; i < numParts; i++ {
		partner, err := dkg.NewDkgPartner(suite, termId, numParts, threshold, partPubs, PartSecs[i])
		if err != nil {
			panic(err)
		}
		//communicator := dkg.NewDummyDkgPeerCommunicator(i, peerChans[i], peerChans)
		//partner.PeerCommunicator = communicator
		//communicator.Run()

		partners = append(partners, partner)
	}
	return partners, PartSecs
}
