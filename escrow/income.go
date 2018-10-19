package escrow

import (
	"github.com/singnet/snet-daemon/blockchain"
	"google.golang.org/grpc/status"
)

type incomeValidator struct {
	agent *blockchain.Agent
}

func NewIncomeValidator(processor *blockchain.Processor) (validator IncomeValidator) {
	return &incomeValidator{
		agent: processor.Agent(),
	}
}

func (validator *incomeValidator) Validate(*IncomeData) (err *status.Status) {
	// TODO: implement
	return nil
}
