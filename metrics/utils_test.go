package metrics

import (
	"go/types"
	"math/big"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/singnet/snet-daemon/v6/config"
	"github.com/singnet/snet-daemon/v6/utils"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
)

func TestGenXid(t *testing.T) {
	id1 := GenXid()
	id2 := GenXid()
	assert.NotEqual(t, id1, id2)

}
func TestGetValue(t *testing.T) {
	md := metadata.Pairs("user-agent", "Test user agent", "user-agent", "user-agent", "content-type", "application/grpc")
	assert.Equal(t, "Test user agent", GetValue(md, "user-agent"))
	assert.Equal(t, GetValue(md, ""), "")
	md = metadata.Pairs()
	assert.Equal(t, GetValue(md, ""), "")
}

func TestPublish(t *testing.T) {
	status := Publish(nil, "", nil, big.NewInt(0))
	assert.Equal(t, status, false)
	status = Publish(nil, "http://localhost:8080", nil, big.NewInt(0))
	assert.Equal(t, status, false)

	status = Publish(struct {
		title string
	}{
		title: "abcd",
	}, "http://localhost:8080", &CommonStats{}, big.NewInt(0))
	assert.Equal(t, status, false)
}

func TestCheckSuccessfulResponse(t *testing.T) {
	status, _ := checkForSuccessfulResponse(nil)
	assert.Equal(t, status, false)
	status, _ = checkForSuccessfulResponse(&http.Response{StatusCode: http.StatusForbidden})
	assert.Equal(t, status, false)
}

//func TestGetSize(t *testing.T) {
//	strt1 := struct {
//		title string
//	}{
//		title: "abcd",
//	}
//	assert.Equal(t, strconv.FormatUint(GetSize(strt1), 10), "20")
//
//	strt2 := struct {
//		title string
//	}{
//		title: "abcdeefffffffffffffffff",
//	}
//	assert.Equal(t, strconv.FormatUint(GetSize(strt2), 10), "39")
//}

func TestGetEpochTime(t *testing.T) {
	currentEpoch := getEpochTime()
	assert.NotNil(t, currentEpoch, "Epoch must not be empty")
	assert.NotEqual(t, currentEpoch, 0, "epoch msut not be zero")
	assert.IsType(t, reflect.TypeOf(types.Int64), reflect.TypeOf(currentEpoch), "Epoch must be an integer")

	// two epochs must not be equal
	time.Sleep(1 * time.Second)
	secondEpoch := getEpochTime()
	assert.NotEqual(t, currentEpoch, secondEpoch, "Epochs msut not be the same")
}

func Test_getPrivateKeyForMetering(t *testing.T) {
	config.Vip().Set(config.PvtKeyForMetering, "063C00D18E147F4F734846E47FE6598FC7A6D56307862F7EDC92B9F43CC27EDD")
	key, err := getPrivateKeyForMetering()
	if err == nil {
		assert.Equal(t, crypto.PubkeyToAddress(key.PublicKey).String(), "0x94d04332C4f5273feF69c4a52D24f42a3aF1F207")
		assert.NotNil(t, key)
		assert.Nil(t, err)

		bytesForMetering := signForMeteringValidation(key, big.NewInt(123), MeteringPrefix, &CommonStats{UserName: "test-user"})
		signature := utils.GetSignature(bytesForMetering, key)
		signer, err := utils.GetSignerAddressFromMessage(bytesForMetering, signature)
		assert.NotNil(t, signer)
		assert.Equal(t, signer.String(), "0x94d04332C4f5273feF69c4a52D24f42a3aF1F207")
		assert.Nil(t, err)
	}
}
