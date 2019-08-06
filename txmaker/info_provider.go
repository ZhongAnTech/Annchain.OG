package txmaker

// InfoProvider defines ways to access chain info
// It is necessary for txmaker to pick up parents and make txs.
// Known provider inplements include RPC provider and embeded provider
type InfoProvider interface {
}
