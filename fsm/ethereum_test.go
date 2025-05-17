package fsm

import (
	"encoding/hex"
	"fmt"
	"github.com/canopy-network/canopy/lib"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/anypb"
	"math/big"
	"strings"
	"testing"
	"time"
)

func TestRLPToSendTxEtherscan(t *testing.T) {
	tests := []struct {
		name     string
		expected *lib.Transaction
	}{
		{
			name: "tx 1 from etherscan", // https://etherscan.io/tx/0x613b3c4675e40322d804e14d1dec3adee6313685b3451087d54a39dfec037389
			expected: &lib.Transaction{
				MessageType: MessageSendName,
				Msg: msg(t, &MessageSend{
					FromAddress: h3x(t, `356B48C7aE4c5E94c2D0cDa787F1e7C129FA535B`),
					ToAddress:   h3x(t, `c8657C64C11f01Bc8bC46924f21756Dafcdc82bf`),
					Amount:      7269, // downscaled to 6 decimals
				}),
				Signature: &lib.Signature{
					PublicKey: h3x(t, `fa838c4b6796b893275685648e93dd55592740d08400c48c84e73f1e4b39a8cffd2a818c6413e089a7d6bfb08bfd1df57f0f8ca766ab2c5b5b1e8bfc45b36e5b`),
					Signature: h3x(t, `02f872011b8404b571c08501dd6ee680825a3c94c8657c64c11f01bc8bc46924f21756dafcdc82bf8719d3fa61c8f9a880c001a05546ba2e5fef1a43c9eecb7f42260d1e9f2e9f983ce096e705e06330720e8c53a0197f62b1d845e2f8e06e37436ce557630bc6c86044158351c2cc43d8c35172aa`),
				},
				CreatedHeight: 27, // nonce
				Time:          uint64(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(23_100) * time.Second).Unix()),
				Fee:           DownscaleTo6Decimals(big.NewInt(8010000000 * 23_100)),
				Memo:          RLPIndicator,
				NetworkId:     uint64(0),
				ChainId:       1,
			},
		},
		{
			name: "tx 2 from etherscan", // https://etherscan.io/tx/0xd6c7c2d871d49818ce67df267028017c9ec9d10e223786b04660047ae66e159d
			expected: &lib.Transaction{
				MessageType: MessageSendName,
				Msg: msg(t, &MessageSend{
					FromAddress: h3x(t, `53EDd10467eA18E7912F6eA8b6364090bC917801`),
					ToAddress:   h3x(t, `14215B76D395611E4BCfba9A0EB799eb007dC828`),
					Amount:      173806, // downscaled to 6 decimals
				}),
				Signature: &lib.Signature{
					PublicKey: h3x(t, `b0484eacfaf002b22005dfebe29fd6eca415c116b562560ac6830cfe681e55d2f958c47a9b64e22b832e42fa3054a59a3e2f9a9ea8d15adb154225ce1cf48e99`),
					Signature: h3x(t, `02f87301448404b571c08501dd6ee680825a3c9414215b76d395611e4bcfba9a0eb799eb007dc8288802697bf0bac9407b80c001a08a7655014e3aa95644ac3e5f0c24ef15d7e42735d995de7d6eb5c0925230b9f2a02dc8c989b2bdee2ea5efe06f5d9a97af64486234a0ed8c606a909ada28ec05d4`),
				},
				CreatedHeight: 68, // nonce
				Time:          uint64(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(23_100) * time.Second).Unix()),
				Fee:           DownscaleTo6Decimals(big.NewInt(8010000000 * 23_100)),
				Memo:          RLPIndicator,
				NetworkId:     uint64(0),
				ChainId:       1,
			},
		}, {
			name: "tx 3 from etherscan", // https://etherscan.io/tx/0xc174da04a9419e84c1f151ff521e1f5d2a184196213deaf326144a9cdb676ddd
			expected: &lib.Transaction{
				MessageType: MessageSendName,
				Msg: msg(t, &MessageSend{
					FromAddress: h3x(t, `685d1b3d33091A28b147F20688D5303E2d3a1752`),
					ToAddress:   h3x(t, `8e775eBAD58241F3444d140846116EF80a6377bC`),
					Amount:      38570, // downscaled to 6 decimals
				}),
				Signature: &lib.Signature{
					PublicKey: h3x(t, `978dae04883323500b9546ec78e1bf347164215b04ed0c3df4c1aad741253cd3425421ce29c9343c4b1707775001f501e9c5344ec1c149fde79e82bce528753d`),
					Signature: h3x(t, `02f87201808439d10680850170f4ccfb825208948e775ebad58241f3444d140846116ef80a6377bc87890737f77ff22880c001a00114c33808435ac0be5d34024b68aa23046b2d14f649dd423af9a5af8d6df960a060d4c352b13b552f48ce31fe4f4e7870866ea66217f6027ff049f5b28fd997bd`),
				},
				CreatedHeight: 0, // nonce
				Time:          uint64(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(21_000) * time.Second).Unix()),
				Fee:           DownscaleTo6Decimals(big.NewInt(6190058747 * 21_000)),
				Memo:          RLPIndicator,
				NetworkId:     uint64(0),
				ChainId:       1,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// execute the function call
			got, err := RLPToCanopyTransaction(test.expected.Signature.Signature)
			require.NoError(t, err)
			j1, err := lib.MarshalJSONIndentString(got)
			require.NoError(t, err)
			j2, err := lib.MarshalJSONIndentString(test.expected)
			require.NoError(t, err)
			require.Equal(t, test.expected, got, fmt.Sprintf("%s\nt%s", j1, j2))
		})
	}
}

