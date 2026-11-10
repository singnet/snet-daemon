package escrow

import (
	"bytes"
	"errors"
	"math/big"
	"testing"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/stretchr/testify/require"
)

func TestControlListAndBatchClaimRejections(t *testing.T) {
	for _, operation := range []string{"list in progress", "batch claim"} {
		t.Run(operation, func(t *testing.T) {
			for _, failure := range []string{"MPE address", "stale block", "signature", "cleanup"} {
				t.Run(failure, func(t *testing.T) {
					isolateConfig(t)
					f := newControlServiceFixture(t)
					mpeAddress := f.svcMD.MpeAddress
					block := uint64(blockchain.MockedCurrentBlock)
					wantErr := "signature"
					switch failure {
					case "MPE address":
						mpeAddress = "invalid"
						wantErr = "mpeAddress"
					case "stale block":
						block -= 6
						wantErr = "signature passed has expired"
					case "cleanup":
						f.channelMock.listClaimsFn = func() ([]Claim, error) { return nil, errors.New("storage failed") }
						wantErr = "error in retrieving claims"
					}
					f.channelMock.startClaimFn = func(*PaymentChannelKey, ChannelUpdate) (Claim, error) {
						t.Fatal("rejected request must not start a claim")
						return nil, nil
					}
					var err error
					if operation == "list in progress" {
						request := &GetPaymentsListRequest{MpeAddress: mpeAddress, CurrentBlock: block}
						if failure != "signature" {
							request.Signature = f.signListRequest("__list_in_progress", block)
						}
						_, err = f.service.GetListInProgress(t.Context(), request)
					} else {
						request := &StartMultipleClaimRequest{MpeAddress: mpeAddress, CurrentBlock: block, ChannelIds: []uint64{1}}
						if failure != "signature" {
							message := bytes.Join([][]byte{
								[]byte("__StartClaimForMultipleChannels_"), f.svcMD.GetMpeAddress().Bytes(),
								bigIntToBytes(big.NewInt(1)), bigIntToBytes(new(big.Int).SetUint64(block)),
							}, nil)
							request.Signature = getSignature(message, f.signerKey)
						}
						_, err = f.service.StartClaimForMultipleChannels(t.Context(), request)
					}
					require.ErrorContains(t, err, wantErr)
				})
			}
		})
	}
}

func TestControlStartClaimRejections(t *testing.T) {
	for _, failure := range []string{"MPE address", "signature", "lookup", "cleanup"} {
		t.Run(failure, func(t *testing.T) {
			isolateConfig(t)
			f := newControlServiceFixture(t)
			f.channelMock.paymentChannelFn = func(*PaymentChannelKey) (*PaymentChannelData, bool, error) {
				return &PaymentChannelData{Nonce: big.NewInt(3)}, true, nil
			}
			request := &StartClaimRequest{MpeAddress: f.svcMD.MpeAddress, ChannelId: bigIntToBytes(big.NewInt(1))}
			request.Signature = getSignature(bytes.Join([][]byte{
				[]byte("__start_claim"), f.svcMD.GetMpeAddress().Bytes(), request.ChannelId, bigIntToBytes(big.NewInt(3)),
			}, nil), f.signerKey)
			wantErr := "signature"
			switch failure {
			case "MPE address":
				request.MpeAddress = "invalid"
				wantErr = "mpeAddress"
			case "signature":
				request.Signature = []byte{1}
			case "lookup":
				f.channelMock.paymentChannelFn = func(*PaymentChannelKey) (*PaymentChannelData, bool, error) {
					return nil, false, errors.New("read failed")
				}
				wantErr = "read failed"
			case "cleanup":
				f.channelMock.listClaimsFn = func() ([]Claim, error) { return nil, errors.New("storage failed") }
				wantErr = "error in retrieving claims"
			}
			f.channelMock.startClaimFn = func(*PaymentChannelKey, ChannelUpdate) (Claim, error) {
				t.Fatal("rejected request must not start a claim")
				return nil, nil
			}
			reply, err := f.service.StartClaim(t.Context(), request)
			require.Nil(t, reply)
			require.ErrorContains(t, err, wantErr)
		})
	}
}

func TestBatchClaimsStopOnChannelReadError(t *testing.T) {
	isolateConfig(t)
	f := newControlServiceFixture(t)
	wantErr := errors.New("channel read failed")
	var requested []int64
	f.channelMock.paymentChannelFn = func(key *PaymentChannelKey) (*PaymentChannelData, bool, error) {
		requested = append(requested, key.ID.Int64())
		return nil, false, wantErr
	}
	_, err := f.service.startClaims(&StartMultipleClaimRequest{ChannelIds: []uint64{1, 2}})
	require.ErrorIs(t, err, wantErr)
	require.Equal(t, []int64{1}, requested)
}

func TestRemoveClaimedPaymentsFinishError(t *testing.T) {
	isolateConfig(t)
	f := newControlServiceFixture(t)
	wantErr := errors.New("delete failed")
	claim := &mockClaim{payment: &Payment{ChannelID: big.NewInt(1), ChannelNonce: big.NewInt(0)}, finishErr: wantErr}
	f.channelMock.listClaimsFn = func() ([]Claim, error) { return []Claim{claim}, nil }
	f.channelMock.paymentChannelFromChainFn = func(*PaymentChannelKey) (*PaymentChannelData, bool, error) {
		return &PaymentChannelData{Nonce: big.NewInt(1)}, true, nil
	}
	require.ErrorIs(t, f.service.removeClaimedPayments(), wantErr)
	require.True(t, claim.finished)
}
