// Copyright © 2019 Annchain Authors <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package dkg

import (
	"fmt"
	"github.com/annchain/OG/common/hexutil"
	"github.com/annchain/OG/common/math"
	"github.com/annchain/kyber/v3"
	"github.com/annchain/kyber/v3/pairing/bn256"
	"github.com/annchain/kyber/v3/share/vss/pedersen"
	"github.com/stretchr/testify/require"
	"testing"
)

func genPartnerPair(p *Partner) (kyber.Scalar, kyber.Point) {
	sc := p.Suite.Scalar().Pick(p.Suite.RandomStream())
	return sc, p.Suite.Point().Mul(sc, nil)
}

func TestBLS(t *testing.T) {
	actualSigners := 7
	threshold := 5
	nbParticipants := 7

	// init all partners data structures.
	var partners []*Partner
	var partPubs []kyber.Point

	// key pair generation for each partner.
	// do it on each partner's local environment
	for i := 0; i < nbParticipants; i++ {
		s := bn256.NewSuiteG2()
		partner := &Partner{
			Threshold:      threshold,
			NbParticipants: nbParticipants,
			ID:             i,
			Suite:          s,
		}
		// partner generate pair for itself.
		partSec, partPub := genPartnerPair(partner)
		// you should always keep MyPartSec safe. Do not share.
		partner.MyPartSec = partSec
		// publish its partPub
		partPubs = append(partPubs, partPub)
		partners = append(partners, partner)
	}

	// Before here, each partner should have obtained all partPubs from the others.
	// This may be done by publishing partPub to the blockchain
	// Generate DKG.
	for _, partner := range partners {
		partner.PartPubs = partPubs
		partner.GenerateDKGer()
	}

	// Partners gossips to share deals and responses.
	for _, partner := range partners {
		//if pi == nbParticipants -1 {
		//	fmt.Println("Skip you")
		//	continue
		//}
		// get all deals that needs to be sent to other partners
		deals, err := partner.Dkger.Deals()
		require.Nil(t, err)
		for i, deal := range deals {
			// send this deal to the corresponding partner. (aligned by index i)
			// get the response from that partner
			resp, err := partners[i].Dkger.ProcessDeal(deal)
			require.Nil(t, err)
			require.Equal(t, vss.StatusApproval, resp.Response.Status)
			// cache this resp and send to others later.
			partners[i].Resps = append(partners[i].Resps, resp)
		}
	}
	// you should do ProcessResponse only after the corresponding index of Deal is received.
	// otherwise there is no deal for that resp so it will fail.
	// In real implementation under async env, cache the resp and wait for Deal coming.
	for _, partner := range partners {
		for _, resp := range partner.Resps {
			// broadcast response to all others
			for j, partnerToReceiveResponse := range partners {
				// ignore myself
				if resp.Response.Index == uint32(j) {
					continue
				}
				respResp, err := partnerToReceiveResponse.Dkger.ProcessResponse(resp)
				require.Nil(t, err)
				require.Nil(t, respResp)
			}
		}
	}

	// each partner should have the ability to aggregate pubkey
	for _, partner := range partners {
		jointPubKey, err := partner.RecoverPub()
		if err != nil {
			fmt.Printf("Partner %d cannot aggregate PubKey: %s\n", partner.ID, err)
		} else {
			fmt.Printf("Partner %d jointPubKey: %s\n", partner.ID, jointPubKey)
		}
	}

	// Each partner sign the message using its partSec and generates a partSig
	msg := []byte("Hello DKG, VSS, TBLS and BLS!")
	sigShares := make([][]byte, 0)
	for i, partner := range partners {
		// stop if we already have enough honest partners
		// signature should be generated successfully
		if i == actualSigners {
			break
		}
		sig, err := partner.Sig(msg)
		if err != nil {
			fmt.Printf("Partner %d cannot partSig: %s\n", partner.ID, err)
		} else {
			fmt.Printf("PartSig %d %s\n", i, hexutil.Encode(sig))
			sigShares = append(sigShares, sig)
		}
	}
	// announce partSig to all partners
	for _, partner := range partners {
		partner.SigShares = sigShares
	}

	// experiment only: simulate missing sigshares
	for _, partner := range partners {
		// Let's make partner #k have only k sig shares
		// any partner #k < threshold will not be able to reconstruct the JoingSignature
		partner.SigShares = partner.SigShares[0:math.MinInt(partner.ID, actualSigners)]
	}

	// recover JointSig
	for _, partner := range partners {
		jointSig, err := partner.RecoverSig(msg)
		if err != nil {
			fmt.Printf("partner %d cannot recover jointSig with %d sigshares: %s\n",
				partner.ID, len(partner.SigShares), err)
		} else {
			fmt.Printf("threshold signature from partner %d: %s\n", partner.ID, hexutil.Encode(jointSig))
			// verify if JointSig meets the JointPubkey
			err = partner.VerifyByDksPublic(msg, jointSig)
			require.NoError(t, err)
		}
	}

	// recover JointSig
	for _, partner := range partners {
		jointSig, err := partner.RecoverSig(msg)
		if err != nil {
			fmt.Printf("partner %d cannot recover jointSig with %d sigshares: %s\n",
				partner.ID, len(partner.SigShares), err)
		} else {
			fmt.Printf("threshold signature from partner %d: %s\n", partner.ID, hexutil.Encode(jointSig))
			// verify if JointSig meets the JointPubkey
			err = partner.VerifyByPubPoly(msg, jointSig)
			require.NoError(t, err)
		}
	}
}
