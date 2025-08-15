package config

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/singnet/snet-daemon/v6/utils"

	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	AllowedUserFlag           = "allowed_user_flag"
	AllowedUserAddresses      = "allowed_user_addresses"
	AuthenticationAddresses   = "authentication_addresses"
	AutoSSLDomainKey          = "auto_ssl_domain"
	AutoSSLCacheDirKey        = "auto_ssl_cache_dir"
	BlockchainEnabledKey      = "blockchain_enabled"
	BlockChainNetworkSelected = "blockchain_network_selected"
	BurstSize                 = "burst_size"
	ConfigPathKey             = "config_path"
	DaemonGroupName           = "daemon_group_name"
	DaemonTypeKey             = "daemon_type" // http/grpc
	DaemonEndpoint            = "daemon_endpoint"
	ExecutablePathKey         = "executable_path"
	EnableDynamicPricing      = "enable_dynamic_pricing"
	IpfsEndpoint              = "ipfs_endpoint"
	LighthouseEndpoint        = "lighthouse_endpoint"
	IpfsTimeout               = "ipfs_timeout"
	LogKey                    = "log"
	MaxMessageSizeInMB        = "max_message_size_in_mb"
	MeteringEnabled           = "metering_enabled"
	// ModelMaintenanceEndPoint This is for grpc server end point for Model Maintenance like Create, update, delete, status check
	ModelMaintenanceEndPoint       = "model_maintenance_endpoint"
	ModelTrainingEnabled           = "model_training_enabled"
	OrganizationId                 = "organization_id"
	ServiceId                      = "service_id"
	PassthroughEnabledKey          = "passthrough_enabled"
	ServiceEndpointKey             = "service_endpoint"
	ServiceCredentialsKey          = "service_credentials"
	RateLimitPerMinute             = "rate_limit_per_minute"
	SSLCertPathKey                 = "ssl_cert"
	SSLKeyPathKey                  = "ssl_key"
	PaymentChannelCertPath         = "payment_channel_cert_path"
	PaymentChannelCaPath           = "payment_channel_ca_path"
	PaymentChannelKeyPath          = "payment_channel_key_path"
	PaymentChannelStorageTypeKey   = "payment_channel_storage_type"
	PaymentChannelStorageClientKey = "payment_channel_storage_client"
	PaymentChannelStorageServerKey = "payment_channel_storage_server"
	BlockchainProviderApiKey       = "blockchain_provider_api_key"
	FreeCallsPerAddress            = "free_calls_per_address"
	TrustedFreeCallSigners         = "trusted_free_call_signers"
	MinBalanceForFreeCall          = "min_balance_for_free_call"
	// Monitoring and Notification
	AlertsEMail                 = "alerts_email"
	HeartbeatServiceEndpoint    = "heartbeat_endpoint"
	MeteringEndpoint            = "metering_endpoint"
	PvtKeyForMetering           = "private_key_for_metering"
	PvtKeyForFreeCalls          = "private_key_for_free_calls"
	NotificationServiceEndpoint = "notification_endpoint"
	ServiceHeartbeatType        = "service_heartbeat_type"
	TokenExpiryInMinutes        = "token_expiry_in_minutes"
	TokenSecretKey              = "token_secret_key"
	//This defaultConfigJson will eventually be replaced by DefaultDaemonConfigurationSchema
	defaultConfigJson string = `
{
	"blockchain_enabled": true,
	"blockchain_network_selected": "sepolia",
	"daemon_endpoint": "127.0.0.1:8080",
	"daemon_group_name":"default_group",
	"payment_channel_storage_type": "etcd",
	"ethereum_json_rpc_http_endpoint": "https://sepolia.infura.io/v3/09027f4a13e841d48dbfefc67e7685d5",
	"ipfs_endpoint": "https://ipfs.singularitynet.io:443", 
	"lighthouse_endpoint": "https://gateway.lighthouse.storage/ipfs/", 
	"ipfs_timeout" : 30,
	"passthrough_enabled": true,
	"service_endpoint":"http://localhost:5000",
	"service_id": "YOUR_SERVICE_ID", 
	"organization_id": "YOUR_ORG_ID",
	"metering_enabled": false,
	"ssl_cert": "",
	"ssl_key": "",
	"max_message_size_in_mb" : 4,
	"daemon_type": "grpc",
    "enable_dynamic_pricing":false,
	"allowed_user_flag" :false,
	"auto_ssl_domain": "",
	"auto_ssl_cache_dir": ".certs",
	"private_key_for_free_calls": "",
	"min_balance_for_free_call" : "10",
	"trusted_free_call_signers": ["0x3Bb9b2499c283cec176e7C707Ecb495B7a961ebf", "0x7DF35C98f41F3Af0df1dc4c7F7D4C19a71Dd059F"],
	"free_calls_per_address":{},
	"log":  {
		"level": "info",
		"timezone": "UTC",
		"formatter": {
			"type": "text",
			"timestamp_format": "2006-01-02T15:04:05.999Z07:00"
		},
		"output": {
			"type": ["file", "stdout"],
			"file_pattern": "./snet-daemon.%Y%m%d.log",
			"current_link": "./snet-daemon.log",
			"max_size_in_mb": 10,
			"max_age_in_days": 7,
			"rotation_count": 0
		},
		"hooks": []
	},
	"payment_channel_storage_client": {
		"connection_timeout": "0s",
		"request_timeout": "0s",
		"hot_reload": false
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
		"log_outputs": ["./etcd-server.log"],
		"enabled": false
	},
	"alerts_email": "", 
	"service_heartbeat_type": "",
	"heartbeat_endpoint": "",
    "token_expiry_in_minutes": 1440,
    "model_training_enabled": false
}`
	MinimumConfigJson string = `{
	"blockchain_network_selected": "sepolia",
	"service_endpoint":"YOUR_SERVICE_ENDPOINT",
	"service_id": "YOUR_SERVICE_ID", 
	"organization_id": "YOUR_ORG_ID",
	"daemon_endpoint": "127.0.0.1:8080",
	"daemon_group_name":"default_group",
	"payment_channel_storage_type": "etcd",
	"ethereum_json_rpc_http_endpoint": "https://sepolia.infura.io/v3/09027f4a13e841d48dbfefc67e7685d5",
	"ipfs_endpoint": "https://ipfs.singularitynet.io:443",
	"lighthouse_endpoint": "https://gateway.lighthouse.storage/ipfs/",
	"private_key_for_free_calls": "",
	"min_balance_for_free_call" : 10,
	"trusted_free_call_signers": ["0x3Bb9b2499c283cec176e7C707Ecb495B7a961ebf", "0x7DF35C98f41F3Af0df1dc4c7F7D4C19a71Dd059F"],
	"log": {
		"output": {
			"type": ["file", "stdout"]
		}
	}}`
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

