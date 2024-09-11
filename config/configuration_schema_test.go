package config

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_getLeafNodeKey(t *testing.T) {
	type args struct {
		key string
	}
	tests := []struct {
		name  string
		args  args
		want  bool
		want1 string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := isLeafNodeKey(tt.args.key)
			if got != tt.want {
				t.Errorf("isLeafNodeKey() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("isLeafNodeKey() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestGetSchemaConfiguration(t *testing.T) {
	schemaDetails, err := GetConfigurationSchema()
	for _, element := range schemaDetails {
		if element.Name == "blockchain_network_selected" {
			assert.Equal(t, element.DefaultValue, "local")
		}
	}
	assert.Nil(t, err)
}
