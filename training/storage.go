package training

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/singnet/snet-daemon/v6/blockchain"
	"github.com/singnet/snet-daemon/v6/config"
	"go.uber.org/zap"

	"github.com/singnet/snet-daemon/v6/storage"
	"github.com/singnet/snet-daemon/v6/utils"
)

type ModelStorage struct {
	delegate             storage.TypedAtomicStorage
	organizationMetaData *blockchain.OrganizationMetaData
}

type ModelUserStorage struct {
	delegate             storage.TypedAtomicStorage
	organizationMetaData *blockchain.OrganizationMetaData
}

type PendingModelStorage struct {
	delegate             storage.TypedAtomicStorage
	organizationMetaData *blockchain.OrganizationMetaData
}

type PublicModelStorage struct {
	delegate             storage.TypedAtomicStorage
	organizationMetaData *blockchain.OrganizationMetaData
}

func NewUserModelStorage(atomicStorage storage.AtomicStorage, orgMetadata *blockchain.OrganizationMetaData) *ModelUserStorage {
	prefixedStorage := storage.NewPrefixedAtomicStorage(atomicStorage, "/model-user/userModelStorage")
	userModelStorage := storage.NewTypedAtomicStorageImpl(
		prefixedStorage, serializeModelUserKey, reflect.TypeOf(ModelUserKey{}), utils.Serialize, utils.Deserialize,
		reflect.TypeOf(ModelUserData{}),
	)
	return &ModelUserStorage{delegate: userModelStorage, organizationMetaData: orgMetadata}
}

func NewModelStorage(atomicStorage storage.AtomicStorage, orgMetadata *blockchain.OrganizationMetaData) *ModelStorage {
	prefixedStorage := storage.NewPrefixedAtomicStorage(atomicStorage, "/model-user/modelStorage")
	modelStorage := storage.NewTypedAtomicStorageImpl(
		prefixedStorage, serializeModelKey, reflect.TypeOf(ModelKey{}), utils.Serialize, utils.Deserialize,
		reflect.TypeOf(ModelData{}),
	)
	return &ModelStorage{delegate: modelStorage, organizationMetaData: orgMetadata}
}

func NewPendingModelStorage(atomicStorage storage.AtomicStorage, orgMetadata *blockchain.OrganizationMetaData) *PendingModelStorage {
	prefixedStorage := storage.NewPrefixedAtomicStorage(atomicStorage, "/model-user/pendingModelStorage")
	pendingModelStorage := storage.NewTypedAtomicStorageImpl(
		prefixedStorage, serializePendingModelKey, reflect.TypeOf(PendingModelKey{}), utils.Serialize, utils.Deserialize,
		reflect.TypeOf(PendingModelData{}),
	)
	return &PendingModelStorage{delegate: pendingModelStorage, organizationMetaData: orgMetadata}
}

func NewPublicModelStorage(atomicStorage storage.AtomicStorage, orgMetadata *blockchain.OrganizationMetaData) *PublicModelStorage {
	prefixedStorage := storage.NewPrefixedAtomicStorage(atomicStorage, "/model-user/publicModelStorage")
	publicModelStorage := storage.NewTypedAtomicStorageImpl(
		prefixedStorage, serializePublicModelKey, reflect.TypeOf(PublicModelKey{}), utils.Serialize, utils.Deserialize,
		reflect.TypeOf(PublicModelData{}),
	)
	return &PublicModelStorage{delegate: publicModelStorage, organizationMetaData: orgMetadata}
}

type ModelKey struct {
	OrganizationId string
	ServiceId      string
	GroupId        string
	ModelId        string
}

func (key *ModelKey) String() string {
	return fmt.Sprintf("{ID:%v|%v|%v|%v}", key.OrganizationId,
		key.ServiceId, key.GroupId, key.ModelId)
}

