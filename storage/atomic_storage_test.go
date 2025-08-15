package storage

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Dummy key serializer: converts key to string
func dummyKeySerializer(key any) (string, error) {
	s, ok := key.(string)
	if !ok {
		return "", fmt.Errorf("key is not a string: %v", key)
	}
	return s, nil
}

// Dummy value deserializer: converts string back to string pointer
func dummyValueDeserializer(serialized string, value any) error {
	ptr := value.(*string)
	*ptr = serialized
	return nil
}

func TestConvertKeyValueDataToTyped(t *testing.T) {
	typedStorage := &TypedAtomicStorageImpl{
		keySerializer:     dummyKeySerializer,
		valueDeserializer: dummyValueDeserializer,
		valueType:         reflect.TypeOf(""),
	}

	testCases := []struct {
		name          string
		conditionKeys []any
		keyValueData  []KeyValueData
		expected      []TypedKeyValueData
		expectedErr   bool
	}{
		{
			name:          "Empty condition keys and data",
			conditionKeys: []any{},
			keyValueData:  []KeyValueData{},
			expected:      []TypedKeyValueData{},
			expectedErr:   false,
		},
		{
			name:          "Key in condition matches key-value data",
			conditionKeys: []any{"key1"},
			keyValueData: []KeyValueData{
				{Key: "key1", Value: "value1", Present: true},
			},
			expected: []TypedKeyValueData{
				{Key: "key1", Value: stringPtr("value1"), Present: true},
			},
			expectedErr: false,
		},
		{
			name:          "Key in condition doesn't match key-value data",
			conditionKeys: []any{"key1"},
			keyValueData: []KeyValueData{
				{Key: "key2", Value: "value2", Present: true},
			},
			expected: []TypedKeyValueData{
				{Key: "key1", Present: false},
			},
			expectedErr: false,
		},
		{
			name:          "Key in condition matches missing value in data",
			conditionKeys: []any{"key1"},
			keyValueData: []KeyValueData{
				{Key: "key1", Value: "", Present: false},
			},
			expected: []TypedKeyValueData{
				{Key: "key1", Present: false},
			},
			expectedErr: false,
		},
		{
			name:          "Multiple condition keys with mixed results",
			conditionKeys: []any{"key1", "key2"},
			keyValueData: []KeyValueData{
				{Key: "key1", Value: "value1", Present: true},
				{Key: "key3", Value: "value3", Present: true},
			},
			expected: []TypedKeyValueData{
				{Key: "key1", Value: stringPtr("value1"), Present: true},
				{Key: "key2", Present: false},
			},
			expectedErr: false,
		},
		{
			name:          "Error in key serialization",
			conditionKeys: []any{123}, // Invalid key for dummyKeySerializer
			keyValueData: []KeyValueData{
				{Key: "key1", Value: "value1", Present: true},
			},
			expected:    nil,
			expectedErr: true,
		},
		{
			name:          "Error in value deserialization",
			conditionKeys: []any{"key1"},
			keyValueData: []KeyValueData{
				{Key: "key1", Value: "", Present: true},
			},
			expected:    nil,
			expectedErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := typedStorage.convertKeyValueDataToTyped(tc.conditionKeys, tc.keyValueData)

			if tc.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

// Helper function to create a string pointer
func stringPtr(s string) *string {
	return &s
}

func TestFindKeyValueByKey(t *testing.T) {
	testCases := []struct {
		name          string
		keyValueData  []KeyValueData
		key           string
		expected      *KeyValueData
		expectedFound bool
	}{
		{
			name: "Key exists in the data",
			keyValueData: []KeyValueData{
				{Key: "key1", Value: "value1", Present: true},
				{Key: "key2", Value: "value2", Present: false},
			},
			key:           "key1",
			expected:      &KeyValueData{Key: "key1", Value: "value1", Present: true},
			expectedFound: true,
		},
		{
			name: "Key does not exist in the data",
			keyValueData: []KeyValueData{
				{Key: "key1", Value: "value1", Present: true},
				{Key: "key2", Value: "value2", Present: false},
			},
			key:           "key3",
			expected:      nil,
			expectedFound: false,
		},
		{
			name:          "Empty keyValueData",
			keyValueData:  []KeyValueData{},
			key:           "key1",
			expected:      nil,
			expectedFound: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, found := findKeyValueByKey(tc.keyValueData, tc.key)
			assert.Equal(t, tc.expectedFound, found)

			if tc.expectedFound {
				assert.Equal(t, tc.expected, result)
			} else {
				assert.Nil(t, result)
			}
		})
	}
}
