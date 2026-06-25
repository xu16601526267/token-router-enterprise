package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestActivateSupplierRoutePreferenceManualUpsertAndDisable(t *testing.T) {
	truncateTables(t)

	supplier := &Supplier{
		Name:   "manual-route-preference-supplier",
		Type:   SupplierTypeThirdParty,
		Status: common.ChannelStatusEnabled,
	}
	require.NoError(t, supplier.Insert())

	invalidCases := []struct {
		name  string
		input SupplierRoutePreferenceActivateInput
		err   string
	}{
		{
			name: "missing supplier",
			input: SupplierRoutePreferenceActivateInput{
				WeightPercent: 40,
				Reason:        "operator quality incident",
			},
			err: "supplier_id is required",
		},
		{
			name: "zero weight",
			input: SupplierRoutePreferenceActivateInput{
				SupplierId:    supplier.Id,
				WeightPercent: 0,
				Reason:        "operator quality incident",
				EffectiveFrom: 100,
				EffectiveTo:   200,
				OperatorNote:  "manual watch",
			},
			err: "weight_percent must be between 1 and 200",
		},
		{
			name: "weight above cap",
			input: SupplierRoutePreferenceActivateInput{
				SupplierId:    supplier.Id,
				WeightPercent: SupplierRoutePreferenceMaxWeightPercent + 1,
				Reason:        "operator quality incident",
				EffectiveFrom: 100,
				EffectiveTo:   200,
				OperatorNote:  "manual watch",
			},
			err: "weight_percent must be between 1 and 200",
		},
		{
			name: "empty reason",
			input: SupplierRoutePreferenceActivateInput{
				SupplierId:    supplier.Id,
				WeightPercent: 40,
				Reason:        " ",
				EffectiveFrom: 100,
				EffectiveTo:   200,
				OperatorNote:  "manual watch",
			},
			err: "reason is required",
		},
		{
			name: "invalid window",
			input: SupplierRoutePreferenceActivateInput{
				SupplierId:    supplier.Id,
				WeightPercent: 40,
				Reason:        "operator quality incident",
				EffectiveFrom: 200,
				EffectiveTo:   100,
				OperatorNote:  "manual watch",
			},
			err: "effective_to must be greater than effective_from",
		},
	}
	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ActivateSupplierRoutePreference(tc.input, 7)
			require.ErrorContains(t, err, tc.err)
		})
	}

	now := common.GetTimestamp()
	active, err := ActivateSupplierRoutePreference(SupplierRoutePreferenceActivateInput{
		SupplierId:    supplier.Id,
		WeightPercent: SupplierRoutePreferenceMaxWeightPercent,
		Reason:        " operator quality incident ",
		EffectiveFrom: now - 10,
		EffectiveTo:   now + 3600,
		OperatorNote:  " manual watch ",
	}, 7)
	require.NoError(t, err)
	require.Equal(t, supplier.Id, active.SupplierId)
	require.Equal(t, 0, active.SourcePostureRecommendationId)
	require.Equal(t, SupplierRoutePreferenceStatusActive, active.Status)
	require.Equal(t, SupplierRoutePreferenceMaxWeightPercent, active.WeightPercent)
	require.Equal(t, "operator quality incident", active.Reason)
	require.Equal(t, "manual watch", active.OperatorNote)
	require.Equal(t, now-10, active.EffectiveFrom)
	require.Equal(t, now+3600, active.EffectiveTo)
	require.Greater(t, active.ActivatedAt, int64(0))
	require.Equal(t, 7, active.ActivatedBy)
	require.Equal(t, 0, active.DisabledBy)

	firstID := active.Id
	firstCreatedAt := active.CreatedAt
	active, err = ActivateSupplierRoutePreference(SupplierRoutePreferenceActivateInput{
		SupplierId:    supplier.Id,
		WeightPercent: 75,
		Reason:        "latency recovered partially",
		EffectiveFrom: now - 5,
		OperatorNote:  "watch next probe",
	}, 8)
	require.NoError(t, err)
	require.Equal(t, firstID, active.Id)
	require.Equal(t, firstCreatedAt, active.CreatedAt)
	require.Equal(t, 0, active.SourcePostureRecommendationId)
	require.Equal(t, 75, active.WeightPercent)
	require.Equal(t, "latency recovered partially", active.Reason)
	require.Equal(t, "watch next probe", active.OperatorNote)
	require.Equal(t, 8, active.ActivatedBy)
	require.Equal(t, int64(0), active.EffectiveTo)

	disabled, err := DisableSupplierRoutePreference(supplier.Id, 9, " restore baseline ")
	require.NoError(t, err)
	require.Equal(t, firstID, disabled.Id)
	require.Equal(t, SupplierRoutePreferenceStatusDisabled, disabled.Status)
	require.Equal(t, SupplierRoutePreferenceBaselineWeightPercent, disabled.WeightPercent)
	require.Equal(t, 0, disabled.SourcePostureRecommendationId)
	require.GreaterOrEqual(t, disabled.EffectiveTo, now)
	require.Greater(t, disabled.DisabledAt, int64(0))
	require.Equal(t, 9, disabled.DisabledBy)
	require.Equal(t, "restore baseline", disabled.OperatorNote)

	_, err = GetActiveSupplierRoutePreferenceBySupplierID(supplier.Id)
	require.Error(t, err)
}

func TestActivateSupplierRoutePreferenceRequiresEnabledSupplier(t *testing.T) {
	truncateTables(t)

	supplier := &Supplier{
		Name:   "manual-route-preference-disabled-supplier",
		Type:   SupplierTypeThirdParty,
		Status: common.ChannelStatusManuallyDisabled,
	}
	require.NoError(t, supplier.Insert())

	_, err := ActivateSupplierRoutePreference(SupplierRoutePreferenceActivateInput{
		SupplierId:    supplier.Id,
		WeightPercent: 40,
		Reason:        "operator quality incident",
		EffectiveFrom: common.GetTimestamp() - 10,
		OperatorNote:  "manual watch",
	}, 7)
	require.ErrorContains(t, err, "supplier must be enabled")

	_, err = GetActiveSupplierRoutePreferenceBySupplierID(supplier.Id)
	require.Error(t, err)
}