type ModelData struct {
	ModelId             string
	IsPublic            bool
	Status              Status
	ModelName           string
	AuthorizedAddresses []string
	CreatedByAddress    string
	UpdatedByAddress    string
	GroupId             string
	OrganizationId      string
	ServiceId           string
	GRPCMethodName      string
	GRPCServiceName     string
	Description         string
	TrainingLink        string
	ValidatePrice       uint64
	TrainPrice          uint64
	UpdatedDate         string
	CreatedDate         string
}

func (data *ModelData) String() string {
	return fmt.Sprintf("{DATA:%v|%v|%v|%v|%v|%v|Name:%v|IsPublic:%v|AuthorizedAddresses:%v|CreatedBy:%v|UpdatedBy:%v|Status:%v|TrainingLink:%v|Updated:%v|Created:%v|ValPrice:%v|TrPrice:%v|Desc:%v}",
		data.OrganizationId, data.ServiceId, data.GroupId, data.GRPCServiceName, data.GRPCMethodName, data.ModelId, data.ModelName, data.IsPublic, data.AuthorizedAddresses,
		data.CreatedByAddress, data.UpdatedByAddress, data.Status, data.TrainingLink, data.UpdatedDate, data.CreatedDate, data.ValidatePrice, data.TrainPrice, data.Description)
}

type ModelUserKey struct {
	OrganizationId string
	ServiceId      string
	GroupId        string
	UserAddress    string
}

func (key *ModelUserKey) String() string {
	return fmt.Sprintf("{ID:%v|%v|%v|%v}", key.OrganizationId,
		key.ServiceId, key.GroupId, key.UserAddress)
}

// ModelUserData maintain the list of all modelIds for a given user address
type ModelUserData struct {
	ModelIds []string
	//the below are only for display purposes
	OrganizationId string
	ServiceId      string
	GroupId        string
	UserAddress    string
}

func (data *ModelUserData) String() string {
	return fmt.Sprintf("{DATA:%v|%v|%v|%v|%v}",
		data.OrganizationId,
		data.ServiceId, data.GroupId, data.UserAddress, data.ModelIds)
}

type PendingModelKey struct {
	OrganizationId string
	ServiceId      string
	GroupId        string
}

func (key *PendingModelKey) String() string {
	return fmt.Sprintf("{ID:%v|%v|%v}", key.OrganizationId, key.ServiceId, key.GroupId)
}

type PendingModelData struct {
	ModelIDs []string
}

// PendingModelData maintain the list of all modelIds that have TRAINING\VALIDATING status
func (data *PendingModelData) String() string {
	return fmt.Sprintf("{DATA:%v}", data.ModelIDs)
}

type PublicModelKey struct {
	OrganizationId string
	ServiceId      string
	GroupId        string
}

func (key *PublicModelKey) String() string {
	return fmt.Sprintf("{ID:%v|%v|%v}", key.OrganizationId, key.ServiceId, key.GroupId)
}

type PublicModelData struct {
	ModelIDs []string
}

func (data *PublicModelData) String() string {
	return fmt.Sprintf("{DATA:%v}", data.ModelIDs)
}

func serializeModelKey(key any) (serialized string, err error) {
	modelKey := key.(*ModelKey)
	return modelKey.String(), nil
}

func (storage *ModelStorage) Get(key *ModelKey) (state *ModelData, ok bool, err error) {
	value, ok, err := storage.delegate.Get(key)
	if err != nil || !ok {
		return nil, ok, err
	}
	return value.(*ModelData), ok, err
}

func (storage *ModelStorage) GetAll() (states []*ModelData, err error) {
	values, err := storage.delegate.GetAll()
	if err != nil {
		return
	}

	return values.([]*ModelData), nil
}

func (storage *ModelStorage) Put(key *ModelKey, state *ModelData) (err error) {
	return storage.delegate.Put(key, state)
}

func (storage *ModelStorage) PutIfAbsent(key *ModelKey, state *ModelData) (ok bool, err error) {
	return storage.delegate.PutIfAbsent(key, state)
}

