package errs

import (
	"fmt"
)

const devPortalURL = "https://dev.singularitynet.io/docs/products/DecentralizedAIPlatform/Daemon/error-codes/#_"

const (
	_ = iota
	ServiceUnavailable
	InvalidMetadata
	InvalidProto
	HTTPRequestBuildError
	InvalidServiceCredentials
	InvalidConfig
	ReceiveMsgError
	BlockchainProviderLimitsExceed
)

func ErrDescURL(code int) string {
	return fmt.Sprintf("\nAbout error & possible fixes: %s%d", devPortalURL, code)
}
