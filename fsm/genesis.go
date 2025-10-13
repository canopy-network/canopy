package fsm

import (
	"encoding/json"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"os"
	"path/filepath"
	"time"
)

/* GENESIS LOGIC: implements logic to import a json file to create the state at height 0 and export the state at any height */

// NewFromGenesisFile() creates a new beginning state from a file
func (s *StateMachine) NewFromGenesisFile() (err lib.ErrorI) {
	// get the genesis object from a file
	genesis, err := s.ReadGenesisFromFile()
	if err != nil {
		return
	}
	// set the state using the genesis object
	if err = s.NewStateFromGenesis(genesis); err != nil {
		return
	}
	// commit the genesis state to persistence (database)
	if _, err = s.store.(lib.StoreI).Commit(); err != nil {
		return
	}
	// update the height from 0 to 1
	s.height += 1
	// exit
	return
}

// ReadGenesisFromFile() reads a GenesisState object from a file
func (s *StateMachine) ReadGenesisFromFile() (genesis *GenesisState, e lib.ErrorI) {
	// create a new genesis object to ensure no nil result
	genesis = new(GenesisState)
	// read the genesis file from the data directory + `genesis.json` path
	bz, err := os.ReadFile(filepath.Join(s.Config.DataDirPath, lib.GenesisFilePath))
	if err != nil {
		return nil, ErrReadGenesisFile(err)
	}
	// populate the genesis object using the file bytes
	if err = json.Unmarshal(bz, genesis); err != nil {
		return nil, ErrUnmarshalGenesis(err)
	}
	// ensure the genesis object is valid
	e = s.ValidateGenesisState(genesis)
	// exit
	return
}

// NewStateFromGenesis() creates a new beginning state using a GenesisState object
// There are purposefully non-included fields that are 'export only'
func (s *StateMachine) NewStateFromGenesis(genesis *GenesisState) (err lib.ErrorI) {
	// create a new supply tracker object reference
	supply := new(Supply)
	// set the accounts from the genesis object in state
	if err = s.SetAccounts(genesis.Accounts, supply); err != nil {
		return
	}
	// set the pools from the genesis object in state
	if err = s.SetPools(genesis.Pools, supply); err != nil {
		return
	}
	// set the validators from the genesis object in state
	if err = s.SetValidators(genesis.Validators, supply); err != nil {
		return
	}
	// set the order books from the genesis object in state
	if err = s.SetOrderBooks(genesis.OrderBooks, supply); err != nil {
		return
	}
	// set the calculated supply from the genesis object in state
	if err = s.SetSupply(supply); err != nil {
		return
	}
	// set the retired committees from the genesis object in state
	if err = s.SetRetiredCommittees(genesis.RetiredCommittees); err != nil {
		return
	}
	// set the governance params from the genesis object in state
	return s.SetParams(genesis.Params)
}

// ValidateGenesisState() validates a GenesisState object
func (s *StateMachine) ValidateGenesisState(genesis *GenesisState) (err lib.ErrorI) {
	// ensure the governance params from the genesis object are valid
	if err = genesis.Params.Check(); err != nil {
		return
	}
	// for each validator, apply basic validations on the required fields
	for _, val := range genesis.Validators {
		// ensure the validator address is the proper length
		if len(val.Address) != crypto.AddressSize {
			return ErrAddressSize()
		}
		// ensure the validator public key is the proper length
		if !val.Delegate && len(val.PublicKey) != crypto.BLS12381PubKeySize {
			return ErrPublicKeySize()
		}
		// ensure the validator output address is the proper length
		if len(val.Output) != crypto.AddressSize {
			return ErrAddressSize()
		}
	}
	// for each account, apply basic validations on the required fields
	for _, account := range genesis.Accounts {
		// ensure the account address has the proper size
		if len(account.Address) != crypto.AddressSize {
			return ErrAddressSize()
		}
	}
	// if the order books aren't nil
	if genesis.OrderBooks != nil {
		// de-duplicate the committee order books
		deDuplicateCommittees := lib.NewDeDuplicator[uint64]()
		// for each order book in the list
		for _, orderBook := range genesis.OrderBooks.OrderBooks {
			// if already found
			if found := deDuplicateCommittees.Found(orderBook.ChainId); found {
				return InvalidSellOrder()
			}
			// if the book exists but there's no sell orders
			if len(orderBook.Orders) == 0 {
				return InvalidSellOrder()
			}
			// ensure there's no duplicate order-ids within the book
			deDuplicateIds := lib.NewDeDuplicator[string]()
			// for each order in the book
			for _, order := range orderBook.Orders {
				// check if order already found
				if found := deDuplicateIds.Found(lib.BytesToString(order.Id)); found {
					return InvalidSellOrder()
				}
			}
		}
	}
	return
}

