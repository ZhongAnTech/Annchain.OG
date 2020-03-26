package cmd

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/common/log"
	"github.com/spf13/cobra"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Controller struct {
	knownPeersDedup map[string]bool
	knownPeers      []string
	mu              sync.RWMutex
	srv             *http.Server
}

type RegisterResponse struct {
	Status  int
	YouPeer string
	Peers   []string
}

func (s *Controller) InitDefault() {
	s.knownPeersDedup = make(map[string]bool)

	router := gin.Default()
	router.GET("/register", func(c *gin.Context) {
		s.mu.Lock()
		defer s.mu.Unlock()
		ip := c.ClientIP()
		if _, ok := s.knownPeersDedup[ip]; !ok {
			s.knownPeersDedup[ip] = true
			s.knownPeers = append(s.knownPeers, ip)
		}
		c.JSON(http.StatusOK, RegisterResponse{
			Status:  0,
			YouPeer: ip,
			Peers:   s.knownPeers,
		})
	})

	router.GET("/peers", func(c *gin.Context) {
		c.JSON(http.StatusOK, RegisterResponse{
			Status:  0,
			YouPeer: c.ClientIP(),
			Peers:   s.knownPeers,
		})
	})
	s.srv = &http.Server{
		Addr:    ":8000",
		Handler: router,
	}

}
func (s *Controller) Start() {
	if err := s.srv.ListenAndServe(); err != nil {
	}
}

func (s *Controller) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s.srv.Shutdown(ctx)
}

// runCmd represents the run command
var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Start a bootstrap node",
	Long:  `Start a bootstrap node to collect all peer information and let them connect each other. Simulate a p2p function`,
	Run: func(cmd *cobra.Command, args []string) {
		setupLogger()

		server := Controller{}
		server.InitDefault()
		server.Start()

		// prevent sudden stop. Do your clean up here
		var gracefulStop = make(chan os.Signal)

		signal.Notify(gracefulStop, syscall.SIGTERM)
		signal.Notify(gracefulStop, syscall.SIGINT)

		func() {
			sig := <-gracefulStop
			log.Warnf("caught sig: %+v", sig)
			log.Warn("Exiting... Please do no kill me")
			// clean job
			server.Stop()
			os.Exit(0)
		}()

	},
}

func init() {
	rootCmd.AddCommand(bootstrapCmd)
}
