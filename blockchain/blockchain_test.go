package blockchain

import (
	"math/big"
	"testing"

	"github.com/singnet/snet-daemon/v6/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const metadataJson = "{\n    \"version\": 1,\n    \"display_name\": \"semyon_dev\",\n    \"encoding\": \"proto\",\n    \"service_type\": \"grpc\",\n    \"service_api_source\": \"ipfs://QmV9bBsLAZfXGibdU3isPwDn8SxAPkqg6YzcKamSCxnBCR\",\n    \"mpe_address\": \"0x7E0aF8988DF45B824b2E0e0A87c6196897744970\",\n    \"groups\": [\n        {\n            \"group_name\": \"default_group\",\n            \"endpoints\": [\n                \"http://localhost:7000\"\n            ],\n            \"pricing\": [\n                {\n                    \"price_model\": \"fixed_price\",\n                    \"price_in_cogs\": 1,\n                    \"default\": true\n                }\n            ],\n            \"group_id\": \"FtNuizEOUsVCd5f2Fij9soehtRSb58LlTePgkVnsgVI=\",\n            \"free_call_signer_address\": \"0x747155e03c892B8b311B7Cfbb920664E8c6792fA\",\n            \"free_calls\": 25,\n            \"daemon_addresses\": [\n                \"0x747155e03c892B8b311B7Cfbb920664E8c6792fA\"\n            ]\n        },\n        {\n            \"group_name\": \"not_default\",\n            \"endpoints\": [\n                \"http://localhost:7000\"\n            ],\n            \"pricing\": [\n                {\n                    \"price_model\": \"fixed_price\",\n                    \"price_in_cogs\": 1,\n                    \"default\": true\n                }\n            ],\n            \"group_id\": \"udN0SLIvsDdvQQe3Ltv/NwqCh7sPKdz4scYmlI7AMdE=\",\n            \"free_call_signer_address\": \"0x747155e03c892B8b311B7Cfbb920664E8c6792fA\",\n            \"free_calls\": 35,\n            \"daemon_addresses\": [\n                \"0x747155e03c892B8b311B7Cfbb920664E8c6792fA\"\n            ]\n        }\n    ],\n    \"assets\": {},\n    \"media\": [],\n    \"tags\": [],\n    \"service_description\": {\n        \"description\": \"Test service with localhost endpoint!\",\n        \"url\": \"\"\n    }\n}"

// ProcessorTestSuite is a test suite for the processor struct
type ProcessorTestSuite struct {
	suite.Suite
	processor Processor
}

// SetupSuite initializes the Ethereum client before running the tests
func (suite *ProcessorTestSuite) SetupSuite() {
	config.Vip().Set(config.BlockchainEnabledKey, true)
	config.Vip().Set(config.BlockChainNetworkSelected, "sepolia")
	config.Validate()
	_, err := InitServiceMetaDataFromJson([]byte(metadataJson))
	assert.Nil(suite.T(), err)
	suite.processor = NewMockProcessor(true)
}

// Test: If the block number difference is within the allowed limit → no error
func (suite *ProcessorTestSuite) TestCompareWithLatestBlockNumber_WithinLimit() {
	latestBlock, err := suite.processor.CurrentBlock()
	suite.Require().NoError(err, "CurrentBlock() should not return an error")

	// Simulate a block number within the allowed range (+2)
	blockNumberPassed := new(big.Int).Add(latestBlock, big.NewInt(2))
	err = suite.processor.CompareWithLatestBlockNumber(blockNumberPassed, 5)

	// Expect no error
	assert.NoError(suite.T(), err, "Expected no error when block difference is within the limit")
}

// Test: If the block number difference exceeds the allowed limit → return an error
func (suite *ProcessorTestSuite) TestCompareWithLatestBlockNumber_ExceedsLimit() {
	latestBlock, err := suite.processor.CurrentBlock()
	suite.Require().NoError(err, "CurrentBlock() should not return an error")

	// Simulate a block number exceeding the allowed limit (+10)
	blockNumberPassed := new(big.Int).Add(latestBlock, big.NewInt(10))
	err = suite.processor.CompareWithLatestBlockNumber(blockNumberPassed, 5)

	// Expect an error
	assert.Error(suite.T(), err, "Expected an error when block difference exceeds the limit")
	assert.Contains(suite.T(), err.Error(), "authentication failed", "Error message should indicate signature expiration")
}

// Run the test suite
func TestProcessorTestSuite(t *testing.T) {
	suite.Run(t, new(ProcessorTestSuite))
}
