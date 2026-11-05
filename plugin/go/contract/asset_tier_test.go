package contract

import "testing"

func TestValidateAssetTierRange(t *testing.T) {
	cases := []struct {
		name    string
		tier    uint32
		wantErr bool
	}{
		{"tier 0 (CNPY) valid", 0, false},
		{"tier 1 valid", 1, false},
		{"tier 2 valid", 2, false},
		{"tier 3 (Restricted) valid, upper bound", 3, false},
		{"tier 4 (Blacklisted) rejected -- never stored, see PrefixAssetTier", 4, true},
		{"tier 5 rejected", 5, true},
		{"tier wrapped from oversized uint32 rejected, not silently cast", 300, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateAssetTierRange(tc.tier)
			if (err != nil) != tc.wantErr {
				t.Errorf("validateAssetTierRange(%d) error = %v, wantErr %v", tc.tier, err, tc.wantErr)
			}
		})
	}
}
