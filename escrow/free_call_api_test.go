package escrow

import (
	"github.com/magiconair/properties/assert"
	"testing"
)

func TestFreeCallUserKey_String(t *testing.T) {
	key := &FreeCallUserKey{UserId: "A", OrganizationId: "B", ServiceId: "C", GroupID: "D"}
	assert.Equal(t, "{ID:A/B/C/D}", key.String())
}

func TestFreeCallUserData_String(t *testing.T) {
	data := &FreeCallUserData{FreeCallsMade: 10,UserId:"abc@test.com"}
	IncrementFreeCallCount(data)
	assert.Equal(t, "{User abc@test.com has made 11 free calls}", data.String())

}
