package vm

// InterpreterConfig are the configuration options for the Interpreter
type InterpreterConfig struct {
	// Debug enabled debugging Interpreter options
	Debug bool
	// Tracer is the op code logger
	Tracer Tracer
	// Enable recording of SHA3/keccak preimages
	EnablePreimageRecording bool
}
