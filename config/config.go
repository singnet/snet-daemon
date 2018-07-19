package config

import (
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	AgentContractAddressKey    = "AGENT_CONTRACT_ADDRESS"
	BlockchainEnabledKey       = "BLOCKCHAIN_ENABLED"
	ConfigPathKey              = "CONFIG_PATH"
	DaemonListeningPortKey     = "DAEMON_LISTENING_PORT"
	DaemonTypeKey              = "DAEMON_TYPE"
	DbPathKey                  = "DB_PATH"
	EthereumJsonRpcEndpointKey = "ETHEREUM_JSON_RPC_ENDPOINT"
	ExecutablePathKey          = "EXECUTABLE_PATH"
	HdwalletIndexKey           = "HDWALLET_INDEX"
	HdwalletMnemonicKey        = "HDWALLET_MNEMONIC"
	LogLevelKey                = "LOG_LEVEL"
	PassthroughEnabledKey      = "PASSTHROUGH_ENABLED"
	PassthroughEndpointKey     = "PASSTHROUGH_ENDPOINT"
	PollSleepSecsKey           = "POLL_SLEEP_SECS"
	PrivateKeyKey              = "PRIVATE_KEY"
	ServiceTypeKey             = "SERVICE_TYPE"
	WireEncodingKey            = "WIRE_ENCODING"
)

var vip *viper.Viper

func init() {
	vip = viper.New()
	vip.SetEnvPrefix("SNET")
	vip.AutomaticEnv()

	vip.SetDefault(BlockchainEnabledKey, true)
	vip.SetDefault(ConfigPathKey, "snetd.config")
	vip.SetDefault(DaemonListeningPortKey, 5000)
	vip.SetDefault(DaemonTypeKey, "grpc")
	vip.SetDefault(DbPathKey, "snetd.db")
	vip.SetDefault(EthereumJsonRpcEndpointKey, "http://127.0.0.1:8545")
	vip.SetDefault(HdwalletIndexKey, 0)
	vip.SetDefault(LogLevelKey, 5)
	vip.SetDefault(PassthroughEnabledKey, false)
	vip.SetDefault(PollSleepSecsKey, time.Duration(5))
	vip.SetDefault(ServiceTypeKey, "grpc")
	vip.SetDefault(WireEncodingKey, "proto")

	vip.SetConfigName(vip.GetString(ConfigPathKey))

	vip.AddConfigPath(".")
	err := vip.ReadInConfig()
	if err != nil {
		log.WithError(err).Debug("error reading config")
	}
}

func Validate() {
	switch dType := vip.GetString(DaemonTypeKey); dType {
	case "grpc":
		switch sType := vip.GetString(ServiceTypeKey); sType {
		case "grpc":
		case "jsonrpc":
		case "process":
			if vip.GetString(ExecutablePathKey) == "" {
				log.Panic("EXECUTABLE required with SERVICE_TYPE 'process'")
			}
		default:
			log.Panicf("unrecognized SERVICE_TYPE '%+v'", sType)
		}

		switch enc := vip.GetString(WireEncodingKey); enc {
		case "proto":
		case "json":
		default:
			log.Panicf("unrecognized WIRE_ENCODING '%+v'", enc)
		}
	case "http":
	default:
		log.Panicf("unrecognized DAEMON_TYPE '%+v'", dType)
	}

	if vip.GetBool(BlockchainEnabledKey) {
		if vip.GetString(PrivateKeyKey) == "" && vip.GetString(HdwalletMnemonicKey) == "" {
			log.Panic("either PRIVATE_KEY or HDWALLET_MNEMONIC are required")
		}
	}
}

func GetString(key string) string {
	return vip.GetString(key)
}

func GetInt(key string) int {
	return vip.GetInt(key)
}

func GetDuration(key string) time.Duration {
	return vip.GetDuration(key)
}

func GetBool(key string) bool {
	return vip.GetBool(key)
}
