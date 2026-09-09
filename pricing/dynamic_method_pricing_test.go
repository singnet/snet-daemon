package pricing

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDynamicMethodPrice_GetPriceType(t *testing.T) {
	priceType := DynamicMethodPrice{}

	assert.Equal(t, DYNAMIC_PRICING, priceType.GetPriceType())
	assert.Equal(t, "dynamic_pricing", priceType.GetPriceType())
}
