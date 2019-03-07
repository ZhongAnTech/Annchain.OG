// Copyright © 2019 Annchain Authors <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

var DOWNLOADS_PATH = ""

type Server struct {
	router *gin.Engine
	server *http.Server
	port   string
}

func main() {
	port := "18012"
	router := gin.New()
	router.GET("og_config", DownloadFile)
	router.GET("", HelpFunc)
	srv := &Server{
		server: &http.Server{
			Addr:    ":" + port,
			Handler: router,
		},
	}
	if err := srv.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		panic(fmt.Errorf("error in Http server %v", err))
	}
}

func DownloadFile(ctx *gin.Context) {
	node_id := ctx.Query("node_id")
	_, err := strconv.Atoi(node_id)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "node_id error"})
		return
	}
	fileName := "config_" + node_id + ".toml"
	targetPath := filepath.Join(DOWNLOADS_PATH, fileName)
	//This ckeck is for example, I not sure is it can prevent all possible filename attacks - will be much better if real filename will not come from user side. I not even tryed this code
	if !strings.HasPrefix(filepath.Clean(targetPath), DOWNLOADS_PATH) {
		ctx.String(403, "Look like you attacking me")
		return
	}
	//Seems this headers needed for some browsers (for example without this headers Chrome will download files as txt)
	//ctx.Header("Content-Description", "File Transfer")
	//ctx.Header("Content-Transfer-Encoding", "binary")
	//ctx.Header("Content-Disposition", "attachment; filename="+fileName )
	//ctx.Header("Content-Type", "application/octet-stream")
	fmt.Println("will serve file", targetPath)
	ctx.File(targetPath)
}

func HelpFunc(ctx *gin.Context) {
	ctx.JSON(403, gin.H{
		"error": "not allowed"})
}