func (storage *ModelStorage) CompareAndSwap(key *ModelKey, prevState *ModelData,
	newState *ModelData) (ok bool, err error) {
	return storage.delegate.CompareAndSwap(key, prevState, newState)
}

func (storage *ModelStorage) buildModelKey(modelID string) (key *ModelKey) {
	key = &ModelKey{
		OrganizationId: config.GetString(config.OrganizationId),
		ServiceId:      config.GetString(config.ServiceId),
		GroupId:        storage.organizationMetaData.GetGroupIdString(),
		ModelId:        modelID,
	}
	return
}

func (storage *ModelStorage) GetModel(modelID string) (data *ModelData, err error) {
	key := storage.buildModelKey(modelID)
	ok := false
	if data, ok, err = storage.Get(key); err != nil || !ok {
		zap.L().Warn("unable to retrieve model data from storage", zap.String("Model Id", key.ModelId), zap.Error(err))
	}
	return
}

func serializeModelUserKey(key any) (serialized string, err error) {
	modelUserKey := key.(*ModelUserKey)
	return modelUserKey.String(), nil
}

func (storage *ModelUserStorage) buildModelUserKey(address string) *ModelUserKey {
	return &ModelUserKey{
		OrganizationId: config.GetString(config.OrganizationId),
		ServiceId:      config.GetString(config.ServiceId),
		GroupId:        storage.organizationMetaData.GetGroupIdString(),
		UserAddress:    strings.ToLower(address),
	}
}

func (storage *ModelUserStorage) Get(key *ModelUserKey) (state *ModelUserData, ok bool, err error) {
	value, ok, err := storage.delegate.Get(key)
	if err != nil || !ok {
		return nil, ok, err
	}
	return value.(*ModelUserData), ok, err
}

func (storage *ModelUserStorage) GetAll() (states []*ModelUserData, err error) {
	values, err := storage.delegate.GetAll()
	if err != nil {
		return
	}

	return values.([]*ModelUserData), nil
}

func (storage *ModelUserStorage) Put(key *ModelUserKey, state *ModelUserData) (err error) {
	return storage.delegate.Put(key, state)
}

func (storage *ModelUserStorage) PutIfAbsent(key *ModelUserKey, state *ModelUserData) (ok bool, err error) {
	return storage.delegate.PutIfAbsent(key, state)
}

func (storage *ModelUserStorage) CompareAndSwap(key *ModelUserKey, prevState *ModelUserData,
	newState *ModelUserData) (ok bool, err error) {
	return storage.delegate.CompareAndSwap(key, prevState, newState)
}

func serializePendingModelKey(key any) (serialized string, err error) {
	pendingModelKey := key.(*PendingModelKey)
	return pendingModelKey.String(), nil
}

func (pendingStorage *PendingModelStorage) Get(key *PendingModelKey) (state *PendingModelData, ok bool, err error) {
	value, ok, err := pendingStorage.delegate.Get(key)
	if err != nil || !ok {
		return nil, ok, err
	}

	return value.(*PendingModelData), ok, err
}

func (pendingStorage *PendingModelStorage) GetAll() (states []*PendingModelData, err error) {
	values, err := pendingStorage.delegate.GetAll()
	if err != nil {
		return
	}

	return values.([]*PendingModelData), nil
}

func (pendingStorage *PendingModelStorage) Put(key *PendingModelKey, state *PendingModelData) (err error) {
	return pendingStorage.delegate.Put(key, state)
}

func (pendingStorage *PendingModelStorage) buildPendingModelKey() *PendingModelKey {
	return &PendingModelKey{
		OrganizationId: config.GetString(config.OrganizationId),
		ServiceId:      config.GetString(config.ServiceId),
		GroupId:        pendingStorage.organizationMetaData.GetGroupIdString(),
	}
}

