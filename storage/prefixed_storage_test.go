package storage

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newPrefixedTestStorage() *PrefixedAtomicStorage {
	return NewPrefixedAtomicStorage(NewMemStorage(), "prefix")
}

func TestPrefixedAtomicStorage_PutGet(t *testing.T) {
	storage := newPrefixedTestStorage()

	err := storage.Put("key", "value")
	assert.NoError(t, err)

	value, ok, err := storage.Get("key")
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "value", value)

	// key without prefix is not visible via prefixed storage
	_, ok, err = storage.Get("other")
	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestPrefixedAtomicStorage_GetMissingKey(t *testing.T) {
	storage := newPrefixedTestStorage()

	_, ok, err := storage.Get("absent")
	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestPrefixedAtomicStorage_PutIfAbsent(t *testing.T) {
	storage := newPrefixedTestStorage()

	ok, err := storage.PutIfAbsent("key", "first")
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = storage.PutIfAbsent("key", "second")
	assert.NoError(t, err)
	assert.False(t, ok)

	value, ok, err := storage.Get("key")
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "first", value)
}

func TestPrefixedAtomicStorage_CompareAndSwap(t *testing.T) {
	storage := newPrefixedTestStorage()

	assert.NoError(t, storage.Put("key", "old"))

	ok, err := storage.CompareAndSwap("key", "wrong", "new")
	assert.NoError(t, err)
	assert.False(t, ok)

	ok, err = storage.CompareAndSwap("key", "old", "new")
	assert.NoError(t, err)
	assert.True(t, ok)

	value, _, err := storage.Get("key")
	assert.NoError(t, err)
	assert.Equal(t, "new", value)
}

func TestPrefixedAtomicStorage_Delete(t *testing.T) {
	storage := newPrefixedTestStorage()

	assert.NoError(t, storage.Put("key", "value"))
	assert.NoError(t, storage.Delete("key"))

	_, ok, err := storage.Get("key")
	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestPrefixedAtomicStorage_GetByKeyPrefix(t *testing.T) {
	storage := newPrefixedTestStorage()

	assert.NoError(t, storage.Put("a/one", "1"))
	assert.NoError(t, storage.Put("a/two", "2"))
	assert.NoError(t, storage.Put("b/three", "3"))

	values, err := storage.GetByKeyPrefix("a/")
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"1", "2"}, values)
}

func TestPrefixedAtomicStorage_AppendKeyPrefix(t *testing.T) {
	storage := newPrefixedTestStorage()

	assert.Equal(t, []string{"prefix/a", "prefix/b"}, storage.appendKeyPrefix([]string{"a", "b"}))
	assert.Equal(t, []string{}, storage.appendKeyPrefix([]string{}))

	// source slice must not be modified
	src := []string{"a"}
	_ = storage.appendKeyPrefix(src)
	assert.Equal(t, []string{"a"}, src)
}

func TestPrefixedAtomicStorage_StartTransaction(t *testing.T) {
	storage := newPrefixedTestStorage()
	assert.NoError(t, storage.Put("key", "value"))

	transaction, err := storage.StartTransaction([]string{"key"})
	assert.NoError(t, err)

	values, err := transaction.GetConditionValues()
	assert.NoError(t, err)
	assert.Equal(t, []KeyValueData{{Key: "key", Value: "value", Present: true}}, values)
}

func TestPrefixedAtomicStorage_CompleteTransaction(t *testing.T) {
	storage := newPrefixedTestStorage()
	assert.NoError(t, storage.Put("key", "value"))

	transaction, err := storage.StartTransaction([]string{"key"})
	assert.NoError(t, err)

	ok, err := storage.CompleteTransaction(transaction, []KeyValueData{{Key: "key", Value: "new", Present: true}})
	assert.NoError(t, err)
	assert.True(t, ok)

	value, ok, err := storage.Get("key")
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "new", value)
}

func TestPrefixedAtomicStorage_ExecuteTransaction(t *testing.T) {
	storage := newPrefixedTestStorage()
	assert.NoError(t, storage.Put("key", "value"))

	request := CASRequest{
		RetryTillSuccessOrError: false,
		ConditionKeys:           []string{"key"},
		Update: func(conditionValues []KeyValueData) ([]KeyValueData, bool, error) {
			// condition values come to the update function without prefix
			assert.Equal(t, []KeyValueData{{Key: "key", Value: "value", Present: true}}, conditionValues)
			return []KeyValueData{{Key: "key", Value: "updated", Present: true}}, true, nil
		},
	}

	ok, err := storage.ExecuteTransaction(request)
	assert.NoError(t, err)
	assert.True(t, ok)

	value, ok, err := storage.Get("key")
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "updated", value)
}

