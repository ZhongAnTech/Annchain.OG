package rpc

import (
	"github.com/annchain/OG/arefactor/og"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
)

type RpcController struct {
	CpDefaultCommunityManager *og.DefaultCommunityManager
	Ledger                    *og.IntArrayLedger
}

func (rpc *RpcController) NewRouter() *gin.Engine {
	router := gin.New()
	//if logrus.GetLevel() > logrus.DebugLevel {
	//	logger := gin.LoggerWithConfig(gin.LoggerConfig{
	//		Formatter: ginLogFormatter,
	//		Output:    logrus.StandardLogger().Out,
	//		SkipPaths: []string{"/"},
	//	})
	//	router.Use(logger)
	//}

	router.Use(gin.RecoveryWithWriter(logrus.StandardLogger().Out))
	return rpc.addRouter(router)
}

func (rpc *RpcController) addRouter(router *gin.Engine) *gin.Engine {
	router.GET("/", rpc.writeListOfEndpoints)
	router.GET("debug", rpc.Debug)
	router.GET("generate", rpc.GenerateBlock)
	return router
}

func Response(c *gin.Context, status int, err error, data interface{}) {
	var msg string
	if err != nil {
		msg = err.Error()
	}
	c.JSON(status, gin.H{
		"err":  msg,
		"data": data,
	})
}

func (rpc *RpcController) Debug(c *gin.Context) {
	// do something
	addr := c.Query("addr")

	rpc.CpDefaultCommunityManager.SendPing(addr)
	Response(c, http.StatusOK, nil, "YES")
}

func (rpc *RpcController) GenerateBlock(c *gin.Context) {
	rpc.Ledger.AddRandomBlock(rpc.Ledger.CurrentHeight() + 1)
	Response(c, http.StatusOK, nil, rpc.Ledger.CurrentHeight()+1)
}
