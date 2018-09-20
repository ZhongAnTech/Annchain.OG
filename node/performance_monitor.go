package node

import (
	"github.com/sirupsen/logrus"
	"time"
)

type ChannelHolder interface {
	Name() string
	GetChannelSizes() map[string]int
}

type PerformanceMonitor struct {
	channelHolders []ChannelHolder
	quit bool
}

var per PerformanceMonitor

func (p *PerformanceMonitor) Register(holder ChannelHolder){
	p.channelHolders = append(p.channelHolders, holder)
}

func (p *PerformanceMonitor) Start() {
	go func() {
		p.quit = false
		for !p.quit {
			fields := logrus.Fields{}
			for _, ch := range p.channelHolders{
				fields[ch.Name()] = ch.GetChannelSizes()
			}
			logrus.WithFields(fields).Warn("Performance")
			time.Sleep(time.Second * 5)
		}
	}()
}

func (p *PerformanceMonitor) Stop() {
	p.quit = true
}

func (PerformanceMonitor) Name() string {
	return "PerformanceMonitor"
}
