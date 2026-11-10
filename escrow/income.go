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

// IncomeStreamData supplies the payment increment and stream context used to
// verify that a call's price is covered by the new authorization.
type IncomeStreamData struct {
	// Income is the new cumulative signed amount minus the previously accepted
	// AuthorizedAmount for the same nonce, in token base units.
	Income *big.Int
	// GrpcContext provides the RPC method and metadata used to determine price.
	GrpcContext *handler.GrpcStreamContext
}

// IncomeStreamValidator checks a stream call's payment increment against its
// pricing policy before the payment transaction is committed.
type IncomeStreamValidator interface {
	// Validate returns nil on success. Errors can include PaymentError for a
	// price mismatch or errors from price/model lookup.
	Validate(*IncomeStreamData) (err error)
}

type incomeStreamValidator struct {
	priceStrategy *pricing.PricingStrategy
	storage       *training.ModelStorage
}

// NewIncomeStreamValidator uses model ValidatePrice for upload_and_validate
// methods and the supplied pricing strategy for other stream methods.
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

// IncomeUnaryValidator checks the payment increment for a unary training call.
type IncomeUnaryValidator interface {
	// Validate returns nil on success or an error from pricing validation or
	// model lookup. A price mismatch is reported as PaymentError.
	Validate(data *IncomeUnaryData) (err error)
}

// IncomeUnaryData supplies a unary call's payment increment and pricing context.
type IncomeUnaryData struct {
	// Income is the new cumulative signed amount minus the previously accepted
	// AuthorizedAmount for the same nonce, in token base units.
	Income *big.Int
	// GrpcContext identifies the unary method and training model in metadata.
	GrpcContext *handler.GrpcUnaryContext
}

// NewTrainValidator checks train_model and validate_model against their stored
// model prices. Other methods skip the price comparison after model lookup.
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
