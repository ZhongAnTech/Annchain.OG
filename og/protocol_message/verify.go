package protocol_message

type VerifiedType byte

const (
	VerifiedNone VerifiedType = iota
	VerifiedFormat
	VerifiedGraph
	VerifiedFormatAndGraph
)

func (t VerifiedType) String() string {
	switch t {
	case VerifiedNone:
		return "None"
	case VerifiedFormat:
		return "Format"
	case VerifiedGraph:
		return "Graph"
	case VerifiedFormatAndGraph:
		return "FormatGraph"
	default:
		return "None"
	}
}

func (t VerifiedType) IsFormatVerified() bool {
	if t == VerifiedFormat || t == VerifiedFormatAndGraph {
		return true
	}
	return false
}

func (t VerifiedType) IsGraphVerified() bool {
	if t == VerifiedGraph || t == VerifiedFormatAndGraph {
		return true
	}
	return false
}

func (t VerifiedType) Merge(v VerifiedType) VerifiedType {
	if v == VerifiedFormat {
		if t == VerifiedGraph {
			return VerifiedFormatAndGraph
		}
		return VerifiedFormat
	} else if v == VerifiedGraph {
		if t == VerifiedFormat {
			return VerifiedFormatAndGraph
		}
		return VerifiedGraph
	}
	//for test , never come here
	return VerifiedNone
}