func TestRLPToSendTxDynamic(t *testing.T) {
	// create the identity / chain fields
	privKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	pub := crypto.FromECDSAPub(&privKey.PublicKey)[1:]
	from := crypto.PubkeyToAddress(privKey.PublicKey)
	chainID := big.NewInt(int64(CanopyIdsToEVMChainId(1, 1)))
	// create the transaction fields
	to := common.HexToAddress("0x000000000000000000000000000000000000dead")
	gas := uint64(21_000)
	gasPrice := big.NewInt(10_000_000_000_000) // manually 10 to 18 decimals
	amount := UpscaleTo18Decimals(20)          // auto-upscaled to 18 decimals
	nonce := uint64(3)
	data := []byte{}
	// create the expected transaction
	expected := &lib.Transaction{
		MessageType: MessageSendName,
		Msg: msg(t, &MessageSend{
			FromAddress: from.Bytes(),
			ToAddress:   to.Bytes(),
			Amount:      20, // downscaled to 6 decimals
		}),
		Signature: &lib.Signature{
			PublicKey: pub,
		},
		CreatedHeight: 3, // nonce
		Time:          uint64(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(21_000) * time.Second).Unix()),
		Fee:           10 * gas, // downscaled to 6 decimals
		Memo:          RLPIndicator,
		NetworkId:     uint64(1),
		ChainId:       1,
	}
	tests := []struct {
		name  string
		input func() *types.Transaction
	}{
		{
			name: "EIP-155 Tx",
			input: func() *types.Transaction {
				tx := types.NewTransaction(nonce, to, amount, gas, gasPrice, data)
				signedTx, e := types.SignTx(tx, types.NewEIP155Signer(chainID), privKey)
				if e != nil {
					panic(e)
				}
				return signedTx
			},
		},
		{
			name: "Access List Tx",
			input: func() *types.Transaction {
				signedTx, e := types.SignNewTx(privKey, types.LatestSignerForChainID(chainID), &types.AccessListTx{
					ChainID:  chainID,
					Nonce:    nonce,
					GasPrice: gasPrice,
					Gas:      gas,
					To:       &to,
					Value:    amount,
					Data:     data,
					AccessList: types.AccessList{
						{Address: to, StorageKeys: []common.Hash{}},
					},
				})
				if e != nil {
					panic(e)
				}
				return signedTx
			},
		}, {
			name: "Dynamic fee Tx",
			input: func() *types.Transaction {
				signedTx, e := types.SignNewTx(privKey, types.LatestSignerForChainID(chainID), &types.DynamicFeeTx{
					ChainID:   chainID,
					Nonce:     nonce,
					GasTipCap: big.NewInt(1e9), // maxPriorityFeePerGas
					GasFeeCap: gasPrice,        // maxFeePerGas
					Gas:       gas,
					To:        &to,
					Value:     amount,
					Data:      data,
				})
				if e != nil {
					panic(e)
				}
				return signedTx
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// marshal the transaction to (rlp) binary
			rlpBytes, e := test.input().MarshalBinary()
			require.NoError(t, e)
			// add to expected
			expected.Signature.Signature = rlpBytes
			// execute the function call
			got, e := RLPToCanopyTransaction(expected.Signature.Signature)
			require.NoError(t, err)
			j1, e := lib.MarshalJSONIndentString(got)
			require.NoError(t, e)
			j2, e := lib.MarshalJSONIndentString(expected)
			require.NoError(t, e)
			require.Equal(t, expected, got, fmt.Sprintf("%s\nt%s", j1, j2))
		})
	}
}

