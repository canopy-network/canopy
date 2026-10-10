package fsm

import (
	"encoding/json"
	"fmt"
	"github.com/canopy-network/canopy/lib"
	"github.com/stretchr/testify/require"
	"os"
	"sort"
	"testing"
)

func TestNewFromGenesisFile(t *testing.T) {
	const dataDirPath = "./"
	tests := []struct {
		name     string
		detail   string
		input    GenesisState
		expected GenesisState
	}{
		{
			name:     "complete",
			detail:   "the complete genesis file testing",
			input:    *newTestGenesisState(t),
			expected: *newTestValidateGenesisState(t),
		},
		{
			name:   "accounts",
			detail: "the genesis file tests accounts only",
			input: GenesisState{
				Accounts: []*Account{{
					Address: newTestAddressBytes(t),
					Amount:  100,
				}, {
					Address: newTestAddressBytes(t, 1),
					Amount:  100,
				}},
				Params: DefaultParams(),
			},
			expected: GenesisState{
				Accounts: []*Account{{
					Address: newTestAddressBytes(t),
					Amount:  100,
				}, {
					Address: newTestAddressBytes(t, 1),
					Amount:  100,
				}},
				OrderBooks: new(lib.OrderBooks),
				Params:     DefaultParams(),
				Supply:     &Supply{Total: 200},
				Committees: &lib.CommitteesData{List: []*lib.CommitteeData{}},
			},
		},
		{
			name:   "validators",
			detail: "the genesis file tests validators only",
			input: GenesisState{
				Validators: []*Validator{{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: 100,
					Committees:   []uint64{lib.CanopyChainId, 2},
					Output:       newTestAddressBytes(t),
				}},
				Params: DefaultParams(),
			},
			expected: GenesisState{
				Validators: []*Validator{{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: 100,
					Committees:   []uint64{lib.CanopyChainId, 2},
					Output:       newTestAddressBytes(t),
				}},
				OrderBooks: new(lib.OrderBooks),
				Params:     DefaultParams(),
				Supply: &Supply{
					Total:  100,
					Staked: 100,
					CommitteeStaked: []*Pool{{
						Id:     lib.CanopyChainId,
						Amount: 100,
					}, {
						Id:     2,
						Amount: 100,
					}},
				},
				Committees: &lib.CommitteesData{List: []*lib.CommitteeData{}},
			},
		},
		{
			name:   "pools",
			detail: "the genesis file tests pools only",
			input: GenesisState{
				Pools: []*Pool{{
					Id:     lib.CanopyChainId,
					Amount: 100,
				}},
				Params: DefaultParams(),
			},
			expected: GenesisState{
				Pools: []*Pool{{
					Id:     lib.CanopyChainId,
					Amount: 100,
				}},
				Committees: &lib.CommitteesData{List: []*lib.CommitteeData{}},
				OrderBooks: new(lib.OrderBooks),
				Params:     DefaultParams(),
				Supply: &Supply{
					Total: 100,
				},
			},
		},
		{
			name:   "order books",
			detail: "the genesis file tests order books only",
			input: GenesisState{
				OrderBooks: &lib.OrderBooks{
					OrderBooks: []*lib.OrderBook{{
						ChainId: lib.CanopyChainId,
						Orders: []*lib.SellOrder{{
							Id:                   newTestOrderId(t, 1),
							Committee:            lib.CanopyChainId,
							AmountForSale:        100,
							RequestedAmount:      100,
							SellerReceiveAddress: newTestAddressBytes(t),
							BuyerReceiveAddress:  newTestAddressBytes(t, 1),
							BuyerChainDeadline:   100, SellersSendAddress: newTestAddressBytes(t, 2),
						}, {
							Id:                 newTestOrderId(t, 2),
							Committee:          2,
							AmountForSale:      100,
							RequestedAmount:    100,
							SellersSendAddress: newTestAddressBytes(t, 2),
						}},
					}},
				},
				Params: DefaultParams(),
			},
			expected: GenesisState{
				Pools: []*Pool{{
					Id:     lib.CanopyChainId + EscrowPoolAddend,
					Amount: 200,
				}},
				Committees: &lib.CommitteesData{List: []*lib.CommitteeData{}},
				OrderBooks: &lib.OrderBooks{
					OrderBooks: []*lib.OrderBook{{
						ChainId: lib.CanopyChainId,
						Orders: []*lib.SellOrder{{
							Id:                   newTestOrderId(t, 1),
							Committee:            lib.CanopyChainId,
							AmountForSale:        100,
							RequestedAmount:      100,
							SellerReceiveAddress: newTestAddressBytes(t),
							BuyerReceiveAddress:  newTestAddressBytes(t, 1),
							BuyerChainDeadline:   100, SellersSendAddress: newTestAddressBytes(t, 2),
						}, {
							Id:                 newTestOrderId(t, 2),
							Committee:          2,
							AmountForSale:      100,
							RequestedAmount:    100,
							SellersSendAddress: newTestAddressBytes(t, 2),
						}},
					}},
				},
				Params: DefaultParams(),
				Supply: &Supply{
					Total:                  200,
					CommitteeDelegatedOnly: nil,
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			sm.height = 0
			// set the data dir path
			sm.Config.DataDirPath = dataDirPath
			// marshal genesis file to bytes
			genesisJsonBytes, err := json.MarshalIndent(&test.input, "", "  ")
			require.NoError(t, err)
			// write test genesis to file
			require.NoError(t, os.WriteFile("genesis.json", genesisJsonBytes, 0777))
			// remove the test file
			defer os.RemoveAll("genesis.json")
			// execute function call
			require.NoError(t, sm.NewFromGenesisFile())
			// validate the exported state
			validateWithExportedState(t, sm, &test.expected)
		})
	}
}

func TestReadGenesisFromFile(t *testing.T) {
	tests := []struct {
		name     string
		detail   string
		expected *GenesisState
		error    string
	}{
		{
			name:   "no genesis file",
			detail: "no genesis file was written so expect a read error",
			error:  "read genesis file failed with err",
		},
		{
			name:   "errored genesis file",
			detail: "the genesis file has an error in it (address size) which makes it invalid",
			expected: &GenesisState{
				Accounts: []*Account{
					{
						Address: nil,
						Amount:  100,
					},
				},
				Params: DefaultParams(),
			},
			error: "address size is invalid",
		},
		{
			name:   "valid genesis file",
			detail: "the genesis file is valid so will compare read (got) vs expected",
			expected: &GenesisState{
				Accounts: []*Account{
					{
						Address: newTestAddressBytes(t),
						Amount:  100,
					},
				},
				Params: DefaultParams(),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			if test.expected != nil {
				// marshal genesis file to bytes
				genesisJsonBytes, err := json.MarshalIndent(&test.expected, "", "  ")
				require.NoError(t, err)
				// write test genesis to file
				require.NoError(t, os.WriteFile("genesis.json", genesisJsonBytes, 0777))
				// remove the test file
				defer os.RemoveAll("genesis.json")
			}
			// execute the function call
			got, err := sm.ReadGenesisFromFile()
			// ensure error is expected
			require.Equal(t, test.error != "", err != nil)
			// if err isn't nil ensure that it contains the expected error
			if err != nil {
				require.ErrorContains(t, err, test.error)
				return
			}
			// compare got with expected
			require.EqualExportedValues(t, test.expected, got)
		})
	}
}

func TestNewStateFromGenesisFile(t *testing.T) {
	tests := []struct {
		name     string
		detail   string
		input    *GenesisState
		expected *GenesisState
	}{
		{
			name:   "complete",
			detail: "the complete genesis file testing",
			input: &GenesisState{
				Pools: []*Pool{{
					Id:     lib.CanopyChainId,
					Amount: 100,
				}},
				Accounts: []*Account{{
					Address: newTestAddressBytes(t),
					Amount:  100,
				}},
				Validators: []*Validator{{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: 100,
					Committees:   []uint64{lib.CanopyChainId, 2},
					Output:       newTestAddressBytes(t),
				}},
				OrderBooks: &lib.OrderBooks{
					OrderBooks: []*lib.OrderBook{{
						ChainId: lib.CanopyChainId,
						Orders: []*lib.SellOrder{{
							Id:                   newTestOrderId(t, 1),
							Committee:            lib.CanopyChainId,
							AmountForSale:        100,
							RequestedAmount:      100,
							SellerReceiveAddress: newTestAddressBytes(t),
							BuyerReceiveAddress:  newTestAddressBytes(t, 1),
							BuyerChainDeadline:   100,
							SellersSendAddress:   newTestAddressBytes(t, 2),
						}, {
							Id:                 newTestOrderId(t, 2),
							Committee:          2,
							AmountForSale:      100,
							RequestedAmount:    100,
							SellersSendAddress: newTestAddressBytes(t, 2),
						}},
					}},
				},
				Committees: &lib.CommitteesData{List: []*lib.CommitteeData{}},
				Params:     DefaultParams(),
			},
			expected: &GenesisState{
				Pools: []*Pool{{
					Id:     lib.CanopyChainId,
					Amount: 100,
				}, {
					Id:     lib.CanopyChainId + EscrowPoolAddend,
					Amount: 200,
				}},
				Accounts: []*Account{{
					Address: newTestAddressBytes(t),
					Amount:  100,
				}},
				Validators: []*Validator{{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: 100,
					Committees:   []uint64{lib.CanopyChainId, 2},
					Output:       newTestAddressBytes(t),
				}},
				OrderBooks: &lib.OrderBooks{
					OrderBooks: []*lib.OrderBook{{
						ChainId: lib.CanopyChainId,
						Orders: []*lib.SellOrder{{
							Id:                   newTestOrderId(t, 1),
							Committee:            lib.CanopyChainId,
							AmountForSale:        100,
							RequestedAmount:      100,
							SellerReceiveAddress: newTestAddressBytes(t),
							BuyerReceiveAddress:  newTestAddressBytes(t, 1),
							BuyerChainDeadline:   100, SellersSendAddress: newTestAddressBytes(t, 2),
						}, {
							Id:                 newTestOrderId(t, 2),
							Committee:          2,
							AmountForSale:      100,
							RequestedAmount:    100,
							SellersSendAddress: newTestAddressBytes(t, 2),
						}},
					}},
				},
				Committees: &lib.CommitteesData{List: []*lib.CommitteeData{}},
				Params:     DefaultParams(),
				Supply: &Supply{
					Total:  500,
					Staked: 100,
					CommitteeStaked: []*Pool{{
						Id:     lib.CanopyChainId,
						Amount: 100,
					}, {
						Id:     2,
						Amount: 100,
					}},
					CommitteeDelegatedOnly: nil,
				},
			},
		},
		{
			name:   "accounts",
			detail: "the genesis file tests accounts only",
			input: &GenesisState{
				Accounts: []*Account{{
					Address: newTestAddressBytes(t),
					Amount:  100,
				}, {
					Address: newTestAddressBytes(t, 1),
					Amount:  100,
				}},
				Params: DefaultParams(),
			},
			expected: &GenesisState{
				Accounts: []*Account{{
					Address: newTestAddressBytes(t),
					Amount:  100,
				}, {
					Address: newTestAddressBytes(t, 1),
					Amount:  100,
				}},
				OrderBooks: new(lib.OrderBooks),
				Params:     DefaultParams(),
				Committees: &lib.CommitteesData{List: []*lib.CommitteeData{}},
				Supply:     &Supply{Total: 200},
			},
		},
		{
			name:   "validators",
			detail: "the genesis file tests validators only",
			input: &GenesisState{
				Validators: []*Validator{{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: 100,
					Committees:   []uint64{lib.CanopyChainId, 2},
					Output:       newTestAddressBytes(t),
				}},
				Params: DefaultParams(),
			},
			expected: &GenesisState{
				Validators: []*Validator{{
					Address:      newTestAddressBytes(t),
					PublicKey:    newTestPublicKeyBytes(t),
					StakedAmount: 100,
					Committees:   []uint64{lib.CanopyChainId, 2},
					Output:       newTestAddressBytes(t),
				}},
				OrderBooks: new(lib.OrderBooks),
				Committees: &lib.CommitteesData{List: []*lib.CommitteeData{}},
				Params:     DefaultParams(),
				Supply: &Supply{
					Total:  100,
					Staked: 100,
					CommitteeStaked: []*Pool{{
						Id:     lib.CanopyChainId,
						Amount: 100,
					}, {
						Id:     2,
						Amount: 100,
					}},
				},
			},
		},
		{
			name:   "pools",
			detail: "the genesis file tests pools only",
			input: &GenesisState{
				Pools: []*Pool{{
					Id:     lib.CanopyChainId,
					Amount: 100,
				}},
				Params: DefaultParams(),
			},
			expected: &GenesisState{
				Pools: []*Pool{{
					Id:     lib.CanopyChainId,
					Amount: 100,
				}},
				OrderBooks: new(lib.OrderBooks),
				Committees: &lib.CommitteesData{List: []*lib.CommitteeData{}},
				Params:     DefaultParams(),
				Supply: &Supply{
					Total: 100,
				},
			},
		},
		{
			name:   "order books",
			detail: "the genesis file tests order books only",
			input: &GenesisState{
				OrderBooks: &lib.OrderBooks{
					OrderBooks: []*lib.OrderBook{{
						ChainId: lib.CanopyChainId,
						Orders: []*lib.SellOrder{{
							Id:                   newTestOrderId(t, 1),
							Committee:            lib.CanopyChainId,
							AmountForSale:        100,
							RequestedAmount:      100,
							SellerReceiveAddress: newTestAddressBytes(t),
							BuyerReceiveAddress:  newTestAddressBytes(t, 1),
							BuyerChainDeadline:   100, SellersSendAddress: newTestAddressBytes(t, 2),
						}, {
							Id:                 newTestOrderId(t, 2),
							Committee:          1,
							AmountForSale:      100,
							RequestedAmount:    100,
							SellersSendAddress: newTestAddressBytes(t, 2),
						}},
					}},
				},
				Params: DefaultParams(),
			},
			expected: &GenesisState{
				Pools: []*Pool{{
					Id:     lib.CanopyChainId + EscrowPoolAddend,
					Amount: 200,
				}},
				OrderBooks: &lib.OrderBooks{
					OrderBooks: []*lib.OrderBook{{
						ChainId: lib.CanopyChainId,
						Orders: []*lib.SellOrder{{
							Id:                   newTestOrderId(t, 1),
							Committee:            lib.CanopyChainId,
							AmountForSale:        100,
							RequestedAmount:      100,
							SellerReceiveAddress: newTestAddressBytes(t),
							BuyerReceiveAddress:  newTestAddressBytes(t, 1),
							BuyerChainDeadline:   100, SellersSendAddress: newTestAddressBytes(t, 2),
						}, {
							Id:                 newTestOrderId(t, 2),
							Committee:          1,
							AmountForSale:      100,
							RequestedAmount:    100,
							SellersSendAddress: newTestAddressBytes(t, 2),
						}},
					}},
				},
				Params:     DefaultParams(),
				Committees: &lib.CommitteesData{List: []*lib.CommitteeData{}},
				Supply: &Supply{
					Total:                  200,
					CommitteeDelegatedOnly: nil,
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			sm.height = 0
			// run the function call
			require.NoError(t, sm.NewStateFromGenesis(test.input))
			// convert written state to
			got, err := sm.ExportState()
			require.NoError(t, err)
			// sort the supply pools
			sortById := func(p []*Pool) {
				sort.Slice(p, func(i, j int) bool {
					return (p)[i].Id >= (p)[j].Id
				})
			}
			// sort the supply pools of got
			sortById(got.Supply.CommitteeStaked)
			sortById(got.Supply.CommitteeDelegatedOnly)
			// sort the supply pools of expected
			sortById(test.expected.Supply.CommitteeStaked)
			sortById(test.expected.Supply.CommitteeDelegatedOnly)
			// json for convenient compare
			gotJson, _ := json.MarshalIndent(got, "", "  ")
			expectedJson, _ := json.MarshalIndent(test.expected, "", "  ")
			// compare got vs expected
			require.EqualExportedValues(t, *test.expected, *got, fmt.Sprintf("EXPECTED:\n%s\nGOT:\n%s", expectedJson, gotJson))
		})
	}
}

func TestValidateGenesisState(t *testing.T) {
	tests := []struct {
		name   string
		detail string
		input  *GenesisState
		error  string
	}{
		{
			name:   "bad validator address",
			detail: "the validator address length is invalid",
			input: &GenesisState{
				Validators: []*Validator{
					{
						Address:      nil,
						PublicKey:    newTestPublicKeyBytes(t),
						StakedAmount: 100,
					},
				},
				Params: DefaultParams(),
			},
			error: "address size is invalid",
		},
		{
			name:   "bad validator public key",
			detail: "the validator public key length is invalid",
			input: &GenesisState{
				Validators: []*Validator{
					{
						Address:      newTestAddressBytes(t),
						PublicKey:    newTestAddressBytes(t),
						StakedAmount: 100,
					},
				},
				Params: DefaultParams(),
			},
			error: "public key size is invalid",
		},
		{
			name:   "bad validator output address",
			detail: "the validator output address length is invalid",
			input: &GenesisState{
				Validators: []*Validator{
					{
						Address:      newTestAddressBytes(t),
						PublicKey:    newTestPublicKeyBytes(t),
						StakedAmount: 100,
						Output:       newTestPublicKeyBytes(t),
					},
				},
				Params: DefaultParams(),
			},
			error: "address size is invalid",
		},
		{
			name:   "account address",
			detail: "the account address length is invalid",
			input: &GenesisState{
				Accounts: []*Account{
					{
						Address: newTestPublicKeyBytes(t),
						Amount:  100,
					},
				},
				Params: DefaultParams(),
			},
			error: "address size is invalid",
		},
		{
			name:   "duplicate committee order book",
			detail: "the order book contains a duplicate committee entry",
			input: &GenesisState{
				OrderBooks: &lib.OrderBooks{
					OrderBooks: []*lib.OrderBook{
						{
							ChainId: 0,
							Orders: []*lib.SellOrder{
								{
									Id:                 newTestOrderId(t, 1),
									Committee:          0,
									AmountForSale:      100,
									SellersSendAddress: newTestAddressBytes(t),
								},
							},
						},
						{
							ChainId: 0,
							Orders: []*lib.SellOrder{
								{
									Id:                 newTestOrderId(t, 2),
									Committee:          0,
									AmountForSale:      101,
									SellersSendAddress: newTestAddressBytes(t, 1),
								},
							},
						},
					},
				},
				Params: DefaultParams(),
			},
			error: "sell order invalid",
		},
		{
			name:   "duplicate sell order id",
			detail: "the order book contains a sell order with a duplicate id within a single committee",
			input: &GenesisState{
				OrderBooks: &lib.OrderBooks{
					OrderBooks: []*lib.OrderBook{
						{
							ChainId: 0,
							Orders: []*lib.SellOrder{
								{
									Id:                 newTestOrderId(t, 1),
									Committee:          0,
									AmountForSale:      100,
									SellersSendAddress: newTestAddressBytes(t),
								},
								{
									Id:                 newTestOrderId(t, 1),
									Committee:          0,
									AmountForSale:      101,
									SellersSendAddress: newTestAddressBytes(t, 2),
								},
							},
						},
					},
				},
				Params: DefaultParams(),
			},
			error: "sell order invalid",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create a state machine instance with default parameters
			sm := newTestStateMachine(t)
			// run function call
			err := sm.ValidateGenesisState(test.input)
			// ensure expected error
			require.Equal(t, test.error != "", err != nil)
			if err != nil {
				// ensure error contains
				require.ErrorContains(t, err, test.error)
			}
		})
	}
}

func newTestGenesisState(t *testing.T) *GenesisState {
	return &GenesisState{
		Pools: []*Pool{{
			Id:     lib.CanopyChainId,
			Amount: 100,
		}},
		Accounts: []*Account{{
			Address: newTestAddressBytes(t),
			Amount:  100,
		}},
		Validators: []*Validator{{
			Address:      newTestAddressBytes(t),
			PublicKey:    newTestPublicKeyBytes(t),
			StakedAmount: 100,
			Committees:   []uint64{lib.CanopyChainId, 2},
			Output:       newTestAddressBytes(t),
		}},
		OrderBooks: &lib.OrderBooks{
			OrderBooks: []*lib.OrderBook{{
				ChainId: lib.CanopyChainId,
				Orders: []*lib.SellOrder{{
					Id:                   newTestOrderId(t, 1),
					Committee:            lib.CanopyChainId,
					AmountForSale:        100,
					RequestedAmount:      100,
					SellerReceiveAddress: newTestAddressBytes(t),
					BuyerReceiveAddress:  newTestAddressBytes(t, 1),
					BuyerChainDeadline:   100,
					SellersSendAddress:   newTestAddressBytes(t, 2),
				}, {
					Id:                 newTestOrderId(t, 2),
					Committee:          2,
					AmountForSale:      100,
					RequestedAmount:    100,
					SellersSendAddress: newTestAddressBytes(t, 2),
				}},
			}},
		},
		Params: DefaultParams(),
	}
}

func newTestValidateGenesisState(t *testing.T) *GenesisState {
	return &GenesisState{
		Pools: []*Pool{{
			Id:     lib.CanopyChainId,
			Amount: 100,
		}, {
			Id:     lib.CanopyChainId + EscrowPoolAddend,
			Amount: 200,
		}},
		Accounts: []*Account{{
			Address: newTestAddressBytes(t),
			Amount:  100,
		}},
		Validators: []*Validator{{
			Address:      newTestAddressBytes(t),
			PublicKey:    newTestPublicKeyBytes(t),
			StakedAmount: 100,
			Committees:   []uint64{lib.CanopyChainId, 2},
			Output:       newTestAddressBytes(t),
		}},
		NonSigners:    nil,
		DoubleSigners: nil,
		OrderBooks: &lib.OrderBooks{
			OrderBooks: []*lib.OrderBook{{
				ChainId: lib.CanopyChainId,
				Orders: []*lib.SellOrder{{
					Id:                   newTestOrderId(t, 1),
					Committee:            lib.CanopyChainId,
					AmountForSale:        100,
					RequestedAmount:      100,
					SellerReceiveAddress: newTestAddressBytes(t),
					BuyerReceiveAddress:  newTestAddressBytes(t, 1),
					BuyerChainDeadline:   100,
					SellersSendAddress:   newTestAddressBytes(t, 2),
				}, {
					Id:                 newTestOrderId(t, 2),
					Committee:          2,
					AmountForSale:      100,
					RequestedAmount:    100,
					SellersSendAddress: newTestAddressBytes(t, 2),
				}},
			}},
		},
		Params: DefaultParams(),
		Supply: &Supply{
			Total:  500,
			Staked: 100,
			CommitteeStaked: []*Pool{{
				Id:     lib.CanopyChainId,
				Amount: 100,
			}, {
				Id:     2,
				Amount: 100,
			}},
			CommitteeDelegatedOnly: nil,
		},
		Committees:        &lib.CommitteesData{List: []*lib.CommitteeData{}},
		RetiredCommittees: nil,
	}
}

func validateWithExportedState(t *testing.T, sm StateMachine, expected *GenesisState) {
	// convert written state to
	got, err := sm.ExportState()
	require.NoError(t, err)
	// sort the supply pools
	sortById := func(p []*Pool) {
		sort.Slice(p, func(i, j int) bool {
			return (p)[i].Id >= (p)[j].Id
		})
	}
	// sort the supply pools of got
	sortById(got.Supply.CommitteeStaked)
	sortById(got.Supply.CommitteeDelegatedOnly)
	// sort the supply pools of expected
	sortById(expected.Supply.CommitteeStaked)
	sortById(expected.Supply.CommitteeDelegatedOnly)
	// json for convenient compare
	gotJson, _ := json.MarshalIndent(got, "", "  ")
	expectedJson, _ := json.MarshalIndent(expected, "", "  ")
	// compare got vs expected
	require.EqualExportedValues(t, expected, got, fmt.Sprintf("EXPECTED:\n%s\nGOT:\n%s", expectedJson, gotJson))
}

// TestGenesisJSONRoundTripPreservesRetiredCommittees ensures that retired committees survive a
// GenesisState -> JSON -> GenesisState round trip. Retired committees are non-subsidized "for
// eternity"; if the JSON codec drops them, an export/import upgrade silently makes them live
// again and they resume receiving minted rewards.
func TestGenesisJSONRoundTripPreservesRetiredCommittees(t *testing.T) {
	// a genesis state carrying two retired committees
	expected := &GenesisState{
		Params:            DefaultParams(),
		RetiredCommittees: []uint64{7, 21},
	}
	// marshal to JSON via the custom GenesisState codec
	bz, err := json.Marshal(expected)
	require.NoError(t, err)
	// unmarshal back into a fresh GenesisState
	got := new(GenesisState)
	require.NoError(t, json.Unmarshal(bz, got))
	// the retired committees must survive the round trip
	require.Equal(t, expected.RetiredCommittees, got.RetiredCommittees,
		"retired committees must survive a genesis export/import round trip; json=%s", string(bz))
}
