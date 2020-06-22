package marshaller

const (
	Int64Size      = 9
	IntSize        = Int64Size
	UintSize       = Int64Size
	Int8Size       = 2
	Int16Size      = 3
	Int32Size      = 5
	Uint8Size      = 2
	ByteSize       = Uint8Size
	Uint16Size     = 3
	Uint32Size     = 5
	Uint64Size     = Int64Size
	Float64Size    = 9
	Float32Size    = 5
	Complex64Size  = 10
	Complex128Size = 18

	TimeSize = 15
	BoolSize = 1
	NilSize  = 1

	MapHeaderSize   = 5
	ArrayHeaderSize = 5

	BytesPrefixSize     = 5
	StringPrefixSize    = 5
	ExtensionPrefixSize = 6
)
