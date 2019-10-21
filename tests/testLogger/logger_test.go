package testLogger

import (
	"fmt"
	"github.com/annchain/OG/common"
	"github.com/sirupsen/logrus"
	"testing"
)

func TestLogger(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	foo := randomFoo()
	foo2 := *randomFoo()
	fmt.Println("func 1")
	logrus.WithField("tx", foo).Tracef("bad tx")
	fmt.Println("func 2")
	logrus.WithField("tx", foo.String()).Trace("parent  is bad")
	fmt.Println("func 3")
	logrus.WithField("tx", foo2).Tracef("bad tx")
	fmt.Println("func 4")
	logrus.WithField("tx", foo2.String()).Tracef("parent is bad")
	fmt.Println("func 5")
	f := foo.Bar
	if logrus.GetLevel() > logrus.DebugLevel {
		logrus.WithField("bar ", f()).Tracef("bar")
	}
	fmt.Println("func 6")

	logrus.WithField("bar ", f()).Tracef("bar")
	fmt.Println("func 7")
	logrus.Tracef("parent is bad %s", foo2)
	fmt.Println("func 8")
	logrus.Tracef("parent is bad %s", foo2.String())

}

type Foo struct {
	Hash common.Hash
	Add  common.Address
	Name string
	Id   uint64
}

func (f Foo) String() string {

	fmt.Println("this function is called")
	s := fmt.Sprintf("[%v-%v-%v-%v]", f.Hash.String(), f.Add.String(), f.Name, f.Id)
	fmt.Println("this function is called with return value ", s)
	//logrus.Debugf("hash %v",f.Hash)
	//logrus.Debugf("hash %v",f.Hash.String())
	return s
}

func (f *Foo) Bar() uint32 {
	fmt.Println("bar is called")
	return 32
}

var globalInt uint64

func randomFoo() *Foo {
	globalInt++
	return &Foo{
		Hash: common.RandomHash(),
		Add:  common.RandomAddress(),
		Name: "test",
		Id:   globalInt,
	}
}
