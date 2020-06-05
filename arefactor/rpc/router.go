package rpc

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"sort"
	"strings"
)

// writes a list of available rpc endpoints as an html page
func (rpc *RpcController) writeListOfEndpoints(c *gin.Context) {

	routerMap := map[string]string{
		// debug
		"debug": "",
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
