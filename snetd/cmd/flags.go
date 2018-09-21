package cmd

import (
	"fmt"
	"github.com/singnet/snet-daemon/config"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
)

var RootCmd = &cobra.Command{
	Use: "snetd",
	Run: func(cmd *cobra.Command, args []string) {
		if command, _, err := cmd.Find(args); err != nil && command != nil {
			command.Execute()
		} else {
			ServeCmd.Run(cmd, args)
		}
	},
}

var (
	cfgFile            = RootCmd.PersistentFlags().StringP("config", "c", "snetd.config.json", "config file")
	autoSSLDomain      = ServeCmd.PersistentFlags().String("auto-ssl-domain", "", "enable SSL via LetsEncrypt for this domain (requires root)")
	autoSSLCacheDir    = ServeCmd.PersistentFlags().String("auto-ssl-cache", ".certs", "auto-SSL certificate cache directory")
	daemonType         = ServeCmd.PersistentFlags().StringP("type", "t", "grpc", "daemon type: one of 'grpc','http'")
	blockchainEnabled  = ServeCmd.PersistentFlags().BoolP("blockchain", "b", true, "enable blockchain processing")
	listenPort         = ServeCmd.PersistentFlags().IntP("port", "p", 5000, "daemon listen port")
	ethEndpoint        = ServeCmd.PersistentFlags().String("ethereum-endpoint", "http://127.0.0.1:8545", "ethereum JSON-RPC endpoint")
	mnemonic           = ServeCmd.PersistentFlags().String("mnemonic", "", "HD wallet mnemonic")
	hdwIndex           = ServeCmd.PersistentFlags().Int("wallet-index", 0, "HD wallet index")
	dbPath             = ServeCmd.PersistentFlags().String("db-path", "snetd.db", "database file path")
	passthroughEnabled = ServeCmd.PersistentFlags().Bool("passthrough", false, "passthrough mode")
	serviceType        = ServeCmd.PersistentFlags().String("service-type", "grpc", "service type: one of 'grpc','jsonrpc','process'")
	sslCertPath        = ServeCmd.PersistentFlags().String("ssl-cert", "", "SSL certificate (.crt)")
	sslKeyPath         = ServeCmd.PersistentFlags().String("ssl-key", "", "SSL key file (.key)")
	wireEncoding       = ServeCmd.PersistentFlags().String("wire-encoding", "proto", "message encoding: one of 'proto','json'")
	pollSleep          = ServeCmd.PersistentFlags().String("poll-sleep", "5s", "blockchain poll sleep time")
)

func init() {
	serveCmdFlags := ServeCmd.PersistentFlags()
	vip := config.Vip()

	RootCmd.AddCommand(InitCmd)
	RootCmd.AddCommand(ServeCmd)

	vip.BindPFlag(config.AutoSSLDomainKey, serveCmdFlags.Lookup("auto-ssl-domain"))
	vip.BindPFlag(config.AutoSSLCacheDirKey, serveCmdFlags.Lookup("auto-ssl-cache"))
	vip.BindPFlag(config.DaemonTypeKey, serveCmdFlags.Lookup("type"))
	vip.BindPFlag(config.BlockchainEnabledKey, serveCmdFlags.Lookup("blockchain"))
	vip.BindPFlag(config.DaemonListeningPortKey, serveCmdFlags.Lookup("port"))
	vip.BindPFlag(config.EthereumJsonRpcEndpointKey, serveCmdFlags.Lookup("ethereum-endpoint"))
	vip.BindPFlag(config.HdwalletMnemonicKey, serveCmdFlags.Lookup("mnemonic"))
	vip.BindPFlag(config.HdwalletIndexKey, serveCmdFlags.Lookup("wallet-index"))
	vip.BindPFlag(config.DbPathKey, serveCmdFlags.Lookup("db-path"))
	vip.BindPFlag(config.PassthroughEnabledKey, serveCmdFlags.Lookup("passthrough"))
	vip.BindPFlag(config.ServiceTypeKey, serveCmdFlags.Lookup("service-type"))
	vip.BindPFlag(config.SSLCertPathKey, serveCmdFlags.Lookup("ssl-cert"))
	vip.BindPFlag(config.SSLKeyPathKey, serveCmdFlags.Lookup("ssl-key"))
	vip.BindPFlag(config.WireEncodingKey, serveCmdFlags.Lookup("wire-encoding"))
	vip.BindPFlag(config.PollSleepKey, serveCmdFlags.Lookup("poll-sleep"))

	cobra.OnInitialize(func() {
		vip.SetConfigFile(*cfgFile)

		if RootCmd.PersistentFlags().Lookup("config").Changed || isFileExist(*cfgFile) {
			if err := vip.ReadInConfig(); err != nil {
				fmt.Println("Error reading config:", *cfgFile, err)
				os.Exit(1)
			}
			fmt.Printf("Using configuration from \"%v\" file\n", *cfgFile)
		} else {
			fmt.Println("Configuration file is not set, using default configuration")
		}

		log.SetLevel(log.Level(config.GetInt(config.LogLevelKey)))
		log.Info("Cobra initialized")
	})
}

func isFileExist(file string) bool {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return false
	} else {
		return true
	}
}
