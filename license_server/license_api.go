package license_server

import (
	"math/big"
)

type LicenseService interface {
	GetLicenseUsage(key LicenseUsageTrackerKey) (*LicenseUsageTrackerData, bool, error)
	UpdateLicenseUsage(channelId *big.Int, serviceId string, revisedUsage *big.Int, updateUsageType string, licenseType string) error
	GetLicenseForChannel(key LicenseDetailsKey) (*LicenseDetailsData, bool, error)
	UpdateLicenseForChannel(channelId *big.Int, serviceId string, license License) error
}

type LicenseFilterCriteria struct {
}

type LicenseTransaction interface {
	ServiceId() string
	ChannelId() *big.Int
	Usage() *big.Int
	Commit() error
	Rollback() error
}

// this is used to track the license Usage
type licenseUsageTrackerTransactionImpl struct {
	channelId *big.Int
	usage     Usage
	serviceId string
}

func (transaction licenseUsageTrackerTransactionImpl) ServiceId() string {
	return transaction.serviceId
}
func (transaction licenseUsageTrackerTransactionImpl) ChannelId() *big.Int {
	return transaction.channelId
}
func (transaction licenseUsageTrackerTransactionImpl) Usage() Usage {
	return transaction.usage
}
func (transaction licenseUsageTrackerTransactionImpl) Commit() error {
	return nil
}
func (transaction licenseUsageTrackerTransactionImpl) Rollback() error {
	return nil
}
