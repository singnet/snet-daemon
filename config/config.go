package config

import (
	"errors"
	"fmt"
	"time"

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
	PollSleepKey               = "POLL_SLEEP"
	PrivateKeyKey              = "PRIVATE_KEY"
	ServiceTypeKey             = "SERVICE_TYPE"
	WireEncodingKey            = "WIRE_ENCODING"
)

var vip *viper.Viper

func init() {
	vip = viper.New()
	vip.SetEnvPrefix("SNET")
	vip.AutomaticEnv()

	vip.SetDefault(LogLevelKey, 5)

	vip.AddConfigPath(".")
}

func Vip() *viper.Viper {
	return vip
}

func Validate() error {
	switch dType := vip.GetString(DaemonTypeKey); dType {
	case "grpc":
		switch sType := vip.GetString(ServiceTypeKey); sType {
		case "grpc":
		case "jsonrpc":
		case "process":
			if vip.GetString(ExecutablePathKey) == "" {
				return errors.New("EXECUTABLE required with SERVICE_TYPE 'process'")
			}
		default:
			return fmt.Errorf("unrecognized SERVICE_TYPE '%+v'", sType)
		}

		switch enc := vip.GetString(WireEncodingKey); enc {
		case "proto":
		case "json":
		default:
			return fmt.Errorf("unrecognized WIRE_ENCODING '%+v'", enc)
		}
	case "http":
	default:
		return fmt.Errorf("unrecognized DAEMON_TYPE '%+v'", dType)
	}

	if vip.GetBool(BlockchainEnabledKey) {
		if vip.GetString(PrivateKeyKey) == "" && vip.GetString(HdwalletMnemonicKey) == "" {
			return errors.New("either PRIVATE_KEY or HDWALLET_MNEMONIC are required")
		}
	}

	return nil
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
