package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSupplierRoutePreferenceSelectionWeight(t *testing.T) {
	preferences := map[int]int{
		10: SupplierRoutePreferenceDowngradeWeightPercent,
		20: 0,
		30: SupplierRoutePreferenceMaxWeightPercent,
		40: SupplierRoutePreferenceMaxWeightPercent + 50,
	}

	require.Equal(t, 25, supplierRoutePreferenceSelectionWeight(100, 10, false, preferences))
	require.Equal(t, 100, supplierRoutePreferenceSelectionWeight(100, 99, false, preferences))
	require.Equal(t, 25, supplierRoutePreferenceSelectionWeight(0, 10, true, preferences))
	require.Equal(t, 100, supplierRoutePreferenceSelectionWeight(0, 99, true, preferences))
	require.Equal(t, 0, supplierRoutePreferenceSelectionWeight(100, 20, false, preferences))
	require.Equal(t, 200, supplierRoutePreferenceSelectionWeight(100, 30, false, preferences))
	require.Equal(t, 200, supplierRoutePreferenceSelectionWeight(100, 40, false, preferences))
	require.Equal(t, 200, supplierRoutePreferenceSelectionWeight(0, 30, true, preferences))
	require.Equal(t, 1, supplierRoutePreferenceSelectionWeight(1, 10, false, preferences))
	require.Equal(t, 0, supplierRoutePreferenceSelectionWeight(0, 10, false, preferences))
}
