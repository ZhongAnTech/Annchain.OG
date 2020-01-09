package encryption

// Vault is the inmemory encrypted storage for private keys
// Currently not implemented.
type Vault struct {
	data []byte
}

func NewVault(data []byte) *Vault {
	return &Vault{
		data: data,
	}
}

func (v *Vault) Fetch() []byte {
	return v.data
}

func (v *Vault) Dump(filepath string, passphrase string) error {
	return EncryptFileDummy(filepath, v.data, passphrase)
}
