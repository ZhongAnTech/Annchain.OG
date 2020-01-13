package cmd

import (
	"github.com/spf13/cobra"
)

var autoTxCmd = &cobra.Command{
	Use:   "tx",
	Short: "run auto client for normal tx generation",
	Run:   runAutoClientManager,
}

func runAutoClientManager(cmd *cobra.Command, args []string) {
	//autoClientManager := &auto.AutoClientManager{
	//	SampleAccounts:         core.GetSampleAccounts(),
	//	NodeStatusDataProvider: org,
	//}
	//// TODO
	//// RegisterNewTxHandler is not for AnnSensus sending txs out.
	//// Not suitable to be used here.
	////autoClientManager.RegisterReceiver = annSensus.RegisterNewTxHandler
	//
	//accountIds := StringArrayToIntArray(viper.GetStringSlice("auto_client.tx.account_ids"))
	//
	//autoClientManager.init(
	//	accountIds,
	//	delegate,
	//	myAcount,
	//)
	// syncManager.OnUpToDate = append(syncManager.OnUpToDate, autoClientManager.UpToDateEventListener)
	// just for debugging, ignoring index OOR
	//rpcServer.C.NewRequestChan = autoClientManager.Clients[0].ManualChan
	// rpcServer.C.AutoTxCli = autoClientManager
}

// legacy code to be removed.

//func (r *RpcController) AutoTx(c *gin.Context) {
//	intervalStr := c.Query("interval_us")
//	interval, err := strconv.Atoi(intervalStr)
//	if err != nil || interval < 0 {
//		Response(c, http.StatusBadRequest, fmt.Errorf("interval format err"), nil)
//		return
//	}
//	r.AutoTxCli.SetTxIntervalUs(interval)
//
//	Response(c, http.StatusOK, nil, nil)
//	return
//}
