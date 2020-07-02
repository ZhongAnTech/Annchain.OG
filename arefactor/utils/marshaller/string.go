package marshaller

func AppendString(b []byte, s string) []byte {
	return AppendBytes(b, []byte(s))
}

func ReadString(b []byte) (string, []byte, error) {
	bts, b, err := ReadBytes(b)
	if err != nil {
		return "", b, err
	}
	return string(bts), b, nil
}

func CalStringSize(s string) int {
	return CalIMarshallerSize(len([]byte(s)))
}

