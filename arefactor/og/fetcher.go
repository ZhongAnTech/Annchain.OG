package og

import (
	"github.com/annchain/OG/arefactor/ogsyncer_interface"
)

type OGResourceFetcher struct {
}

func (O OGResourceFetcher) Fetch(request ogsyncer_interface.ResourceRequest) ogsyncer_interface.Resource {
	panic("implement me")
}