func newStringTypedStorage(atomicStorage AtomicStorage) TypedAtomicStorage {
	return NewTypedAtomicStorageImpl(
		atomicStorage,
		strKeySerializer,
		reflect.TypeFor[string](),
		strValueSerializer,
		strValueDeserializer,
		reflect.TypeFor[string](),
	)
}

func strKeySerializer(key any) (string, error) {
	s, ok := key.(string)
	if !ok {
		return "", fmt.Errorf("key is not a string: %v", key)
	}
	return s, nil
}

func strValueSerializer(value any) (string, error) {
	s, ok := value.(*string)
	if !ok {
		return "", fmt.Errorf("value is not a *string: %T", value)
	}
	return *s, nil
}

func strValueDeserializer(serialized string, value any) error {
	s, ok := value.(*string)
	if !ok {
		return fmt.Errorf("value is not a *string: %T", value)
	}
	*s = serialized
	return nil
}

func TestTypedAtomicStorageImpl_PutGet(t *testing.T) {
	storage := newStringTypedStorage(NewMemStorage())

	err := storage.Put("key", new("value"))
	assert.NoError(t, err)

	value, ok, err := storage.Get("key")
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, new("value"), value)

	value, ok, err = storage.Get("absent")
	assert.NoError(t, err)
	assert.False(t, ok)
	assert.Nil(t, value)
}

func TestTypedAtomicStorageImpl_PutIfAbsent(t *testing.T) {
	storage := newStringTypedStorage(NewMemStorage())

	ok, err := storage.PutIfAbsent("key", new("first"))
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = storage.PutIfAbsent("key", new("second"))
	assert.NoError(t, err)
	assert.False(t, ok)

	value, ok, err := storage.Get("key")
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, new("first"), value)
}

func TestTypedAtomicStorageImpl_CompareAndSwap(t *testing.T) {
	storage := newStringTypedStorage(NewMemStorage())

	assert.NoError(t, storage.Put("key", new("old")))

	ok, err := storage.CompareAndSwap("key", new("wrong"), new("new"))
	assert.NoError(t, err)
	assert.False(t, ok)

	ok, err = storage.CompareAndSwap("key", new("old"), new("new"))
	assert.NoError(t, err)
	assert.True(t, ok)

	value, _, err := storage.Get("key")
	assert.NoError(t, err)
	assert.Equal(t, new("new"), value)
}

func TestTypedAtomicStorageImpl_Delete(t *testing.T) {
	storage := newStringTypedStorage(NewMemStorage())

	assert.NoError(t, storage.Put("key", new("value")))
	assert.NoError(t, storage.Delete("key"))

	_, ok, err := storage.Get("key")
	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestTypedAtomicStorageImpl_GetAll(t *testing.T) {
	storage := newStringTypedStorage(NewMemStorage())

	assert.NoError(t, storage.Put("one", new("1")))
	assert.NoError(t, storage.Put("two", new("2")))

	values, err := storage.GetAll()
	assert.NoError(t, err)
	assert.ElementsMatch(t, []*string{new("1"), new("2")}, values)
}

func TestTypedAtomicStorageImpl_ExecuteTransaction(t *testing.T) {
	storage := newStringTypedStorage(NewMemStorage())
	assert.NoError(t, storage.Put("key", new("value")))

	request := TypedCASRequest{
		RetryTillSuccessOrError: false,
		ConditionKeys:           []any{"key"},
		Update: func(conditionValues []TypedKeyValueData) ([]TypedKeyValueData, bool, error) {
			assert.Equal(t, []TypedKeyValueData{{Key: "key", Value: new("value"), Present: true}}, conditionValues)
			return []TypedKeyValueData{{Key: "key", Value: new("updated"), Present: true}}, true, nil
		},
	}

	ok, err := storage.ExecuteTransaction(request)
	assert.NoError(t, err)
	assert.True(t, ok)

	value, ok, err := storage.Get("key")
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, new("updated"), value)
}

func TestTypedAtomicStorageImpl_SerializationErrors(t *testing.T) {
	storage := newStringTypedStorage(NewMemStorage())

	// invalid key type
	_, ok, err := storage.Get(123)
	assert.Error(t, err)
	assert.False(t, ok)

	err = storage.Put(123, new("value"))
	assert.Error(t, err)

	// invalid value type
	err = storage.Put("key", "not-a-pointer")
	assert.Error(t, err)

	ok, err = storage.PutIfAbsent(123, new("value"))
	assert.Error(t, err)
	assert.False(t, ok)

	ok, err = storage.CompareAndSwap("key", "invalid", new("new"))
	assert.Error(t, err)
	assert.False(t, ok)

	err = storage.Delete(123)
	assert.Error(t, err)
}
