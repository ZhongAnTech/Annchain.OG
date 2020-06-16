package types

type verifiedType byte

const (
	VerifiedNone verifiedType = iota
	VerifiedFormat
	VerifiedGraph
	VerifiedFormatAndGraph
)

func (t verifiedType) String() string {
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

func (t verifiedType) IsFormatVerified() bool {
	if t == VerifiedFormat || t == VerifiedFormatAndGraph {
		return true
	}
	return false
}

func (t verifiedType) IsGraphVerified() bool {
	if t == VerifiedGraph || t == VerifiedFormatAndGraph {
		return true
	}
	return false
}

func (t verifiedType) merge(v verifiedType) verifiedType {
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
