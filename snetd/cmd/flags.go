package cmd

import (
	"github.com/singnet/snet-daemon/config"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	cfgFile            = ServeCmd.PersistentFlags().StringP("config", "c", "snetd.config", "config file")
	daemonType         = ServeCmd.PersistentFlags().StringP("type", "t", "grpc", "daemon type: one of 'grpc','http'")
	blockchainEnabled  = ServeCmd.PersistentFlags().BoolP("blockchain", "b", true, "enable blockchain processing")
	listenPort         = ServeCmd.PersistentFlags().IntP("port", "p", 5000, "daemon listen port")
	ethEndpoint        = ServeCmd.PersistentFlags().String("ethereum-endpoint", "http://127.0.0.1:8545", "ethereum JSON-RPC endpoint")
	mnemonic           = ServeCmd.PersistentFlags().String("mnemonic", "", "HD wallet mnemonic")
	hdwIndex           = ServeCmd.PersistentFlags().Int("wallet-index", 0, "HD wallet index")
	dbPath             = ServeCmd.PersistentFlags().String("db-path", "snetd.db", "database file path")
	passthroughEnabled = ServeCmd.PersistentFlags().Bool("passthrough", false, "passthrough mode")
	serviceType        = ServeCmd.PersistentFlags().String("service-type", "grpc", "service type: one of 'grpc','jsonrpc','process'")
	wireEncoding       = ServeCmd.PersistentFlags().String("wire-encoding", "proto", "message encoding: one of 'proto','json'")
	pollSleep          = ServeCmd.PersistentFlags().String("poll-sleep", "5s", "blockchain poll sleep time")
)

func init() {
	rf := ServeCmd.PersistentFlags()
	vip := config.Vip()

	vip.BindPFlag(config.ConfigPathKey, rf.Lookup("config"))
	vip.BindPFlag(config.DaemonTypeKey, rf.Lookup("type"))
	vip.BindPFlag(config.BlockchainEnabledKey, rf.Lookup("blockchain"))
	vip.BindPFlag(config.DaemonListeningPortKey, rf.Lookup("port"))
	vip.BindPFlag(config.EthereumJsonRpcEndpointKey, rf.Lookup("ethereum-endpoint"))
	vip.BindPFlag(config.HdwalletMnemonicKey, rf.Lookup("mnemonic"))
	vip.BindPFlag(config.HdwalletIndexKey, rf.Lookup("wallet-index"))
	vip.BindPFlag(config.DbPathKey, rf.Lookup("db-path"))
	vip.BindPFlag(config.PassthroughEnabledKey, rf.Lookup("passthrough"))
	vip.BindPFlag(config.ServiceTypeKey, rf.Lookup("service-type"))
	vip.BindPFlag(config.WireEncodingKey, rf.Lookup("wire-encoding"))
	vip.BindPFlag(config.PollSleepSecsKey, rf.Lookup("poll-sleep"))

	cobra.OnInitialize(func() {
		vip.SetConfigName(*cfgFile)

		if err := config.Read(); err != nil {
			log.WithError(err).Debug("error reading config")
		}
	})
}
