package escrow

import (

	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/singnet/snet-daemon/blockchain"

)
//todo, Work in progress
func TestProviderControlService_GetListInProgress(t *testing.T) {

}

func TestProviderControlService_checkMpeAddress(t *testing.T) {
	servicemetadata := blockchain.ServiceMetadata{}
	servicemetadata.MpeAddress = "0xE8D09a6C296aCdd4c01b21f407ac93fdfC63E78C"
	control_service := NewProviderControlService(nil,&servicemetadata,nil)
	err := control_service.checkMpeAddress("0xe8D09a6C296aCdd4c01b21f407ac93fdfC63E78C")
	assert.Nil(t,err)
	err = control_service.checkMpeAddress("0xe9D09a6C296aCdd4c01b21f407ac93fdfC63E78C")
	assert.Equal(t,err.Error(),"the mpeAddress: 0xe9D09a6C296aCdd4c01b21f407ac93fdfC63E78C passed does not match to what has been registered")
}