// ExportState() creates a GenesisState object from the current state
func (s *StateMachine) ExportState() (genesis *GenesisState, err lib.ErrorI) {
	// create a new genesis state object
	genesis = new(GenesisState)
	// populate the accounts from the state
	genesis.Accounts, err = s.GetAccounts()
	if err != nil {
		return nil, err
	}
	// populate the pools from the state
	genesis.Pools, err = s.GetPools()
	if err != nil {
		return nil, err
	}
	// populate the validators from the state
	genesis.Validators, err = s.GetValidators()
	if err != nil {
		return nil, err
	}
	// populate the governance params from the state
	genesis.Params, err = s.GetParams()
	if err != nil {
		return nil, err
	}
	// populate the non-signers from the state
	genesis.NonSigners, err = s.GetNonSigners()
	if err != nil {
		return nil, err
	}
	// populate the double-signers from the state
	genesis.DoubleSigners, err = s.GetDoubleSigners()
	if err != nil {
		return nil, err
	}
	// populate the order books from the state
	genesis.OrderBooks, err = s.GetOrderBooks()
	if err != nil {
		return nil, err
	}
	// populate the supply from the state
	genesis.Supply, err = s.GetSupply()
	if err != nil {
		return nil, err
	}
	// populate the retired committees from the state
	genesis.RetiredCommittees, err = s.GetRetiredCommittees()
	if err != nil {
		return nil, err
	}
	// populate the list of committee data from the state
	genesis.Committees, err = s.GetCommitteesData()
	if err != nil {
		return nil, err
	}
	// return the genesis file
	return genesis, nil
}

// GENESIS HELPERS BELOW

// genesisState is the json.Marshaller and json.Unmarshaler implementation for the GenesisState object
type genesisState struct {
	Time          string              `json:"time,omitempty"`
	Pools         []*Pool             `json:"pools,omitempty"`
	Accounts      []*Account          `protobuf:"bytes,3,rep,name=accounts,proto3" json:"accounts,omitempty"`
	NonSigners    NonSigners          `json:"nonSigners"`
	Validators    []*Validator        `protobuf:"bytes,4,rep,name=validators,proto3" json:"validators,omitempty"`
	Params        *Params             `protobuf:"bytes,5,opt,name=params,proto3" json:"params,omitempty"`
	Supply        *Supply             `json:"supply"`
	OrderBooks    *lib.OrderBooks     `protobuf:"bytes,7,opt,name=order_books,json=orderBooks,proto3" json:"orderBooks,omitempty"`
	DoubleSigners []*lib.DoubleSigner `protobuf:"bytes,6,rep,name=double_signers,json=doubleSigners,proto3" json:"doubleSigners,omitempty"` // only used for export
}

// MarshalJSON() is the json.Marshaller implementation for the GenesisState object
func (x *GenesisState) MarshalJSON() ([]byte, error) {
	var t string
	if x.Time != 0 {
		t = time.UnixMicro(int64(x.Time)).Format(time.DateTime)
	}
	return json.Marshal(genesisState{
		Time:          t,
		Pools:         x.Pools,
		Accounts:      x.Accounts,
		NonSigners:    x.NonSigners,
		Validators:    x.Validators,
		Params:        x.Params,
		Supply:        x.Supply,
		OrderBooks:    x.OrderBooks,
		DoubleSigners: x.DoubleSigners,
	})
}

// UnmarshalJSON() is the json.Unmarshaler implementation for the GenesisState object
func (x *GenesisState) UnmarshalJSON(bz []byte) (err error) {
	ptr := new(genesisState)
	if err = json.Unmarshal(bz, ptr); err != nil {
		return
	}
	if ptr.Time != "" {
		t, e := time.Parse(time.DateTime, ptr.Time)
		if e != nil {
			return e
		}
		x.Time = uint64(t.UnixMicro())
	}
	x.Params, x.Pools, x.Supply = ptr.Params, ptr.Pools, ptr.Supply
	x.Accounts, x.Validators, x.NonSigners = ptr.Accounts, ptr.Validators, ptr.NonSigners
	x.OrderBooks, x.DoubleSigners = ptr.OrderBooks, ptr.DoubleSigners
	return
}
