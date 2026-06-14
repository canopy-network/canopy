package tutorial

import (
    "testing"
    contract "github.com/canopy-network/go-plugin/contract"
)

func TestFundMetaMaskAddr(t *testing.T) {
    validatorAddr := "e7c7dad131a03f7ea0cc09a637ad096eb3495f77"
    validatorKey, err := keystoreGetKey(validatorAddr, "")
    if err != nil { t.Fatalf("key: %v", err) }
    h, _ := getHeight()
    msg := &contract.MessageSend{
        FromAddress: hexDecode(validatorAddr),
        ToAddress:   hexDecode("0790d558482cc8495962e8996e5e6311c5889fef"),
        Amount:      1000000000000,
    }
    hash := submitSendTx(t, validatorKey, msg, h)
    if err := waitForTx(validatorAddr, hash, 60*1e9); err != nil {
        t.Fatalf("send: %v", err)
    }
    t.Logf("Funded! balance check: %d", getBalance("0790d558482cc8495962e8996e5e6311c5889fef"))
}
