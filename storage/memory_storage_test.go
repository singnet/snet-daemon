package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetValueDataForKeyReturnsNotFound(t *testing.T) {
	data, present := getValueDataForKey("missing", []KeyValueData{{Key: "other", Value: "value"}})

	require.False(t, present)
	require.Equal(t, KeyValueData{}, data)
}

func TestPutAndGet(t *testing.T) {
	s := NewMemStorage()

	err := s.Put("foo", "bar")
	assert.NoError(t, err, "Put should not return an error")

	val, ok, err := s.Get("foo")
	assert.NoError(t, err, "Get should not return an error")
	assert.True(t, ok, "Expected key to exist")
	assert.Equal(t, "bar", val, "Expected value 'bar'")
}

func TestPutIfAbsent(t *testing.T) {
	s := NewMemStorage()

	ok, err := s.PutIfAbsent("key", "val1")
	assert.NoError(t, err)
	assert.True(t, ok, "PutIfAbsent should succeed on empty key")

	ok, err = s.PutIfAbsent("key", "val2")
	assert.NoError(t, err)
	assert.False(t, ok, "PutIfAbsent should not overwrite existing key")

	val, _, _ := s.Get("key")
	assert.Equal(t, "val1", val, "Value should remain 'val1'")
}

func TestCompareAndSwap(t *testing.T) {
	s := NewMemStorage()
	_ = s.Put("k", "old")

	ok, err := s.CompareAndSwap("k", "old", "new")
	assert.NoError(t, err)
	assert.True(t, ok, "CAS should succeed with correct old value")

	val, _, _ := s.Get("k")
	assert.Equal(t, "new", val, "Value should be updated to 'new'")

	ok, _ = s.CompareAndSwap("k", "wrong", "other")
	assert.False(t, ok, "CAS should fail with wrong old value")
}

func TestDeleteAndClear(t *testing.T) {
	s := NewMemStorage()
	_ = s.Put("a", "1")
	_ = s.Put("b", "2")

	_ = s.Delete("a")
	_, ok, _ := s.Get("a")
	assert.False(t, ok, "Expected key 'a' to be deleted")

	_ = s.Clear()
	assert.Empty(t, s.data, "Storage should be empty after Clear")
}

func TestGetByKeyPrefix(t *testing.T) {
	s := NewMemStorage()
	_ = s.Put("user:1", "Alice")
	_ = s.Put("user:2", "Bob")
	_ = s.Put("order:1", "XYZ")

	users, _ := s.GetByKeyPrefix("user:")
	assert.Len(t, users, 2, "Expected 2 users")
}
