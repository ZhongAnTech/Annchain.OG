package archive

//import (
//	"fmt"
//	"github.com/annchain/OG/common"
//	"github.com/annchain/OG/common/hexutil"
//)

//func (d *DkgPartner) zSelectCandidates(seq *tx_types.Sequencer) {
//	d.mu.RLock()
//	defer d.mu.RUnlock()
//	defer func() {
//		//set nil after select
//		d.term.ClearCampaigns()
//	}()
//	log := d.log()
//	campaigns := d.term.Campaigns()
//	if len(campaigns) == d.partner.NbParticipants {
//		log.Debug("campaign number is equal to participant number ,all will be senator")
//		var txs types.Txis
//		for _, cp := range campaigns {
//			if bytes.Equal(cp.PublicKey, d.myAccount.PublicKey.KeyBytes) {
//				d.isValidPartner = true
//				d.dkgOn = true
//			}
//			txs = append(txs, cp)
//		}
//		sort.Sort(txs)
//		log.WithField("txs ", txs).Debug("lucky cps")
//		for _, tx := range txs {
//			cp := tx.(*tx_types.Campaign)
//			publicKey := ogcrypto.Signer.PublicKeyFromBytes(cp.PublicKey)
//			d.term.AddCandidate(cp, publicKey)
//			if d.isValidPartner {
//				d.addPartner(cp)
//			}
//		}
//
//		if d.isValidPartner {
//			//d.generateDkg()
//			log.Debug("you are lucky one")
//		}
//		return
//	}
//	if len(campaigns) < d.partner.NbParticipants {
//		panic("never come here , programmer error")
//	}
//	randomSeed := CalculateRandomSeed(seq.Signature)
//	log.WithField("rand seed ", hexutil.Encode(randomSeed)).Debug("generated")
//	var vrfSelections VrfSelections
//	var j int
//	for addr, cp := range campaigns {
//		vrfSelect := VrfSelection{
//			addr:   addr,
//			Vrf:    cp.Vrf.Vrf,
//			XORVRF: XOR(cp.Vrf.Vrf, randomSeed),
//			Id:     j,
//		}
//		j++
//		vrfSelections = append(vrfSelections, vrfSelect)
//	}
//	log.Debugf("we have %d capmpaigns, select %d of them ", len(campaigns), d.partner.NbParticipants)
//	for j, v := range vrfSelections {
//		log.WithField("v", v).WithField(" j ", j).Trace("before sort")
//	}
//	//log.Trace(vrfSelections)
//	sort.Sort(vrfSelections)
//	log.WithField("txs ", vrfSelections).Debug("lucky cps")
//	for j, v := range vrfSelections {
//		if j == d.partner.NbParticipants {
//			break
//		}
//		cp := d.term.GetCampaign(v.addr)
//		if cp == nil {
//			panic("cp is nil")
//		}
//		publicKey := ogcrypto.Signer.PublicKeyFromBytes(cp.PublicKey)
//		d.term.AddCandidate(cp, publicKey)
//		log.WithField("v", v).WithField(" j ", j).Trace("you are lucky one")
//		if bytes.Equal(cp.PublicKey, d.myAccount.PublicKey.KeyBytes) {
//			log.Debug("congratulation i am a partner of dkg")
//			d.isValidPartner = true
//			d.partner.Id = uint32(j)
//		}
//		//add here with sorted
//		d.addPartner(cp)
//	}
//	if !d.isValidPartner {
//		log.Debug("unfortunately i am not a partner of dkg")
//	} else {
//		d.dkgOn = true
//		//d.generateDkg()
//	}
//
//	//log.Debug(vrfSelections)
//
//	//for _, camp := range camps {
//	//	d.partner.PartPubs = append(d.partner.PartPubs, camp.GetDkgPublicKey())
//	//	d.partner.addressIndex[camp.Sender()] = len(d.partner.PartPubs) - 1
//	//}
//}

//type VrfSelection struct {
//	addr   common.Address
//	Vrf    hexutil.KeyBytes
//	XORVRF hexutil.KeyBytes
//	Id     int //for test
//}
//
//
//
//type VrfSelections []VrfSelection
//
//func (a VrfSelections) Len() int      { return len(a) }
//func (a VrfSelections) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
//func (a VrfSelections) Less(i, j int) bool {
//	return hexutil.Encode(a[i].XORVRF) < hexutil.Encode(a[j].XORVRF)
//}
//
//func (v VrfSelection) String() string {
//	return fmt.Sprintf("id-%d-a-%s-v-%s-xor-%s", v.Id, hexutil.Encode(v.addr.KeyBytes[:4]), hexutil.Encode(v.Vrf[:4]), hexutil.Encode(v.XORVRF[:4]))
//}
//
//// XOR takes two byte slices, XORs them together, returns the resulting slice.
//func XOR(a, b []byte) []byte {
//	c := make([]byte, len(a))
//	for i := 0; i < len(a); i++ {
//		c[i] = a[i] ^ b[i]
//	}
//	return c
//}
//
