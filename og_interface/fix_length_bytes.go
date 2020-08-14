package og_interface

func BytesCmp(a FixLengthBytes, b FixLengthBytes) int {
	if a.Length() != b.Length() {
		return -1
	}

	ab := a.Bytes()
	bb := b.Bytes()

	for i := 0; i < a.Length(); i++ {
		if ab[i] > bb[i] {
			return 1
		} else if ab[i] < bb[i] {
			return -1
		}
	}
	return 0
}
