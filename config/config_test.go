package config

import (
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCustomSubMap(t *testing.T) {
	var config = viper.New()
	config.Set("outer.inner", "inner-value")
	config.SetDefault("outer.inner-default", "inner-default-value")

	var sub = SubWithDefault(config, "outer")

	assert.Equal(t, "inner-value", sub.Get("inner"))
	assert.Equal(t, "inner-default-value", sub.Get("inner-default"))
}

func TestCustomSubSingleValue(t *testing.T) {
	var config = viper.New()
	config.SetDefault("outer.inner-default", "inner-default-value")

	var sub = SubWithDefault(config, "outer")

	assert.Equal(t, "inner-default-value", sub.Get("inner-default"))
}

func TestCustomSubNoValue(t *testing.T) {
	var config = viper.New()
	config.SetDefault("outer", "inner-default")

	var sub = SubWithDefault(config, "outer")

	assert.NotNil(t, sub)
	assert.Equal(t, nil, sub.Get("inner-default"))
}

func TestCustomSubNoKey(t *testing.T) {
	var config = viper.New()

	var sub = SubWithDefault(config, "unknown")

	assert.Nil(t, sub)
}

func TestReadConfigFromJsonString(t *testing.T) {
	var config = viper.New()
	var jsonConfig = `
{
  "object": {
  	  "field": "value"
  },
  "array": [ "item-1", "item-2" ],
  "string-key": "string-value",
  "int-key": 42
}`

	ReadConfigFromJsonString(config, jsonConfig)

	assert.Equal(t, map[string]interface{}{"field": "value"}, config.Get("object"))
	assert.Equal(t, "value", config.Get("object.field"))
	assert.Equal(t, []interface{}{"item-1", "item-2"}, config.Get("array"))
	assert.Equal(t, "string-value", config.Get("string-key"))
	assert.Equal(t, 42, config.GetInt("int-key"))
}