func (pendingStorage *PendingModelStorage) AddPendingModelId(key *PendingModelKey, modelId string) (err error) {

	typedUpdateFunc := func(conditionValues []storage.TypedKeyValueData) (update []storage.TypedKeyValueData, ok bool, err error) {
		if len(conditionValues) != 1 || conditionValues[0].Key != key {
			return nil, false, fmt.Errorf("unexpected condition values or missing key")
		}

		// Fetch the current list of pending model IDs from the storage
		currentValue, _, err := pendingStorage.delegate.Get(key)
		if err != nil {
			return nil, false, err
		}

		var pendingModelData *PendingModelData
		if currentValue == nil {
			pendingModelData = &PendingModelData{ModelIDs: make([]string, 0, 100)}
		} else {
			pendingModelData = currentValue.(*PendingModelData)
		}

		// Check if the modelId already exists
		for _, currentModelId := range pendingModelData.ModelIDs {
			if currentModelId == modelId {
				// If the model ID already exists, no update is needed
				return nil, false, nil
			}
		}

		zap.L().Debug("[AddPendingModelId]", zap.Strings("modelIDS", pendingModelData.ModelIDs))

		// Add the new model ID to the list
		pendingModelData.ModelIDs = append(pendingModelData.ModelIDs, modelId)

		zap.L().Debug("[AddPendingModelId]", zap.Strings("modelIDS", pendingModelData.ModelIDs))

		// Prepare the updated values for the transaction
		newValues := []storage.TypedKeyValueData{
			{
				Key:     key,
				Value:   pendingModelData,
				Present: true,
			},
		}

		return newValues, true, nil
	}

	request := storage.TypedCASRequest{
		ConditionKeys:           []any{key},
		RetryTillSuccessOrError: true,
		Update:                  typedUpdateFunc,
	}

	// Execute the transaction
	ok, err := pendingStorage.delegate.ExecuteTransaction(request)
	if err != nil {
		return fmt.Errorf("transaction execution failed: %w", err)
	}
	if !ok {
		return fmt.Errorf("transaction was not successful")
	}

	return nil
}

func (pendingStorage *PendingModelStorage) RemovePendingModelId(key *PendingModelKey, modelId string) (err error) {

	typedUpdateFunc := func(conditionValues []storage.TypedKeyValueData) (update []storage.TypedKeyValueData, ok bool, err error) {
		if len(conditionValues) != 1 || conditionValues[0].Key != key {
			return nil, false, fmt.Errorf("unexpected condition values or missing key")
		}

		// Fetch the current list of pending model IDs from the storage
		currentValue, ok, err := pendingStorage.delegate.Get(key)
		if err != nil {
			return nil, false, err
		}

		var pendingModelData *PendingModelData
		if currentValue == nil {
			return
		} else {
			pendingModelData = currentValue.(*PendingModelData)
		}

		zap.L().Debug("[RemovePendingModelId]", zap.Strings("modelIDS", pendingModelData.ModelIDs))

		pendingModelData.ModelIDs = remove(pendingModelData.ModelIDs, modelId)

		zap.L().Debug("[RemovePendingModelId]", zap.Strings("after remove modelIDS", pendingModelData.ModelIDs))

		// Prepare the updated values for the transaction
		newValues := []storage.TypedKeyValueData{
			{
				Key:     key,
				Value:   pendingModelData,
				Present: true,
			},
		}

		return newValues, true, nil
	}

	request := storage.TypedCASRequest{
		ConditionKeys:           []any{key},
		RetryTillSuccessOrError: true,
		Update:                  typedUpdateFunc,
	}

	// Execute the transaction
	ok, err := pendingStorage.delegate.ExecuteTransaction(request)
	if err != nil {
		return fmt.Errorf("transaction execution failed: %w", err)
	}
	if !ok {
		return fmt.Errorf("transaction was not successful")
	}

	return nil
}

