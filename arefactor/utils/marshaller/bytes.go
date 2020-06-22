package marshaller

func AppendBytes(b []byte, bts []byte) []byte {
	b = AppendHeader(b, len(bts))
	b = append(b, bts...)
	return b
}

func ReadBytes(b []byte) ([]byte, []byte, error) {
	b, btsLen, err := DecodeHeader(b)
	if err != nil {
		return nil, nil, err
	}
	if len(b) < btsLen {
		return nil, nil, ErrorNotEnoughBytes(btsLen, len(b))
	}

	bts := b[:btsLen]
	b = b[btsLen:]
	return bts, b, nil
}
