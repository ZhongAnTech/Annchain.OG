package rpc

import (
	"fmt"
	"github.com/annchain/OG/vm/eth/common/math"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"sync/atomic"
	"time"
)

var requestId uint32
func getRequestId()uint32 {
	if requestId > math.MaxUint32-1000 {
		requestId = 10
	}
	return atomic.AddUint32(&requestId,1)
}

// defaultLogFormatter is the default log format function Logger middleware uses.
var ginLogFormatter = func(param gin.LogFormatterParams) string {
	if logrus.GetLevel() < logrus.TraceLevel {
		return ""
	}
	var statusColor, methodColor, resetColor string
	if param.IsOutputColor() {
		statusColor = param.StatusCodeColor()
		methodColor = param.MethodColor()
		resetColor = param.ResetColor()
	}

	if param.Latency > time.Minute {
		// Truncate in a golang < 1.8 safe way
		param.Latency = param.Latency - param.Latency%time.Second
	}

	logEntry :=  fmt.Sprintf("GIN %v %s %3d %s %13v  %15s %s %-7s %s %s %s  id_%d",
		param.TimeStamp.Format("2006/01/02 - 15:04:05"),
		statusColor, param.StatusCode, resetColor,
		param.Latency,
		param.ClientIP,
		methodColor, param.Method, resetColor,
		param.Path,
		param.ErrorMessage,
		getRequestId(),

	)
	logrus.Tracef("gin log %v ", logEntry)
	//return  logEntry
	return ""
}