func (pendingStorage *PendingModelStorage) PutIfAbsent(key *PendingModelKey, state *PendingModelData) (ok bool, err error) {
	return pendingStorage.delegate.PutIfAbsent(key, state)
}

func (pendingStorage *PendingModelStorage) CompareAndSwap(key *PendingModelKey, prevState *PendingModelData,
	newState *PendingModelData) (ok bool, err error) {
	return pendingStorage.delegate.CompareAndSwap(key, prevState, newState)
}

func serializePublicModelKey(key any) (serialized string, err error) {
	pendingModelKey := key.(*PublicModelKey)
	return pendingModelKey.String(), nil
}

func (publicStorage *PublicModelStorage) Get(key *PublicModelKey) (state *PublicModelData, ok bool, err error) {
	value, ok, err := publicStorage.delegate.Get(key)
	if err != nil || !ok {
		return nil, ok, err
	}

	return value.(*PublicModelData), ok, err
}

func (publicStorage *PublicModelStorage) GetAll() (states []*PublicModelData, err error) {
	values, err := publicStorage.delegate.GetAll()
	if err != nil {
		return
	}

	return values.([]*PublicModelData), nil
}

func (publicStorage *PublicModelStorage) Put(key *PublicModelKey, state *PublicModelData) (err error) {
	return publicStorage.delegate.Put(key, state)
}

func (publicStorage *PublicModelStorage) AddPublicModelId(key *PublicModelKey, modelId string) (err error) {
	typedUpdateFunc := func(conditionValues []storage.TypedKeyValueData) (update []storage.TypedKeyValueData, ok bool, err error) {
		if len(conditionValues) != 1 || conditionValues[0].Key != key {
			return nil, false, fmt.Errorf("unexpected condition values or missing key")
		}

		// Fetch the current list of public model IDs from the storage
		currentValue, _, err := publicStorage.delegate.Get(key)
		if err != nil {
			return nil, false, err
		}

		var publicModelData *PublicModelData
		if currentValue == nil {
			publicModelData = &PublicModelData{ModelIDs: make([]string, 0, 100)}
		} else {
			publicModelData = currentValue.(*PublicModelData)
		}

		// Check if the modelId already exists
		for _, currentModelId := range publicModelData.ModelIDs {
			if currentModelId == modelId {
				// If the model ID already exists, no update is needed
				return nil, false, nil
			}
		}

		// Add the new model ID to the list
		publicModelData.ModelIDs = append(publicModelData.ModelIDs, modelId)

		// Prepare the updated values for the transaction
		newValues := []storage.TypedKeyValueData{
			{
				Key:     key,
				Value:   publicModelData,
				Present: true,
			},
		}

		return newValues, true, nil
	}

	request := storage.TypedCASRequest{
		ConditionKeys:           []any{key},
		RetryTillSuccessOrError: true,
		Update:                  typedUpdateFunc,
	}

	// Execute the transaction
	ok, err := publicStorage.delegate.ExecuteTransaction(request)
	if err != nil {
		return fmt.Errorf("transaction execution failed: %w", err)
	}
	if !ok {
		return fmt.Errorf("transaction was not successful")
	}

	return nil
}

func (publicStorage *PublicModelStorage) PutIfAbsent(key *PublicModelKey, state *PublicModelData) (ok bool, err error) {
	return publicStorage.delegate.PutIfAbsent(key, state)
}

func (publicStorage *PublicModelStorage) CompareAndSwap(key *PublicModelKey, prevState *PublicModelData,
	newState *PublicModelData) (ok bool, err error) {
	return publicStorage.delegate.CompareAndSwap(key, prevState, newState)
}

func (publicStorage *PublicModelStorage) buildPublicModelKey() *PublicModelKey {
	return &PublicModelKey{
		OrganizationId: config.GetString(config.OrganizationId),
		ServiceId:      config.GetString(config.ServiceId),
		GroupId:        publicStorage.organizationMetaData.GetGroupIdString(),
	}
}
