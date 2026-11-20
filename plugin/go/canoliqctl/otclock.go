package main

import (
	"fmt"
	"strings"

	"github.com/canopy-network/go-plugin/contract"
)

// parseOTCLockTier maps the CLI tier spelling onto the proto enum. Only the
// two funded tiers are accepted; OTC_LOCK_UNSPECIFIED is never reachable from
// the CLI so a typo becomes an error rather than a zero-valued tier.
func parseOTCLockTier(s string) (contract.OTCLockTier, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "90d", "90":
		return contract.OTCLockTier_OTC_LOCK_90D, nil
	case "120d", "120":
		return contract.OTCLockTier_OTC_LOCK_120D, nil
	default:
		return contract.OTCLockTier_OTC_LOCK_UNSPECIFIED, fmt.Errorf("invalid tier %q: want 90d or 120d", s)
	}
}

// cmdOTCLockCreate submits MessageOTCLockCreate, moving cCNPY into a term
// position and reserving its CPLQ reward from the program budget.
func cmdOTCLockCreate(args []string, gf globalFlags) error {
	if err := requireArgs(args, 3, commandUsages["otc-lock"]); err != nil {
		return err
	}
	signer, err := fetchSigner(gf.adminURL, args[0], gf.password)
	if err != nil {
		return err
	}
	amount, err := parseUint(args[1], "ccnpy-amount")
	if err != nil {
		return err
	}
	tier, err := parseOTCLockTier(args[2])
	if err != nil {
		return err
	}
	from, err := addrFromHex(signer.Address)
	if err != nil {
		return err
	}

	msg := &contract.MessageOTCLockCreate{
		FromAddress: from,
		CcnpyAmount: amount,
		Tier:        tier,
	}
	return submitAndReport(gf, signer, "otc_lock_create", msg, "otc-lock",
		fmt.Sprintf("from=%s ccnpy=%d tier=%s", signer.Address, amount, args[2]))
}

// cmdOTCLockClaim submits MessageOTCLockClaim, releasing a matured position's
// cCNPY together with its reserved CPLQ reward.
func cmdOTCLockClaim(args []string, gf globalFlags) error {
	if err := requireArgs(args, 2, commandUsages["otc-lock-claim"]); err != nil {
		return err
	}
	signer, err := fetchSigner(gf.adminURL, args[0], gf.password)
	if err != nil {
		return err
	}
	lockID, err := parseUint(args[1], "lock-id")
	if err != nil {
		return err
	}
	from, err := addrFromHex(signer.Address)
	if err != nil {
		return err
	}

	msg := &contract.MessageOTCLockClaim{FromAddress: from, LockId: lockID}
	return submitAndReport(gf, signer, "otc_lock_claim", msg, "otc-lock-claim",
		fmt.Sprintf("from=%s lock_id=%d", signer.Address, lockID))
}

// cmdOTCLockCancel submits MessageOTCLockCancel. This forfeits the entire CPLQ
// reward, so the confirmation line says so explicitly: it is a separate
// command from claim precisely so the forfeiture is never accidental.
func cmdOTCLockCancel(args []string, gf globalFlags) error {
	if err := requireArgs(args, 2, commandUsages["otc-lock-cancel"]); err != nil {
		return err
	}
	signer, err := fetchSigner(gf.adminURL, args[0], gf.password)
	if err != nil {
		return err
	}
	lockID, err := parseUint(args[1], "lock-id")
	if err != nil {
		return err
	}
	from, err := addrFromHex(signer.Address)
	if err != nil {
		return err
	}

	msg := &contract.MessageOTCLockCancel{FromAddress: from, LockId: lockID}
	return submitAndReport(gf, signer, "otc_lock_cancel", msg, "otc-lock-cancel (forfeits the CPLQ reward)",
		fmt.Sprintf("from=%s lock_id=%d", signer.Address, lockID))
}
