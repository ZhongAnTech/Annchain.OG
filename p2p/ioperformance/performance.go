package ioperformance

import (
	"github.com/annchain/OG/common"
	"github.com/annchain/OG/common/goroutine"
	"sync"

	"time"
)

type IoData struct {
	send    common.IOSize
	recv    common.IOSize
	Send    string    `json:"send"`
	Recv    string    `json:"recv"`
	SendNum int       `json:"send_num"`
	RecvNum int       `json:"recv_num"`
	Time    time.Time `json:"time"`
}

type IoDataInfo struct {
	DataSize     []IoData `json:"data_size"`
	AvgSend      string   `json:"avg_send"`
	TotalSendNum int      `json:"total_send_num"`
	AvgRecv      string   `json:"avg_recv"`
	TotalRecvNum int      `json:"total_receiv_num"`
}

type iOPerformance struct {
	quit     chan bool
	dataSize []IoData
	send     int
	sendNum  int
	recv     int
	recvNum  int

	mu sync.RWMutex
}

var performance *iOPerformance

func Init() *iOPerformance {
	performance = new(iOPerformance)
	performance.quit = make(chan bool)
	return performance
}

func (i *iOPerformance) Start() {
	goroutine.New(i.run)
}

func (i *iOPerformance) Stop() {
	close(i.quit)
}

func (i *iOPerformance) Name() string {
	return "iOPerformance"
}

func GetNetPerformance() *IoDataInfo {
	i := performance
	var info IoDataInfo
	var dataSize []IoData
	i.mu.RLock()
	dataSize = i.dataSize
	i.mu.RUnlock()
	if len(dataSize) == 0 {
		return &info
	}
	totalSend := common.IOSize(0)
	totalReceiv := common.IOSize(0)
	for i, d := range dataSize {
		totalSend += d.send
		totalReceiv += d.recv
		info.TotalRecvNum += d.RecvNum
		info.TotalSendNum += d.SendNum
		dataSize[i].Send = d.send.String()
		dataSize[i].Recv = d.recv.String()
		info.DataSize = append(info.DataSize, dataSize[len(dataSize)-i-1])
	}
	avgSend := totalSend / common.IOSize(len(dataSize))
	avgRecv := totalReceiv / common.IOSize(len(dataSize))
	info.AvgSend = avgSend.String()
	info.AvgRecv = avgRecv.String()
	return &info
}

func AddSendSize(size int) {
	performance.AddSendSize(size)
}

func AddRecvSize(size int) {
	performance.AddRecvSize(size)
}

func (i *iOPerformance) AddSendSize(size int) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.send += size
	i.sendNum++
}

func (i *iOPerformance) AddRecvSize(size int) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.recv += size
	i.recvNum++
}

func (s *iOPerformance) run() {
	var i int
	for {
		select {
		case <-time.After(time.Second):
			s.mu.Lock()
			if i == 60 {
				ioData := IoData{send: common.IOSize(s.send), recv: common.IOSize(s.recv), SendNum: s.sendNum, RecvNum: s.recvNum}
				ioData.Time = time.Now()
				s.dataSize = s.dataSize[1:]
				s.dataSize = append(s.dataSize, ioData)
				s.send, s.recv, s.sendNum, s.recvNum = 0, 0, 0, 0

			} else {
				ioData := IoData{send: common.IOSize(s.send), recv: common.IOSize(s.recv), SendNum: s.sendNum, RecvNum: s.recvNum}
				ioData.Time = time.Now()
				s.dataSize = append(s.dataSize, ioData)
				s.send, s.recv, s.sendNum, s.recvNum = 0, 0, 0, 0
				i++
			}
			s.mu.Unlock()
			//logrus.WithField("data ", s.dataSize[0]).Debug("performance")
		//
		case <-s.quit:
			return
		}
	}
}