// support old deprecated alias
func migrateDeprecatedParams(v *viper.Viper) {

	deprecated := map[string]string{
		"daemon_end_point":           DaemonEndpoint,
		"ipfs_end_point":             IpfsEndpoint,
		"passthrough_endpoint":       ServiceEndpointKey,
		"metering_end_point":         MeteringEndpoint,
		"heartbeat_svc_end_point":    HeartbeatServiceEndpoint,
		"notification_svc_end_point": NotificationServiceEndpoint,
		"pvt_key_for_metering":       PvtKeyForMetering,
		"pvt_key_for_free_calls":     PvtKeyForFreeCalls,
	}

	for oldKey, newKey := range deprecated {
		if v.IsSet(oldKey) {
			val := v.Get(oldKey)
			v.Set(newKey, val)
			zap.L().Warn(fmt.Sprintf("Config parameter '%s' is deprecated, use '%s' instead", oldKey, newKey))
		}
	}
}

// SetVip allows setting a new Viper instance.
// This is useful for testing, where you may want to change the configuration.
func SetVip(newVip *viper.Viper) {
	vip = newVip
}

// ReadConfigFromJsonString function reads settings from json string to the
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

	migrateDeprecatedParams(Vip())

	switch dType := vip.GetString(DaemonTypeKey); dType {
	case "grpc":
	case "http":
		zap.L().Warn("daemon type http is not for production mode, be careful")
	default:
		return fmt.Errorf("unrecognized DAEMON_TYPE '%+v'", dType)
	}
	if err := setBlockChainNetworkDetails(BlockChainNetworkFileName); err != nil {
		return err
	}
	certPath, keyPath := vip.GetString(SSLCertPathKey), vip.GetString(SSLKeyPathKey)
	if (certPath != "" && keyPath == "") || (certPath == "" && keyPath != "") {
		return errors.New("SSL requires both key and certificate when enabled")
	}

	// Validate metrics URL and set state
	serviceEndpoint := vip.GetString(ServiceEndpointKey)
	daemonEndpoint := vip.GetString(DaemonEndpoint)
	err := ValidateEndpoints(daemonEndpoint, serviceEndpoint)
	if err != nil {
		return err
	}

	// Check if the Daemon is on the latest version or not
	if message, err := CheckVersionOfDaemon(); err != nil {
		// In case of any error on version check, just log it
		zap.L().Warn(err.Error())
	} else {
		// Print current version of daemon
		zap.L().Info(message)
	}

	// Check the maximum message size (The maximum that the server can receive - 2GB).
	maxMessageSize := vip.GetInt(MaxMessageSizeInMB)
	if maxMessageSize <= 0 || maxMessageSize > 2048 {
		return errors.New(" max_message_size_in_mb cannot be more than 2GB (i.e 2048 MB) and has to be a positive number")
	}
	if err = allowedUserConfigurationChecks(); err != nil {
		return err
	}

	if GetString(PvtKeyForFreeCalls) != "" {
		if utils.ParsePrivateKey(GetString(PvtKeyForFreeCalls)) == nil {
			return errors.New("invalid " + PvtKeyForFreeCalls)
		}
	}

	return validateMeteringChecks()
}

