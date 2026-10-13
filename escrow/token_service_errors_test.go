package escrow

import (
	"errors"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTokenServicePaymentUpdateFailures(t *testing.T) {
	for _, failure := range []string{"start transaction", "commit", "usage update"} {
		t.Run(failure, func(t *testing.T) {
			f := newTokenServiceFixture(t)
			wantErr := errors.New(failure + " failed")
			f.channelMock.paymentChannelFn = func(*PaymentChannelKey) (*PaymentChannelData, bool, error) {
				return f.channel(), true, nil
			}
			f.channelMock.startPaymentTransactionFn = func(payment *Payment) (PaymentTransaction, error) {
				require.Equal(t, big.NewInt(11), payment.Amount)
				if failure == "start transaction" {
					return nil, wantErr
				}
				transaction := &paymentTransactionMock{}
				if failure == "commit" {
					transaction.err = wantErr
				}
				return transaction, nil
			}
			if failure == "usage update" {
				f.usageMock.updateErr = wantErr
			}
			request := &TokenRequest{ChannelId: 1, SignedAmount: 11, CurrentBlock: 99}
			f.signRequest(request, f.senderKey)
			reply, err := f.service.GetToken(t.Context(), request)
			require.Nil(t, reply)
			require.ErrorIs(t, err, wantErr)
			if failure != "usage update" {
				require.Empty(t, f.usageMock.updatedType, "failed payment must not update usage")
			}
		})
	}
}

type usageReader struct {
	PrePaidService
	get func(PrePaidDataKey) (*PrePaidData, bool, error)
}

func (s usageReader) GetUsage(key PrePaidDataKey) (*PrePaidData, bool, error) {
	return s.get(key)
}

func TestTokenServiceUsageReadFailures(t *testing.T) {
	for _, usageType := range []string{USED_AMOUNT, PLANNED_AMOUNT} {
		t.Run(usageType, func(t *testing.T) {
			f := newTokenServiceFixture(t)
			f.channelMock.paymentChannelFn = func(*PaymentChannelKey) (*PaymentChannelData, bool, error) {
				return f.channel(), true, nil
			}
			wantErr := errors.New("usage read failed")
			f.service.prePaidUsageService = usageReader{get: func(key PrePaidDataKey) (*PrePaidData, bool, error) {
				if key.UsageType == usageType {
					return &PrePaidData{Amount: big.NewInt(0)}, true, wantErr
				}
				return nil, false, nil
			}}
			request := &TokenRequest{ChannelId: 1, SignedAmount: 10, CurrentBlock: 99}
			f.signRequest(request, f.senderKey)
			reply, err := f.service.GetToken(t.Context(), request)
			require.Nil(t, reply)
			require.ErrorIs(t, err, wantErr)
		})
	}
}

func TestTokenServiceBlockLookupFailure(t *testing.T) {
	f := newTokenServiceFixture(t)
	wantErr := errors.New("blockchain unavailable")
	f.service.validator.currentBlock = func() (*big.Int, error) { return nil, wantErr }
	require.ErrorIs(t, f.service.allowedBlockNumberCheck(big.NewInt(99)), wantErr)
}
