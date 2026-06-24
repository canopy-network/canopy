package canoliq

import "github.com/canopy-network/go-plugin/contract"

// unlockedAmount returns the cumulative CPLQ that should be unlocked for a
// vesting schedule at the given block height. Returns 0 before the cliff,
// linearly interpolates between start and end, and saturates at total_amount
// after end_height.
func unlockedAmount(s *contract.VestingSchedule, height uint64) uint64 {
	if s == nil || s.TotalAmount == 0 {
		return 0
	}
	if height < s.CliffHeight {
		return 0
	}
	if s.EndHeight <= s.StartHeight {
		// Degenerate schedule: anything past the cliff is fully vested.
		return s.TotalAmount
	}
	if height >= s.EndHeight {
		return s.TotalAmount
	}
	elapsed := height - s.StartHeight
	span := s.EndHeight - s.StartHeight
	return mulDiv(s.TotalAmount, elapsed, span)
}
