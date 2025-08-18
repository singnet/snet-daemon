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

// Dummy value deserializer: converts string back to *string, but errors on empty input
func dummyValueDeserializer(serialized string, value any) error {
	if serialized == "" {
		return fmt.Errorf("empty value")
	}
	ptr, ok := value.(*string)
	if !ok {
		return fmt.Errorf("expected *string, got %T", value)
	}
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

func TestRemoveKeyValuePrefix_RemovesOnlyLeadingPrefix(t *testing.T) {
	s := &PrefixedAtomicStorage{keyPrefix: "app"}

	in := []KeyValueData{
		{Key: "app/user", Value: "v1", Present: true},
		{Key: "app/app/user", Value: "v2", Present: true},         // "app/" appears again inside the key
		{Key: "app/user/app-profile", Value: "v3", Present: true}, // "app/" appears later in the key
	}
	got := s.removeKeyValuePrefix(in)

	// Expect ONLY the leading "app/" to be removed
	want := []KeyValueData{
		{Key: "user", Value: "v1", Present: true},
		{Key: "app/user", Value: "v2", Present: true},         // inner "app/" should remain
		{Key: "user/app-profile", Value: "v3", Present: true}, // inner "app/" should remain
	}

	assert.Equal(t, want, got)
}

func TestRemoveKeyValuePrefix_NoChangeWhenNoLeadingPrefix(t *testing.T) {
	s := &PrefixedAtomicStorage{keyPrefix: "app"}

	in := []KeyValueData{
		{Key: "xapp/user", Value: "v1", Present: true}, // contains "app/" but not as a leading prefix
		{Key: "noapp/user", Value: "v2", Present: true},
		{Key: "application/user", Value: "v3", Present: true},
	}
	got := s.removeKeyValuePrefix(in)

	// Expect no changes (no leading "app/")
	want := []KeyValueData{
		{Key: "xapp/user", Value: "v1", Present: true},
		{Key: "noapp/user", Value: "v2", Present: true},
		{Key: "application/user", Value: "v3", Present: true},
	}

	assert.Equal(t, want, got)
}

func TestRemoveKeyValuePrefix_TrailingSlashInPrefix(t *testing.T) {
	s := &PrefixedAtomicStorage{keyPrefix: "app/"} // trailing slash

	in := []KeyValueData{
		{Key: "app/user", Value: "v1", Present: true},
		{Key: "app/app/user", Value: "v2", Present: true}, // inner "app/" should remain
	}
	got := s.removeKeyValuePrefix(in)

	want := []KeyValueData{
		{Key: "user", Value: "v1", Present: true},
		{Key: "app/user", Value: "v2", Present: true},
	}
	assert.Equal(t, want, got)
}

func TestRemoveKeyValuePrefix_EmptyPrefixSymmetry(t *testing.T) {
	s := &PrefixedAtomicStorage{keyPrefix: ""}

	in := []KeyValueData{
		{Key: "/user/1", Value: "v", Present: true}, // how a key may look when joined with an empty prefix
		{Key: "user/1", Value: "v", Present: true},  // no leading "/"
	}
	got := s.removeKeyValuePrefix(in)

	want := []KeyValueData{
		{Key: "user/1", Value: "v", Present: true},
		{Key: "user/1", Value: "v", Present: true},
	}
	assert.Equal(t, want, got)
}

func TestRemoveKeyValuePrefix_Idempotent(t *testing.T) {
	s := &PrefixedAtomicStorage{keyPrefix: "app"}

	in := []KeyValueData{
		{Key: "app/a/b", Value: "v", Present: true},
		{Key: "x/app/b", Value: "v", Present: true}, // no leading prefix
	}
	once := s.removeKeyValuePrefix(in)
	twice := s.removeKeyValuePrefix(once)
	assert.Equal(t, once, twice)
}

func TestPrefixRoundTrip_AppendThenRemove(t *testing.T) {
	s := &PrefixedAtomicStorage{keyPrefix: "app"}
	orig := []KeyValueData{
		{Key: "a/b", Value: "v1", Present: true},
		{Key: "c", Value: "v2", Present: false},
	}
	with := s.appendKeyValuePrefix(orig)
	back := s.removeKeyValuePrefix(with)
	assert.Equal(t, orig, back)
}
