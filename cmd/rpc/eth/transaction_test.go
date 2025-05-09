package eth

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
)

func TestNewEthereumTransaction(t *testing.T) {
	// Create a simple transaction
	tx := types.NewTransaction(
		1, // nonce
		common.HexToAddress("0x1234567890123456789012345678901234567890"), // to
		big.NewInt(1000000000000000000),                                   // value (1 ETH)
		21000,                                                             // gas limit
		big.NewInt(1000000000),                                            // gas price (1 gwei)
		[]byte("test data"),                                               // data
	)

	// Create the wrapper
	ethTx := NewEthereumTransaction(tx)

	// Verify the transaction was properly wrapped
	assert.Equal(t, tx, ethTx.tx, "Transaction should be correctly stored in the wrapper")
}

func TestFrom(t *testing.T) {
	// Create a private key for signing
	privateKey, err := crypto.GenerateKey()
	assert.NoError(t, err)

	// Get the address from the private key
	address := crypto.PubkeyToAddress(privateKey.PublicKey)

	// Create a transaction with ChainID 1 (mainnet)
	tx := types.NewTransaction(
		0,
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
		big.NewInt(0),
		21000,
		big.NewInt(1000000000),
		[]byte{},
	)

	// Sign the transaction
	signer := types.NewEIP155Signer(big.NewInt(1))
	signedTx, err := types.SignTx(tx, signer, privateKey)
	assert.NoError(t, err)

	// Create the wrapper
	ethTx := NewEthereumTransaction(signedTx)

	// Test From() method
	fromAddr, err := ethTx.From()
	assert.NoError(t, err)
	assert.Equal(t, address.Hex(), fromAddr, "From address should match the signer's address")
}

func TestFromError(t *testing.T) {
	// Create an unsigned transaction (no signature)
	tx := types.NewTransaction(
		0,
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
		big.NewInt(0),
		21000,
		big.NewInt(1000000000),
		[]byte{},
	)

	ethTx := NewEthereumTransaction(tx)

	// This should fail because the transaction is not signed
	_, err := ethTx.From()
	assert.Error(t, err, "From should return an error for unsigned transactions")
}

func TestIsSelfSend(t *testing.T) {
	// Create a private key for signing
	privateKey, err := crypto.GenerateKey()
	assert.NoError(t, err)

	// Get the address from the private key
	address := crypto.PubkeyToAddress(privateKey.PublicKey)

	// Test cases:
	testCases := []struct {
		name      string
		toAddress common.Address
		expected  bool
	}{
		{
			name:      "Self-send transaction",
			toAddress: address,
			expected:  true,
		},
		{
			name:      "Regular transaction",
			toAddress: common.HexToAddress("0x3333333333333333333333333333333333333333"),
			expected:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a transaction
			tx := types.NewTransaction(
				0,
				tc.toAddress,
				big.NewInt(0),
				21000,
				big.NewInt(1000000000),
				[]byte{},
			)

			// Sign the transaction
			signer := types.NewEIP155Signer(big.NewInt(1))
			signedTx, err := types.SignTx(tx, signer, privateKey)
			assert.NoError(t, err)

			// Create the wrapper
			ethTx := NewEthereumTransaction(signedTx)

			// Test IsSelfSend method
			isSelfSend, err := ethTx.IsSelfSend()
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, isSelfSend)
		})
	}
}

func TestIsSelfSendError(t *testing.T) {
	// Create an unsigned transaction
	tx := types.NewTransaction(
		0,
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
		big.NewInt(0),
		21000,
		big.NewInt(1000000000),
		[]byte{},
	)

	ethTx := NewEthereumTransaction(tx)

	// IsSelfSend should fail because it calls From() which will fail
	_, err := ethTx.IsSelfSend()
	assert.Error(t, err, "IsSelfSend should return an error for unsigned transactions")
}

func TestIsSelfSendWithNilToAddress(t *testing.T) {
	// Create a private key for signing
	privateKey, err := crypto.GenerateKey()
	assert.NoError(t, err)

	tx := types.NewTx(&types.LegacyTx{
		Nonce:    0,
		GasPrice: big.NewInt(0),
	})

	// Sign the transaction
	signer := types.NewEIP155Signer(big.NewInt(1))
	signedTx, err := types.SignTx(tx, signer, privateKey)
	assert.NoError(t, err)

	// Create the wrapper
	ethTx := NewEthereumTransaction(signedTx)

	// This should handle the nil To address case
	isSelfSend, err := ethTx.IsSelfSend()
	assert.NoError(t, err)
	assert.False(t, isSelfSend, "Contract creation cannot be a self-send")
}

func TestData(t *testing.T) {
	testData := []byte("test transaction data")

	// Create a transaction with data
	tx := types.NewTransaction(
		0,
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
		big.NewInt(0),
		21000,
		big.NewInt(1000000000),
		testData,
	)

	ethTx := NewEthereumTransaction(tx)

	// Test the Data method
	data := ethTx.Data()
	assert.Equal(t, testData, data, "Transaction data should match the input data")
}

func TestDataEmpty(t *testing.T) {
	// Create a transaction with empty data
	tx := types.NewTransaction(
		0,
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
		big.NewInt(0),
		21000,
		big.NewInt(1000000000),
		[]byte{},
	)

	ethTx := NewEthereumTransaction(tx)

	// Test the Data method with empty data
	data := ethTx.Data()
	assert.Empty(t, data, "Transaction data should be empty")
}
