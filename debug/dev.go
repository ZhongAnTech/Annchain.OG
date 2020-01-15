package main

// All stucts that need call InitDefault() before being used
type NeedInitDefault interface {
	InitDefault()
}