func TestRLPToSendTxERC20(t *testing.T) {
	// create the identity / chain fields
	privKey, err := crypto.GenerateKey()
	require.NoError(t, err)
	pub := crypto.FromECDSAPub(&privKey.PublicKey)[1:]
	from := crypto.PubkeyToAddress(privKey.PublicKey)
	chainID := big.NewInt(int64(CanopyIdsToEVMChainId(1, 1)))
	// create the transaction fields
	contractAddress := common.HexToAddress(CNPYContractAddress)
	gas := uint64(21_000)
	gasPrice := UpscaleTo18Decimals(10)
	amount := big.NewInt(20)
	nonce := uint64(3)
	// create the data field
	const abiJSON = `
	[{"name": "transfer", "type": "function",
		"inputs": [
    		{"name": "recipient", "type": "address"},
    		{"name": "amount", "type": "uint256"}
  		],
  		"outputs": [
    		{"name": "", "type": "bool"}
  		]
	}]`

	// parse the ABI
	parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
	require.NoError(t, err)

	// Define inputs
	to := common.HexToAddress("0x000000000000000000000000000000000000dead")

	// Encode the function call
	data, err := parsedABI.Pack("transfer", to, amount)
	require.NoError(t, err)
	// create the expected transaction
	expected := &lib.Transaction{
		MessageType: MessageSendName,
		Msg: msg(t, &MessageSend{
			FromAddress: from.Bytes(),
			ToAddress:   to.Bytes(),
			Amount:      amount.Uint64(),
		}),
		Signature: &lib.Signature{
			PublicKey: pub,
		},
		CreatedHeight: 3, // nonce
		Time:          uint64(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(21_000) * time.Second).Unix()),
		Fee:           DownscaleTo6Decimals(gasPrice) * gas,
		Memo:          RLPIndicator,
		NetworkId:     uint64(1),
		ChainId:       1,
	}
	tests := []struct {
		name  string
		input func() *types.Transaction
	}{
		{
			name: "EIP-155 Tx",
			input: func() *types.Transaction {
				tx := types.NewTransaction(nonce, contractAddress, big.NewInt(0), gas, gasPrice, data)
				signedTx, e := types.SignTx(tx, types.NewEIP155Signer(chainID), privKey)
				if e != nil {
					panic(e)
				}
				return signedTx
			},
		},
		{
			name: "Access List Tx",
			input: func() *types.Transaction {
				signedTx, e := types.SignNewTx(privKey, types.LatestSignerForChainID(chainID), &types.AccessListTx{
					ChainID:  chainID,
					Nonce:    nonce,
					GasPrice: gasPrice,
					Gas:      gas,
					To:       &contractAddress,
					Value:    big.NewInt(0),
					Data:     data,
					AccessList: types.AccessList{
						{Address: to, StorageKeys: []common.Hash{}},
					},
				})
				if e != nil {
					panic(e)
				}
				return signedTx
			},
		}, {
			name: "Dynamic fee Tx",
			input: func() *types.Transaction {
				signedTx, e := types.SignNewTx(privKey, types.LatestSignerForChainID(chainID), &types.DynamicFeeTx{
					ChainID:   chainID,
					Nonce:     nonce,
					GasTipCap: big.NewInt(1e9), // maxPriorityFeePerGas
					GasFeeCap: gasPrice,        // maxFeePerGas
					Gas:       gas,
					To:        &contractAddress,
					Value:     big.NewInt(0),
					Data:      data,
				})
				if e != nil {
					panic(e)
				}
				return signedTx
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// marshal the transaction to (rlp) binary
			rlpBytes, e := test.input().MarshalBinary()
			require.NoError(t, e)
			// add to expected
			expected.Signature.Signature = rlpBytes
			// execute the function call
			got, e := RLPToCanopyTransaction(expected.Signature.Signature)
			require.NoError(t, err)
			j1, e := lib.MarshalJSONIndentString(got)
			require.NoError(t, e)
			j2, e := lib.MarshalJSONIndentString(expected)
			require.NoError(t, e)
			require.Equal(t, expected, got, fmt.Sprintf("%s\nt%s", j1, j2))
		})
	}
}

func msg(t *testing.T, send *MessageSend) *anypb.Any {
	a, err := lib.NewAny(send)
	require.NoError(t, err)
	return a
}

func h3x(t *testing.T, s string) []byte {
	b, err := hex.DecodeString(s)
	require.NoError(t, err)
	return b
}
