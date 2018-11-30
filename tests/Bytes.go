package main

//go:generate msgp
type Bytes []byte

type Person struct {
	Name string
	Id   string
}
