package cmd

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// Command is an CLI command abstraction
type Command interface {
	Run() error
}

// CommandConstructor creates new command using command line arguments,
// cobra context and initialized components
type CommandConstructor func(cmd *cobra.Command, args []string, components *Components) (command Command, err error)

// RunAndCleanup initializes components, constructs command, runs it, cleanups
// components and returns results
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

// ListCmd command to list channels, claims, etc
var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "List channels, claims in progress, etc",
	Long:  "List command prints lists of objects from the shared storage; each object type has separate subcommand",
}

var FreeCallUserCmd = &cobra.Command{
	Use:   "freecall",
	Short: "Manage operations on free call users",
	Long: "List commands prints a list of all free call users for the given service," +
		"reset will set the counter to zero on free calls used, " +
		"unlock will release the lock on the given user ",
}

var GenerateEvmKeys = &cobra.Command{
	Use:   "generate-key",
	Short: "Generate new pair of keys",
	Long:  "Generate new pair of ethereum keys (private key and address)",
	Run: func(cmd *cobra.Command, args []string) {
		privateKey, err := crypto.GenerateKey()
		if err != nil {
			fmt.Printf("Failed to generate private key: %v", err)
		}

		privateKeyBytes := crypto.FromECDSA(privateKey)
		fmt.Printf("Private Key: %s\n", hexutil.Encode(privateKeyBytes)[2:]) // cut "0x"

		publicKey := privateKey.Public()
		publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
		if !ok {
			fmt.Println("Failed to cast public key to ECDSA")
		}

		address := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()
		fmt.Printf("Address: %s\n", address)
		fmt.Println("⚠️ Save these keys or add them to the daemon config. The daemon doesn't store or publish these keys anywhere!")
	},
}

const (
	UnlockChannelFlag = "unlock"
	UserIdFlag        = "user-id"
	AddressFlag       = "address"
)

var (
	cfgFile = RootCmd.PersistentFlags().StringP("config", "c", "snetd.config.json", "config file")

	autoSSLDomain      = ServeCmd.PersistentFlags().String("auto-ssl-domain", "", "enable SSL via LetsEncrypt for this domain (requires root)")
	autoSSLCacheDir    = ServeCmd.PersistentFlags().String("auto-ssl-cache", ".certs", "auto-SSL certificate cache directory")
	daemonType         = ServeCmd.PersistentFlags().StringP("type", "t", "grpc", "daemon type: one of 'grpc','http'")
	blockchainEnabled  = ServeCmd.PersistentFlags().BoolP("blockchain", "b", true, "enable blockchain processing")
	listenPort         = ServeCmd.PersistentFlags().IntP("port", "p", 5000, "daemon listen port")
	mnemonic           = ServeCmd.PersistentFlags().String("mnemonic", "", "HD wallet mnemonic")
	hdwIndex           = ServeCmd.PersistentFlags().Int("wallet-index", 0, "HD wallet index")
	dbPath             = ServeCmd.PersistentFlags().String("db-path", "snetd.db", "database file path")
	passthroughEnabled = ServeCmd.PersistentFlags().Bool("passthrough", false, "passthrough mode")
	serviceType        = ServeCmd.PersistentFlags().String("service-type", "grpc", "service type: one of 'grpc','jsonrpc','process'")
	sslCertPath        = ServeCmd.PersistentFlags().String("ssl-cert", "", "SSL certificate (.crt)")
	sslKeyPath         = ServeCmd.PersistentFlags().String("ssl-key", "", "SSL key file (.key)")
	wireEncoding       = ServeCmd.PersistentFlags().String("wire-encoding", "proto", "message encoding: one of 'proto','json'")
	pollSleep          = ServeCmd.PersistentFlags().String("poll-sleep", "5s", "blockchain poll sleep time")

	claimChannelId   string
	claimPaymentId   string
	claimSendBack    bool
	claimTimeout     string
	paymentChannelId string
	freeCallUserId   string
	freeCallAddress  string
)

func init() {
	serveCmdFlags := ServeCmd.PersistentFlags()
	vip := config.Vip()

	RootCmd.AddCommand(InitCmd)
	RootCmd.AddCommand(InitFullCmd)
	RootCmd.AddCommand(ServeCmd)

	RootCmd.AddCommand(ListCmd)
	RootCmd.AddCommand(ChannelCmd)
	RootCmd.AddCommand(VersionCmd)
	RootCmd.AddCommand(FreeCallUserCmd)
	RootCmd.AddCommand(GenerateEvmKeys)

	FreeCallUserCmd.AddCommand(FreeCallUserUnLockCmd)
	FreeCallUserCmd.AddCommand(FreeCallUserResetCmd)
	FreeCallUserCmd.AddCommand(ListFreeCallUserCmd)

	ListCmd.AddCommand(ListChannelsCmd)
	ListCmd.AddCommand(ListClaimsCmd)

	ChannelCmd.Flags().StringVarP(&paymentChannelId, UnlockChannelFlag, "u", "", "unlocks the payment channel with the given ID, see \"list channels\"")
	FreeCallUserResetCmd.Flags().StringVarP(&freeCallUserId, UserIdFlag, "u", "", "resets the free call usage count to zero for the user with the given ID")
	FreeCallUserResetCmd.Flags().StringVarP(&freeCallAddress, AddressFlag, "a", "", "resets the free call usage count to zero for the user with the given address")
	FreeCallUserUnLockCmd.Flags().StringVarP(&freeCallUserId, UserIdFlag, "u", "", "unlocks the free call user with the given ID")
	FreeCallUserUnLockCmd.Flags().StringVarP(&freeCallAddress, AddressFlag, "a", "", "unlocks the free call user with the given address")

	vip.BindPFlag(config.AutoSSLDomainKey, serveCmdFlags.Lookup("auto-ssl-domain"))
	vip.BindPFlag(config.AutoSSLCacheDirKey, serveCmdFlags.Lookup("auto-ssl-cache"))
	vip.BindPFlag(config.DaemonTypeKey, serveCmdFlags.Lookup("type"))
	vip.BindPFlag(config.BlockchainEnabledKey, serveCmdFlags.Lookup("blockchain"))

	vip.BindPFlag(config.PassthroughEnabledKey, serveCmdFlags.Lookup("passthrough"))
	vip.BindPFlag(config.SSLCertPathKey, serveCmdFlags.Lookup("ssl-cert"))
	vip.BindPFlag(config.SSLKeyPathKey, serveCmdFlags.Lookup("ssl-key"))

	cobra.OnInitialize(func() {
		zap.L().Info("Cobra initialized")
	})
}
