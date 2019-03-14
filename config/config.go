package config

import (
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
)

const (
	RegistryAddressKey   = "registry_address_key" //to be read from github
	AutoSSLDomainKey     = "auto_ssl_domain"
	AutoSSLCacheDirKey   = "auto_ssl_cache_dir"
	BlockchainEnabledKey = "blockchain_enabled"
	BurstSize            = "burst_size"
	ConfigPathKey        = "config_path"

	DaemonGroupName                = "daemon_group_name"
	DaemonTypeKey                  = "daemon_type"
	DaemonEndPoint                 = "daemon_end_point"
	EthereumJsonRpcEndpointKey     = "ethereum_json_rpc_endpoint"
	ExecutablePathKey              = "executable_path"
	HdwalletIndexKey               = "hdwallet_index"
	HdwalletMnemonicKey            = "hdwallet_mnemonic"
	IpfsEndPoint                   = "ipfs_end_point"
	IpfsTimeout                    = "ipfs_timeout"
	LogKey                         = "log"
	MonitoringEnabled              = "monitoring_enabled"
	MonitoringServiceEndpoint      = "monitoring_svc_end_point"
	OrganizationId                 = "organization_id"
	ServiceId                      = "service_id"
	PassthroughEnabledKey          = "passthrough_enabled"
	PassthroughEndpointKey         = "passthrough_endpoint"
	PrivateKeyKey                  = "private_key"
	RateLimitPerMinute             = "rate_limit_per_minute"
	SSLCertPathKey                 = "ssl_cert"
	SSLKeyPathKey                  = "ssl_key"
	PaymentChannelStorageTypeKey   = "payment_channel_storage_type"
	PaymentChannelStorageClientKey = "payment_channel_storage_client"
	PaymentChannelStorageServerKey = "payment_channel_storage_server"
	//configs for Daemon Monitoring and Notification
	AlertsEMail                 = "alerts_email"
	HeartbeatServiceEndpoint    = "heartbeat_svc_end_point"
	NotificationServiceEndpoint = "notification_svc_end_point"
	ServiceHeartbeatType        = "service_heartbeat_type"
	//none|grpc|http

	defaultConfigJson string = `
{
	"auto_ssl_domain": "",
	"auto_ssl_cache_dir": ".certs",
	"blockchain_enabled": true,
	"daemon_end_point": "127.0.0.1:8080",
	"daemon_group_name":"default_group",
	"daemon_type": "grpc",
	"ethereum_json_rpc_endpoint": "http://127.0.0.1:8545",
	"hdwallet_index": 0,
	"hdwallet_mnemonic": "",
	"ipfs_end_point": "http://localhost:5002/", 
	"ipfs_timeout" : 30,
	"monitoring_enabled": true,
	"monitoring_svc_end_point": "https://n4rzw9pu76.execute-api.us-east-1.amazonaws.com/beta",
	"organization_id": "ExampleOrganizationId", 
	"passthrough_enabled": false,
	"registry_address_key": "0x4e74fefa82e83e0964f0d9f53c68e03f7298a8b2",
	"service_id": "ExampleServiceId", 
	"private_key": "",
	"ssl_cert": "",
	"ssl_key": "",
	"log":  {
		"level": "info",
		"timezone": "UTC",
		"formatter": {
			"type": "text",
			"timestamp_format": "2006-01-02T15:04:05.999999999Z07:00"
		},
		"output": {
			"type": "file",
			"file_pattern": "./snet-daemon.%Y%m%d.log",
			"current_link": "./snet-daemon.log",
			"rotation_time_in_sec": 86400,
			"max_age_in_sec": 604800,
			"rotation_count": 0
		},
		"hooks": []
	},
	"payment_channel_storage_type": "etcd",
	"payment_channel_storage_client": {
		"connection_timeout": "5s",
		"request_timeout": "3s",
		"endpoints": ["http://127.0.0.1:2379"]
	},
	"payment_channel_storage_server": {
		"id": "storage-1",
		"scheme": "http",
		"host" : "127.0.0.1",
		"client_port": 2379,
		"peer_port": 2380,
		"token": "unique-token",
		"cluster": "storage-1=http://127.0.0.1:2380",
		"startup_timeout": "1m",
		"data_dir": "storage-data-dir-1.etcd",
		"log_level": "info",
		"enabled": true
	},
	"alerts_email": "", 
	"service_heartbeat_type": "http",
	"heartbeat_svc_end_point": "http://demo3208027.mockable.io/heartbeat",
	"notification_svc_end_point": "http://demo3208027.mockable.io"
}
`
)

