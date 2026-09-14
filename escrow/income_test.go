package escrow

import (
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/handler"
	"github.com/singnet/snet-daemon/v6/pricing"
	"github.com/singnet/snet-daemon/v6/storage"
	"github.com/singnet/snet-daemon/v6/training"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type incomeValidatorMockType struct {
	err error
}

func (incomeValidator *incomeValidatorMockType) Validate(income *IncomeStreamData) (err error) {
	return incomeValidator.err
}

type MockPriceType struct {
}

func (priceType MockPriceType) GetPrice(GrpcContext *handler.GrpcStreamContext) (price *big.Int, err error) {
	return big.NewInt(0), nil
}
func (priceType MockPriceType) GetPriceType() string {
	return pricing.FIXED_PRICING
}

var testJsonDataFixedPrice = "{   \"version\": 1,   \"display_name\": \"Example1\",   \"encoding\": \"grpc\",   \"service_type\": \"grpc\",   \"payment_expiration_threshold\": 40320,   \"model_ipfs_hash\": \"Qmdiq8Hu6dYiwp712GtnbBxagyfYyvUY1HYqkH7iN76UCc\",   \"mpe_address\": \"0x7E6366Fbe3bdfCE3C906667911FC5237Cc96BD08\",   \"groups\": [     {       \"endpoints\": [\"http://34.344.33.1:2379\",\"http://34.344.33.1:2389\"],       \"group_id\": \"88ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",\"group_name\": \"default_group\",       \"pricing\": [         {           \"price_model\": \"fixed_price\",    \"default\":true,         \"price_in_cogs\": 2         },          {         \"package_name\": \"example_service\",         \"price_model\": \"fixed_price_per_method\",                \"details\": [           {             \"service_name\": \"Calculator\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 3               }             ]           },           {             \"service_name\": \"Calculator2\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 3               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 2               }             ]           }         ]       }]     },     {       \"endpoints\": [\"http://97.344.33.1:2379\",\"http://67.344.33.1:2389\"],    \"group_name\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",   \"group_id\": \"99ybRIg2wAx55mqVsA6sB4S7WxPQHNKqa4BPu/bhj+U=\",       \"pricing\": [         {         \"package_name\": \"example_service\",         \"price_model\": \"fixed_price_per_method\",         \"details\": [           {             \"service_name\": \"Calculator\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 3               }             ]           },           {             \"service_name\": \"Calculator2\",             \"method_pricing\": [               {                 \"method_name\": \"add\",                 \"price_in_cogs\": 2               },               {                 \"method_name\": \"sub\",                 \"price_in_cogs\": 1               },               {                 \"method_name\": \"div\",                 \"price_in_cogs\": 3               },               {                 \"method_name\": \"mul\",                 \"price_in_cogs\": 2               }             ]           }         ]       }]     }   ] } "

var pricingMetadata, _ = blockchain.InitServiceMetaDataFromJson([]byte(testJsonDataFixedPrice))

func TestIncomeValidate(t *testing.T) {
	one := big.NewInt(1)
	income := big.NewInt(0)
	pricingStrt, err := pricing.InitPricingStrategy(pricingMetadata)
	assert.Nil(t, err)
	pricingStrt.AddPricingTypes(&MockPriceType{})
	incomeValidator := NewIncomeStreamValidator(pricingStrt, nil)
	price := big.NewInt(0)

	income.Sub(price, one)
	err = incomeValidator.Validate(&IncomeStreamData{Income: income, GrpcContext: &handler.GrpcStreamContext{Info: &grpc.StreamServerInfo{FullMethod: "test"}}})
	assert.Equal(t, NewPaymentError(Unauthenticated, "income %s does not equal to price %s", income, price), err)

	income.Set(price)
	err = incomeValidator.Validate(&IncomeStreamData{Income: income, GrpcContext: &handler.GrpcStreamContext{Info: &grpc.StreamServerInfo{FullMethod: "test"}}})
	assert.Nil(t, err)

	income.Add(price, one)
	err = incomeValidator.Validate(&IncomeStreamData{Income: income, GrpcContext: &handler.GrpcStreamContext{Info: &grpc.StreamServerInfo{FullMethod: "test"}}})
	assert.Equal(t, NewPaymentError(Unauthenticated, "income %s does not equal to price %s", income, price), err)
}

type MockPriceErrorType struct {
}

func (priceType MockPriceErrorType) GetPrice(GrpcContext *handler.GrpcStreamContext) (price *big.Int, err error) {
	return nil, fmt.Errorf("Error in Determining Price")
}

func (priceType MockPriceErrorType) GetPriceType() string {
	return pricing.FIXED_PRICING
}
func TestIncomeValidateForPriceError(t *testing.T) {
	pricingStrt, err := pricing.InitPricingStrategy(pricingMetadata)
	assert.Nil(t, err)
	pricingStrt.AddPricingTypes(&MockPriceErrorType{})
	incomeValidator := NewIncomeStreamValidator(pricingStrt, nil)
	err = incomeValidator.Validate(&IncomeStreamData{Income: big.NewInt(0), GrpcContext: &handler.GrpcStreamContext{Info: &grpc.StreamServerInfo{FullMethod: "test"}}})
	assert.Equal(t, err.Error(), "Error in Determining Price")
}

const testModelID = "model-1"

func initIncomeTestOrgMetadata(t *testing.T) *blockchain.OrganizationMetaData {
	t.Helper()
	orgMetadata, err := blockchain.InitOrganizationMetaDataFromJson([]byte(testJsonOrgGroupData))
	require.NoError(t, err)
	config.Vip().Set(config.OrganizationId, orgMetadata.OrgID)
	config.Vip().Set(config.ServiceId, "test-service")
	config.Vip().Set(config.DaemonGroupName, "default_group")
	return orgMetadata
}

func newTestModelStorage(t *testing.T, atomic storage.AtomicStorage, orgMetadata *blockchain.OrganizationMetaData,
	validatePrice, trainPrice uint64) *training.ModelStorage {
	t.Helper()
	modelStorage := training.NewModelStorage(atomic, orgMetadata)
	err := modelStorage.Put(&training.ModelKey{
		OrganizationId: orgMetadata.OrgID,
		ServiceId:      "test-service",
		GroupId:        orgMetadata.GetGroupIdString(),
		ModelId:        testModelID,
	}, &training.ModelData{
		ModelId:       testModelID,
		ValidatePrice: validatePrice,
		TrainPrice:    trainPrice,
	})
	require.NoError(t, err)
	return modelStorage
}

func trainingStreamContext(md metadata.MD) *handler.GrpcStreamContext {
	return &handler.GrpcStreamContext{
		Info: &grpc.StreamServerInfo{FullMethod: "/snet.training/upload_and_validate"},
		MD:   md,
	}
}

func modelMD() metadata.MD {
	return metadata.New(map[string]string{handler.TrainingModelId: testModelID})
}

func TestIncomeStreamValidatorTraining(t *testing.T) {
	orgMetadata := initIncomeTestOrgMetadata(t)
	validatePrice := uint64(10)

	t.Run("missing model id", func(t *testing.T) {
		validator := NewIncomeStreamValidator(nil, newTestModelStorage(t, storage.NewMemStorage(), orgMetadata, validatePrice, 20))
		err := validator.Validate(&IncomeStreamData{Income: big.NewInt(0), GrpcContext: trainingStreamContext(metadata.New(nil))})
		assert.EqualError(t, err, "no training model found")
	})

	t.Run("get model error", func(t *testing.T) {
		broken := &failingStorage{MemoryStorage: storage.NewMemStorage(), getErr: errors.New("storage boom")}
		validator := NewIncomeStreamValidator(nil, newTestModelStorage(t, broken, orgMetadata, validatePrice, 20))
		err := validator.Validate(&IncomeStreamData{Income: big.NewInt(0), GrpcContext: trainingStreamContext(modelMD())})
		assert.EqualError(t, err, "no training model found")
	})

	t.Run("income matches validate price", func(t *testing.T) {
		validator := NewIncomeStreamValidator(nil, newTestModelStorage(t, storage.NewMemStorage(), orgMetadata, validatePrice, 20))
		err := validator.Validate(&IncomeStreamData{Income: big.NewInt(int64(validatePrice)), GrpcContext: trainingStreamContext(modelMD())})
		assert.NoError(t, err)
	})

	t.Run("income does not match validate price", func(t *testing.T) {
		validator := NewIncomeStreamValidator(nil, newTestModelStorage(t, storage.NewMemStorage(), orgMetadata, validatePrice, 20))
		err := validator.Validate(&IncomeStreamData{Income: big.NewInt(5), GrpcContext: trainingStreamContext(modelMD())})
		require.Error(t, err)
		assert.Equal(t, Unauthenticated, err.(*PaymentError).Code)
	})
}

func TestTrainUnaryValidator(t *testing.T) {
	orgMetadata := initIncomeTestOrgMetadata(t)
	trainPrice := uint64(20)
	validatePrice := uint64(10)

	newValidator := func(t *testing.T, atomic storage.AtomicStorage) IncomeUnaryValidator {
		return NewTrainValidator(newTestModelStorage(t, atomic, orgMetadata, validatePrice, trainPrice))
	}

	unaryContext := func(fullMethod string, md metadata.MD) *handler.GrpcUnaryContext {
		return &handler.GrpcUnaryContext{Info: &grpc.UnaryServerInfo{FullMethod: fullMethod}, MD: md}
	}

	t.Run("missing model id", func(t *testing.T) {
		validator := newValidator(t, storage.NewMemStorage())
		err := validator.Validate(&IncomeUnaryData{Income: big.NewInt(0), GrpcContext: unaryContext("/svc/train_model", metadata.New(nil))})
		assert.EqualError(t, err, "[trainUnaryValidator] no training model found")
	})

	t.Run("get model error", func(t *testing.T) {
		broken := &failingStorage{MemoryStorage: storage.NewMemStorage(), getErr: errors.New("storage boom")}
		validator := newValidator(t, broken)
		err := validator.Validate(&IncomeUnaryData{Income: big.NewInt(0), GrpcContext: unaryContext("/svc/train_model", modelMD())})
		assert.EqualError(t, err, "[trainUnaryValidator] no training model found")
	})

	t.Run("train_model income matches train price", func(t *testing.T) {
		validator := newValidator(t, storage.NewMemStorage())
		err := validator.Validate(&IncomeUnaryData{Income: big.NewInt(int64(trainPrice)), GrpcContext: unaryContext("/svc/train_model", modelMD())})
		assert.NoError(t, err)
	})

	t.Run("train_model income does not match train price", func(t *testing.T) {
		validator := newValidator(t, storage.NewMemStorage())
		err := validator.Validate(&IncomeUnaryData{Income: big.NewInt(1), GrpcContext: unaryContext("/svc/train_model", modelMD())})
		require.Error(t, err)
		assert.Equal(t, Unauthenticated, err.(*PaymentError).Code)
	})

	t.Run("validate_model income matches validate price", func(t *testing.T) {
		validator := newValidator(t, storage.NewMemStorage())
		err := validator.Validate(&IncomeUnaryData{Income: big.NewInt(int64(validatePrice)), GrpcContext: unaryContext("/svc/validate_model", modelMD())})
		assert.NoError(t, err)
	})

	t.Run("validate_model income does not match validate price", func(t *testing.T) {
		validator := newValidator(t, storage.NewMemStorage())
		err := validator.Validate(&IncomeUnaryData{Income: big.NewInt(1), GrpcContext: unaryContext("/svc/validate_model", modelMD())})
		require.Error(t, err)
		assert.Equal(t, Unauthenticated, err.(*PaymentError).Code)
	})

	t.Run("unknown method returns nil", func(t *testing.T) {
		validator := newValidator(t, storage.NewMemStorage())
		err := validator.Validate(&IncomeUnaryData{Income: big.NewInt(0), GrpcContext: unaryContext("/svc/other_method", modelMD())})
		assert.NoError(t, err)
	})
}
