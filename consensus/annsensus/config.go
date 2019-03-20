package annsensus


type AnnSensusCOnfig struct {
	BFTId  int `json:"bft_id"`
	DKgSecretKey  []byte `json:"d_kg_secret_key"`
	DKgJointPublicKey []byte `json:"d_kg_joint_public_key"`
}

func (a *AnnSensus) WriteConsensusData () error{
	var data []byte
	//
	_ = data
	return nil
}