package cmd

import (
	"github.com/singnet/snet-daemon/config"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
)

type Runnable interface {
	Run() error
}

type CommandConstructor func(cmd *cobra.Command, args []string, components *Components) (command Runnable, err error)

func RunAndCleanup(cmd *cobra.Command, args []string, constructor CommandConstructor) (err error) {
	components := InitComponents(cmd)
	defer components.Close()

	command, err := constructor(cmd, args, components)
	if err != nil {
		return
	}

	return command.Run()
}

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

var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "List channels, claims in progress, etc",
	Long:  "List command prints lists of objects from the shared storage; each object type has separate subcommand",
}

const (
	ClaimChannelIdFlag = "channel-id"
	ClaimSendBackFlag  = "send-back"
	ClaimTimeoutFlag   = "timeout"
)

var (
	cfgFile = RootCmd.PersistentFlags().StringP("config", "c", "snetd.config.json", "config file")

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

	claimChannelId string
	claimSendBack  bool
	claimTimeout   string
)

func init() {
	serveCmdFlags := ServeCmd.PersistentFlags()
	vip := config.Vip()

	RootCmd.AddCommand(InitCmd)
	RootCmd.AddCommand(ServeCmd)
	RootCmd.AddCommand(ClaimCmd)
	RootCmd.AddCommand(ListCmd)

	ListCmd.AddCommand(ListChannelsCmd)

	ClaimCmd.Flags().StringVar(&claimChannelId, ClaimChannelIdFlag, "", "id of the payment channel to claim money")
	ClaimCmd.MarkFlagRequired(ClaimChannelIdFlag)
	ClaimCmd.Flags().BoolVar(&claimSendBack, ClaimSendBackFlag, false, "send the rest of the channel value back to channel sender")
	ClaimCmd.Flags().StringVar(&claimTimeout, ClaimTimeoutFlag, "5s", "timeout for blockchain transaction")

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

		log.SetOutput(os.Stdout)
		log.SetLevel(log.InfoLevel)

		log.Info("Cobra initialized")
	})
}