var vip *viper.Viper

func init() {
	var err error

	vip = viper.New()
	vip.SetEnvPrefix("SNET")
	vip.AutomaticEnv()

	var defaults = viper.New()
	err = ReadConfigFromJsonString(defaults, defaultConfigJson)
	if err != nil {
		panic(fmt.Sprintf("Cannot load default config: %v", err))
	}
	SetDefaultFromConfig(vip, defaults)

	vip.AddConfigPath(".")
}

// ReadConfigFromJsonString function reads settigs from json string to the
// config instance. String should contain valid JSON config.
func ReadConfigFromJsonString(config *viper.Viper, json string) error {
	config.SetConfigType("json")
	return config.ReadConfig(strings.NewReader(json))
}

// SetDefaultFromConfig sets all settings from defaults as default values to
// the config.
func SetDefaultFromConfig(config *viper.Viper, defaults *viper.Viper) {
	for key, value := range defaults.AllSettings() {
		config.SetDefault(key, value)
	}
}

func Vip() *viper.Viper {
	return vip
}

func Validate() error {
	switch dType := vip.GetString(DaemonTypeKey); dType {
	case "grpc":
	case "http":
	default:
		return fmt.Errorf("unrecognized DAEMON_TYPE '%+v'", dType)
	}

	certPath, keyPath := vip.GetString(SSLCertPathKey), vip.GetString(SSLKeyPathKey)
	if (certPath != "" && keyPath == "") || (certPath == "" && keyPath != "") {
		return errors.New("SSL requires both key and certificate when enabled")
	}
	// validate monitoring service endpoints
	if vip.GetBool(MonitoringEnabled) &&
		vip.GetString(MonitoringServiceEndpoint) != "" &&
		!IsValidUrl(vip.GetString(MonitoringServiceEndpoint)) {
		return errors.New("service endpoint must be a valid URL")
	}

	// Validate metrics URL and set state
	return nil
}

func LoadConfig(configFile string) error {
	vip.SetConfigFile(configFile)
	return vip.ReadInConfig()
}

func WriteConfig(configFile string) error {
	vip.SetConfigFile(configFile)
	return vip.WriteConfig()
}

func GetString(key string) string {
	return vip.GetString(key)
}

func GetInt(key string) int {
	return vip.GetInt(key)
}

func GetBigInt(key string) *big.Int {
	return big.NewInt(int64(vip.GetInt(key)))
}

func GetDuration(key string) time.Duration {
	return vip.GetDuration(key)
}

func GetBool(key string) bool {
	return vip.GetBool(key)
}

// SubWithDefault returns sub-config by keys including configuration defaults
// values. It returns nil if no such key. It is analog of the viper.Sub()
// function. This is workaround for the issue
// https://github.com/spf13/viper/issues/559
func SubWithDefault(config *viper.Viper, key string) *viper.Viper {
	var allSettingsByKey, ok = config.AllSettings()[strings.ToLower(key)]
	if !ok {
		return nil
	}

	var subMap = cast.ToStringMap(allSettingsByKey)
	var sub = viper.New()
	for subKey, value := range subMap {
		sub.Set(subKey, value)
	}

	return sub
}

var hiddenKeys = map[string]bool{
	strings.ToUpper(PrivateKeyKey):       true,
	strings.ToUpper(HdwalletMnemonicKey): true,
}

func LogConfig() {
	log.Info("Final configuration:")
	keys := vip.AllKeys()
	sort.Strings(keys)
	for _, key := range keys {
		if hiddenKeys[strings.ToUpper(key)] {
			log.Infof("%v: ***", key)
		} else {
			log.Infof("%v: %v", key, vip.Get(key))
		}
	}
}

func GetBigIntFromViper(config *viper.Viper, key string) (value *big.Int, err error) {
	value = &big.Int{}
	err = value.UnmarshalText([]byte(config.GetString(key)))
	return
}

// isValidUrl tests a string to determine if it is a url or not.
func IsValidUrl(urlToTest string) bool {
	_, err := url.ParseRequestURI(urlToTest)
	if err != nil {
		return false
	} else {
		return true
	}
}

// validates in input URL
func ValidateEmail(email string) bool {
	Re := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)
	return Re.MatchString(email)
}