func GetTrustedFreeCallSignersAddresses() []common.Address {
	var addrs []common.Address

	slice := vip.GetStringSlice(TrustedFreeCallSigners)
	if len(slice) > 0 {
		for _, addr := range slice {
			if common.IsHexAddress(addr) {
				addrs = append(addrs, common.HexToAddress(addr))
			}
		}
		return addrs
	}

	addr := vip.GetString(TrustedFreeCallSigners)
	if common.IsHexAddress(addr) {
		return []common.Address{common.HexToAddress(addr)}
	}

	return addrs
}

// allowedUserConfigurationChecks restrict access to only certain users, this feature is useful when you are
// in a test environment and don't want everyone to make requests to your service.
// Since this was flag was introduced to restrict users while in testing mode, we don't want this configuration
// to be mistakenly set on the mainnet
func allowedUserConfigurationChecks() error {
	if GetBool(AllowedUserFlag) {
		if GetString(BlockChainNetworkSelected) == "main" {
			return fmt.Errorf("service cannot be restricted to certain users when set up against Ethereum mainnet,the flag %v is set to true", AllowedUserFlag)
		}
		if err := SetAllowedUsers(); err != nil {
			return err
		}
	}
	return nil
}

func validateMeteringChecks() (err error) {
	if GetBool(MeteringEnabled) && !IsValidUrl(GetString(MeteringEndpoint)) {
		return errors.New("to Support Metering you need to have a valid Metering End point")
	}
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

func GetStringMap(key string) map[string]any {
	return vip.GetStringMap(key)
}

func normalizeMapKeysToLower(m map[string]any) map[string]any {
	normalized := make(map[string]any, len(m))
	for k, v := range m {
		normalized[strings.ToLower(k)] = v
	}
	return normalized
}

// GetFreeCallsAllowed returns the number of free calls allowed for the given user address.
//
// It looks up the address in the free_calls_per_address configuration:
//   - int: returned as-is.
//   - float64: converted to int.
//   - string "unlimited"/"infinity": returns -1 (unlimited).
//
// If no entry is found or the type is invalid, it returns 0.
func GetFreeCallsAllowed(userAddr string) int {
	freeCallsUsers := normalizeMapKeysToLower(GetStringMap(FreeCallsPerAddress))
	if countFreeCalls, ok := freeCallsUsers[strings.ToLower(userAddr)]; ok {
		switch countCasted := countFreeCalls.(type) {
		case int:
			return countCasted
		case float64:
			return int(countCasted)
		case string:
			if countCasted == "unlimited" || countCasted == "infinity" {
				return -1
			}
		default:
			zap.L().Error("Invalid free_calls_per_address param: ", zap.Any("invalid value", countFreeCalls))
			return 0
		}
	}
	return 0
}

func GetStringSlice(key string) []string {
	return vip.GetStringSlice(key)
}

func Get(key string) any {
	return vip.Get(key)
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

var DisplayKeys = map[string]bool{
	strings.ToUpper(AllowedUserFlag):                true,
	strings.ToUpper(AllowedUserAddresses):           true,
	strings.ToUpper(AuthenticationAddresses):        true,
	strings.ToUpper(AutoSSLDomainKey):               true,
	strings.ToUpper(AutoSSLCacheDirKey):             true,
	strings.ToUpper(BlockchainEnabledKey):           true,
	strings.ToUpper(BlockChainNetworkSelected):      true,
	strings.ToUpper(BurstSize):                      true,
	strings.ToUpper(ConfigPathKey):                  true,
	strings.ToUpper(DaemonGroupName):                true,
	strings.ToUpper(DaemonTypeKey):                  true,
	strings.ToUpper(DaemonEndpoint):                 true,
	strings.ToUpper(ExecutablePathKey):              true,
	strings.ToUpper(IpfsEndpoint):                   true,
	strings.ToUpper(LighthouseEndpoint):             true,
	strings.ToUpper(IpfsTimeout):                    false,
	strings.ToUpper(LogKey):                         true,
	strings.ToUpper(MaxMessageSizeInMB):             true,
	strings.ToUpper(OrganizationId):                 true,
	strings.ToUpper(ServiceId):                      true,
	strings.ToUpper(PassthroughEnabledKey):          true,
	strings.ToUpper(ServiceEndpointKey):             true,
	strings.ToUpper(RateLimitPerMinute):             true,
	strings.ToUpper(SSLCertPathKey):                 true,
	strings.ToUpper(SSLKeyPathKey):                  true,
	strings.ToUpper(PaymentChannelCertPath):         true,
	strings.ToUpper(PaymentChannelCaPath):           true,
	strings.ToUpper(PaymentChannelKeyPath):          true,
	strings.ToUpper(PaymentChannelStorageTypeKey):   true,
	strings.ToUpper(PaymentChannelStorageClientKey): true,
	strings.ToUpper(PaymentChannelStorageServerKey): true,
	strings.ToUpper(AlertsEMail):                    true,
	strings.ToUpper(HeartbeatServiceEndpoint):       true,
	strings.ToUpper(MeteringEnabled):                true,
	strings.ToUpper(MeteringEndpoint):               true,
	strings.ToUpper(NotificationServiceEndpoint):    true,
	strings.ToUpper(ServiceHeartbeatType):           true,
}

func LogConfig() {
	zap.L().Info("Final configuration: ")
	keys := vip.AllKeys()
	sort.Strings(keys)
	for _, key := range keys {
		if DisplayKeys[strings.ToUpper(key)] {
			if v, ok := vip.Get(key).(string); ok && v == "" {
				continue
			}
			zap.L().Info(key, zap.Any("value", vip.Get(key)))
		}
	}
}

func GetBigIntFromViper(config *viper.Viper, key string) (value *big.Int, err error) {
	value = &big.Int{}
	err = value.UnmarshalText([]byte(config.GetString(key)))
	return
}

// IsValidUrl tests a string to determine if it is url or not.
func IsValidUrl(urlToTest string) bool {
	_, err := url.ParseRequestURI(urlToTest)
	return err == nil
}

// ValidateEmail validates an input email
func ValidateEmail(email string) bool {
	Re := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)
	return Re.MatchString(email)
}

// ValidateEndpoints checks that the daemon endpoint and the service endpoint
// are valid and not pointing to the same address.
//
// It ensures:
//   - serviceEndpoint has a URL scheme (adds "http://" if missing).
//   - serviceEndpoint is a valid URL with a non-empty host.
//   - daemonEndpoint can be split into host and port.
//   - daemonEndpoint and serviceEndpoint do not have the same host and port.
//   - Special case: if the daemon host is "0.0.0.0" and the service host is
//     "127.0.0.1" or "localhost" with the same port, it is also considered invalid.
//
// Returns an error if validation fails, or nil if endpoints are valid.
func ValidateEndpoints(daemonEndpoint string, serviceEndpoint string) error {

	if !strings.Contains(serviceEndpoint, "://") {
		serviceEndpoint = "http" + "://" + serviceEndpoint
	}

	serviceURL, err := url.Parse(serviceEndpoint)
	if err != nil || serviceURL.Host == "" {
		return errors.New("service_endpoint is the endpoint of your AI service in the daemon config and needs to be a valid url")
	}

	daemonHost, daemonPort, err := net.SplitHostPort(daemonEndpoint)
	if err != nil {
		return errors.New("couldn't split host:post of daemon endpoint")
	}

	if daemonHost == serviceURL.Hostname() && daemonPort == serviceURL.Port() {
		return errors.New("service_endpoint can't be the same as daemon endpoint")
	}

	if (daemonPort == serviceURL.Port()) &&
		(daemonHost == "0.0.0.0") &&
		(serviceURL.Hostname() == "127.0.0.1" || serviceURL.Hostname() == "localhost") {
		return errors.New("service_endpoint can't be the same as daemon endpoint")
	}
	return nil
}

var userAddress []common.Address

func IsAllowedUser(address *common.Address) bool {
	for _, user := range userAddress {
		zap.L().Info("user address from config", zap.String("value", user.Hex()+"<>"+address.Hex()))
		if user == *address {
			return true
		}
	}
	return false
}

// SetAllowedUsers sets the list of allowed users
func SetAllowedUsers() (err error) {
	users := vip.GetStringSlice(AllowedUserAddresses)
	if len(users) == 0 {
		return fmt.Errorf("a valid Address needs to be specified for the config %v to ensure that, only these users can make calls", AllowedUserAddresses)
	}
	userAddress = make([]common.Address, 0, len(users))
	for _, user := range users {
		if !common.IsHexAddress(user) {
			err = fmt.Errorf("%v is not a valid hex address", user)
			return err
		} else {
			userAddress = append(userAddress, common.BytesToAddress(common.FromHex(user)))
		}
	}
	return nil
}

// NewJsonConfigFromString for tests
func NewJsonConfigFromString(config string) *viper.Viper {
	v := viper.New()
	v.SetConfigType("json")
	err := v.ReadConfig(bytes.NewBufferString(config))
	if err != nil {
		zap.L().Error("Error reading string config", zap.Error(err))
	}
	return v
}
