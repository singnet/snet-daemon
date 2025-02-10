package training

import (
	"bytes"
	_ "embed"
	"github.com/bufbuild/protocompile/linker"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/structpb"
	"math/big"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/singnet/snet-daemon/v5/authutils"
	"github.com/singnet/snet-daemon/v5/utils"
)

// message used to sign is of the form ("__create_model", mpe_address, current_block_number)
func (ds *DaemonService) verifySignature(request *AuthorizationDetails) error {
	return authutils.VerifySigner(ds.getMessageBytes(request.Message, request),
		request.GetSignature(), utils.ToChecksumAddress(request.SignerAddress))
}

// "user passed message	", user_address, current_block_number
func (ds *DaemonService) getMessageBytes(prefixMessage string, request *AuthorizationDetails) []byte {
	userAddress := utils.ToChecksumAddress(request.SignerAddress)
	message := bytes.Join([][]byte{
		[]byte(prefixMessage),
		userAddress.Bytes(),
		math.U256Bytes(big.NewInt(int64(request.CurrentBlock))),
	}, nil)
	return message
}

func remove(s []string, r string) []string {
	for i, v := range s {
		if v == r {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}

func difference(oldAddresses []string, newAddresses []string) []string {
	var diff []string
	for i := 0; i < 2; i++ {
		for _, s1 := range oldAddresses {
			found := false
			for _, s2 := range newAddresses {
				if s1 == s2 {
					found = true
					break
				}
			}
			// String not found. We add it to return slice
			if !found {
				diff = append(diff, s1)
			}
		}
		// Swap the slices, only if it was the first loop
		if i == 0 {
			oldAddresses, newAddresses = newAddresses, oldAddresses
		}
	}
	return diff
}

func isValuePresent(value string, list []string) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}

//go:embed training.proto
var TrainingProtoEmbeded string

// parseTrainingMetadata parses metadata from Protobuf files to identify training-related methods
// and their associated metadata.
// Input:
// - protos: a collection of Protobuf files containing definitions of services and methods.
// Output:
//   - methodsMD: a map where the key is the combination of service and method names,
//     and the value is metadata related to the method (MethodMetadata).
//   - trainingMD: a structure containing metadata for training methods, including
//     whether training methods are defined and their names grouped by service.
//   - err: an error, if any occurred during the parsing process.
func parseTrainingMetadata(protos linker.Files) (methodsMD map[string]*MethodMetadata, trainingMD *TrainingMetadata, err error) {
	methodsMD = make(map[string]*MethodMetadata)
	trainingMD = &TrainingMetadata{}
	trainingMD.TrainingMethods = make(map[string]*structpb.ListValue)

	for _, protoFile := range protos {
		for servicesCounter := 0; servicesCounter < protoFile.Services().Len(); servicesCounter++ {
			protoService := protoFile.Services().Get(servicesCounter)
			if protoService == nil {
				continue
			}
			for methodsCounter := 0; methodsCounter < protoService.Methods().Len(); methodsCounter++ {
				method := protoFile.Services().Get(servicesCounter).Methods().Get(methodsCounter)
				if method == nil {
					continue
				}
				inputFields := method.Input().Fields()
				if inputFields == nil {
					continue
				}
				for fieldsCounter := 0; fieldsCounter < inputFields.Len(); fieldsCounter++ {
					if inputFields.Get(fieldsCounter).Message() != nil {
						// if the method accepts modelId, then we consider it as training
						if inputFields.Get(fieldsCounter).Message().FullName() == "training.ModelID" {
							// init array if nil
							trainingMD.TrainingInProto = true
							if trainingMD.TrainingMethods[string(protoService.Name())] == nil {
								trainingMD.TrainingMethods[string(protoService.Name())], _ = structpb.NewList(nil)
							}
							value := structpb.NewStringValue(string(method.Name()))
							trainingMD.TrainingMethods[string(protoService.Name())].Values = append(trainingMD.TrainingMethods[string(protoService.Name())].Values, value)
						}
					}
				}

				methodOptions, ok := method.Options().(*descriptorpb.MethodOptions)
				if ok && methodOptions != nil {
					key := string(protoService.Name() + method.Name())
					methodsMD[key] = &MethodMetadata{}
					if proto.HasExtension(methodOptions, E_DatasetDescription) {
						if datasetDesc, ok := proto.GetExtension(methodOptions, E_DatasetDescription).(string); ok {
							methodsMD[key].DatasetDescription = datasetDesc
						}
					}
					if proto.HasExtension(methodOptions, E_DatasetType) {
						if datasetType, ok := proto.GetExtension(methodOptions, E_DatasetType).(string); ok {
							methodsMD[key].DatasetType = datasetType
						}
					}
					if proto.HasExtension(methodOptions, E_DatasetFilesType) {
						if datasetDesc, ok := proto.GetExtension(methodOptions, E_DatasetFilesType).(string); ok {
							methodsMD[key].DatasetFilesType = datasetDesc
						}
					}
					if proto.HasExtension(methodOptions, E_MaxModelsPerUser) {
						if datasetDesc, ok := proto.GetExtension(methodOptions, E_MaxModelsPerUser).(uint64); ok {
							methodsMD[key].MaxModelsPerUser = datasetDesc
						}
					}
					if proto.HasExtension(methodOptions, E_DefaultModelId) {
						if defaultModelId, ok := proto.GetExtension(methodOptions, E_DefaultModelId).(string); ok {
							methodsMD[key].DefaultModelId = defaultModelId
						}
					}
					if proto.HasExtension(methodOptions, E_DatasetMaxSizeSingleFileMb) {
						if d, ok := proto.GetExtension(methodOptions, E_DatasetMaxSizeSingleFileMb).(uint64); ok {
							methodsMD[key].DatasetMaxSizeSingleFileMb = d
						}
					}
					if proto.HasExtension(methodOptions, E_DatasetMaxCountFiles) {
						if maxCountFiles, ok := proto.GetExtension(methodOptions, E_DatasetMaxCountFiles).(uint64); ok {
							methodsMD[key].DatasetMaxCountFiles = maxCountFiles
						}
					}
					if proto.HasExtension(methodOptions, E_DatasetMaxSizeMb) {
						if datasetMaxSizeMb, ok := proto.GetExtension(methodOptions, E_DatasetMaxSizeMb).(uint64); ok {
							methodsMD[key].DatasetMaxSizeMb = datasetMaxSizeMb
						}
					}
					if methodsMD[key].DefaultModelId != "" {
						zap.L().Debug("training metadata", zap.String("method", string(method.Name())), zap.String("key", key), zap.Any("metadata", methodsMD[key]))
					}
				}
			}
		}
	}
	zap.L().Debug("training methods", zap.Any("methods", trainingMD.TrainingMethods))
	return
}
