package message

import (
	"errors"
	"github.com/annchain/OG/og"
)

type OGMessageUnmarshaller interface {
	DoUnmarshal(message *og.OGMessage) error
}

type OGMessageUnmarshalManager struct {
	unmarshallers []OGMessageUnmarshaller
}

func (ma *OGMessageUnmarshalManager) RegisterUnmarshall(m OGMessageUnmarshaller) {
	ma.unmarshallers = append(ma.unmarshallers, m)
}

func (ma *OGMessageUnmarshalManager) Unmarshal(message *og.OGMessage) error {
	// try plugins
	for _, m := range ma.unmarshallers {
		err := m.DoUnmarshal(message)
		if err == nil {
			return nil
		}
	}
	return errors.New("message cannot be unmarshalled. check messageunmarshalmanager")

}

type OGBasicMessageUnmarshaller struct {
}

func (d OGBasicMessageUnmarshaller) DoUnmarshal(message *og.OGMessage) error {
	err := message.Unmarshal()
	return err
}
