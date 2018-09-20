package rpc

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"sort"
	"strings"
)

func (rpc *RpcControler) Newrouter() *gin.Engine {
	router := gin.Default()
	router.GET("/", rpc.writeListOfEndpoints)
	// init paths here
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	router.GET("status", rpc.Status)
	router.GET("net_info", rpc.NetInfo)
	router.GET("peers_info", rpc.PeersInfo)
	router.GET("og_peers_info", rpc.OgPeersInfo)
	router.GET("transaction", rpc.Transaction)
	router.GET("validators", rpc.Validator)
	router.GET("sequencer", rpc.Sequencer)
	router.GET("genesis", rpc.Genesis)
	// broadcast API
	router.POST("new_transaction", rpc.NewTransaction)
	router.GET("new_transaction", rpc.NewTransaction)

	// query API
	router.GET("query", rpc.Query)
	router.GET("query_nonce", rpc.QueryNonce)
	router.GET("query_balance", rpc.QueryBalance)
	router.GET("query_share", rpc.QueryShare)
	router.GET("contract_payload", rpc.ConstructPayload)

	router.GET("query_receipt", rpc.QueryReceipt)

	router.GET("query_contract", rpc.QueryContract)

	router.GET("debug", rpc.Debug)
	return router

}

// writes a list of available rpc endpoints as an html page
func (rpc *RpcControler) writeListOfEndpoints(c *gin.Context) {

	routerMap := map[string]string{
		// info API
		"status":        "",
		"net_info":      "",
		"peers_info":    "",
		"validators":    "",
		"sequencer":     "",
		"og_peers_info": "",
		"genesis":       "",
		// broadcast API
		"new_transaction": "tx",

		// query API
		"query":            "query",
		"query_nonce":      "address",
		"query_balance":    "address",
		"query_share":      "pubkey",
		"contract_payload": "payload, abistr",

		"query_receipt": "hash",
		"transaction":   "hash",

		"query_contract": "tx",
	}
	noArgNames := []string{}
	argNames := []string{}
	for name, args := range routerMap {
		if len(args) == 0 {
			noArgNames = append(noArgNames, name)
		} else {
			argNames = append(argNames, name)
		}
	}
	sort.Strings(noArgNames)
	sort.Strings(argNames)
	buf := new(bytes.Buffer)
	buf.WriteString("<html><body>")
	buf.WriteString("<br>Available endpoints:<br>")

	for _, name := range noArgNames {
		link := fmt.Sprintf("http://%s/%s", c.Request.Host, name)
		buf.WriteString(fmt.Sprintf("<a href=\"%s\">%s</a></br>", link, link))
	}

	buf.WriteString("<br>Endpoints that require arguments:<br>")
	for _, name := range argNames {
		link := fmt.Sprintf("http://%s/%s?", c.Request.Host, name)
		args := routerMap[name]
		argNames := strings.Split(args, ",")
		for i, argName := range argNames {
			link += argName + "=_"
			if i < len(argNames)-1 {
				link += "&"
			}
		}
		buf.WriteString(fmt.Sprintf("<a href=\"%s\">%s</a></br>", link, link))
	}
	buf.WriteString("</body></html>")
	//w.Header().Set("Content-Type", "text/html")
	//w.WriteHeader(200)
	//w.Write(buf.Bytes()) // nolint: errcheck
	c.Data(http.StatusOK, "text/html", buf.Bytes())
}
