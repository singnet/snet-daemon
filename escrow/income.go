package escrow

import (
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/singnet/snet-daemon/v6/pricing"
	"github.com/singnet/snet-daemon/v6/training"
	"go.uber.org/zap"

	"github.com/singnet/snet-daemon/v6/handler"
)

// IncomeStreamData is used to pass information to the pricing validation system.
// This system can use information about call to calculate price and verify
// income received.
type IncomeStreamData struct {
	// Income is a difference between previous authorized amount and amount
	// which was received with the current call.
	Income *big.Int
	// GrpcContext contains gRPC stream context information. For instance,
	// metadata could be used to pass invoice id to check pricing.
	GrpcContext *handler.GrpcStreamContext
}

// IncomeStreamValidator uses pricing information to check that call was paid
// correctly by channel sender. This interface can be implemented differently
// depending on pricing policy. For instance one can verify that call is paid
// according to invoice. Each RPC method can have different price and so on. To
// implement these strategies additional information from gRPC context can be
// required. In such case it should be added into handler.GrpcStreamContext.
type IncomeStreamValidator interface {
	// Validate returns nil if validation is successful or correct PaymentError
	// status to be sent to client in case of validation error.
	Validate(*IncomeStreamData) (err error)
}

type incomeStreamValidator struct {
	priceStrategy *pricing.PricingStrategy
	storage       *training.ModelStorage
}

// NewIncomeStreamValidator returns new income validator instance
func NewIncomeStreamValidator(pricing *pricing.PricingStrategy, storage *training.ModelStorage) (validator IncomeStreamValidator) {
	return &incomeStreamValidator{priceStrategy: pricing, storage: storage}
}

func (validator *incomeStreamValidator) Validate(data *IncomeStreamData) (err error) {

	price := big.NewInt(0)

	if data.GrpcContext != nil && strings.Contains(data.GrpcContext.Info.FullMethod, "/upload_and_validate") {
		modelID, ok := data.GrpcContext.MD[handler.TrainingModelId]
		if !ok {
			return errors.New("no training model found")
		}

		model, err := validator.storage.GetModel(modelID[0])
		if err != nil {
			return errors.New("no training model found")
		}

		price = price.SetUint64(model.ValidatePrice)
	} else {
		price, err = validator.priceStrategy.GetPrice(data.GrpcContext)
		if err != nil {
			return err
		}
	}

	if data.Income.Cmp(price) != 0 {
		err = NewPaymentError(Unauthenticated, "income %d does not equal to price %d", data.Income, price)
		return
	}

	return
}

type trainUnaryValidator struct {
	priceStrategy *pricing.PricingStrategy
	storage       *training.ModelStorage
}

type IncomeUnaryValidator interface {
	// Validate returns nil if validation is successful or correct PaymentError
	// status to be sent to client in case of validation error.
	Validate(data *IncomeUnaryData) (err error)
}

type IncomeUnaryData struct {
	// Income is a difference between previous authorized amount and amount
	// which was received with current call.
	Income *big.Int
	// GrpcContext contains gRPC stream context information. For instance
	// metadata could be used to pass invoice id to check pricing.
	GrpcContext *handler.GrpcUnaryContext
}

// NewTrainValidator returns a new income validator instance
func NewTrainValidator(storage *training.ModelStorage) (validator IncomeUnaryValidator) {
	return &trainUnaryValidator{storage: storage}
}

func (validator *trainUnaryValidator) Validate(data *IncomeUnaryData) (err error) {
	modelID, ok := data.GrpcContext.MD[handler.TrainingModelId]
	if !ok {
		return errors.New("[trainUnaryValidator] no training model found")
	}

	model, err := validator.storage.GetModel(modelID[0])
	if err != nil {
		return errors.New("[trainUnaryValidator] no training model found")
	}

	price := big.NewInt(0)

	lastSlash := strings.LastIndex(data.GrpcContext.Info.FullMethod, "/")
	methodName := data.GrpcContext.Info.FullMethod[lastSlash+1:]

	switch methodName {
	case "train_model":
		price = price.SetUint64(model.TrainPrice)
	case "validate_model":
		price = price.SetUint64(model.ValidatePrice)
	default:
		return nil
	}

	zap.L().Debug("[Validate]", zap.Uint64("price", price.Uint64()))

	if data.Income.Cmp(price) != 0 {
		zap.L().Error(fmt.Sprintf("[Validate] income %d does not equal to price %d", data.Income, price))
		err = NewPaymentError(Unauthenticated, "income %d does not equal to price %d", data.Income, price)
		return
	}

	return
}
