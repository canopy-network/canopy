package fsm

import (
	"bytes"
	"encoding/json"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"sort"
)

/* This file defines the account, pool, and supply tracker state interactions */

// GetAccount() returns an Account structure for a specific address
func (s *StateMachine) GetAccount(address crypto.AddressI) (*Account, lib.ErrorI) {
	// check cache
	if acc, found := s.cache.accounts[lib.MemHash(address.Bytes())]; found {
		return acc, nil
	}
	// retrieve the account from the state store
	bz, err := s.Get(KeyForAccount(address))
	if err != nil {
		return nil, err
	}
	// convert the account bytes into a structure
	acc, err := s.unmarshalAccount(bz)
	if err != nil {
		return nil, err
	}
	// convert the address into bytes and set it
	acc.Address = address.Bytes()
	return acc, nil
}

// GetAccounts() returns all Account structures in the state
func (s *StateMachine) GetAccounts() (result []*Account, err lib.ErrorI) {
	// iterate through the account prefix
	it, err := s.Iterator(AccountPrefix())
	if err != nil {
		return nil, err
	}
	defer it.Close()
	// for each item of the iterator
	for ; it.Valid(); it.Next() {
		var acc *Account
		acc, err = s.unmarshalAccount(it.Value())
		if err != nil {
			return nil, err
		}
		result = append(result, acc)
	}
	// return the result
	return result, nil
}

// GetAccountsPaginated() returns a page of Account structures in the state
func (s *StateMachine) GetAccountsPaginated(p lib.PageParams) (page *lib.Page, err lib.ErrorI) {
	// create a new 'accounts' page
	page, res := lib.NewPage(p, AccountsPageName), make(AccountPage, 0)
	// load the page using the account prefix iterator
	err = page.Load(AccountPrefix(), false, &res, s.store, func(_, b []byte) (err lib.ErrorI) {
		acc, err := s.unmarshalAccount(b)
		if err == nil {
			res = append(res, acc)
		}
		return
	})
	return
}

// GetAccountBalance() returns the balance of an Account at a specific address
func (s *StateMachine) GetAccountBalance(address crypto.AddressI) (uint64, lib.ErrorI) {
	// get the account from the state
	account, err := s.GetAccount(address)
	if err != nil {
		return 0, err
	}
	// return the amount linked to the account
	return account.Amount, nil
}

// SetAccount() upserts an account into the state
func (s *StateMachine) SetAccount(account *Account) lib.ErrorI {
	// add to cache
	s.cache.accounts[lib.MemHash(account.Address)] = account
	// convert bytes to the address object
	address := crypto.NewAddressFromBytes(account.Address)
	// if the amount is 0, delete the account from state to prevent unnecessary bloat
	if account.Amount == 0 {
		return s.Delete(KeyForAccount(address))
	}
	// convert the account into bytes
	bz, err := s.marshalAccount(account)
	if err != nil {
		return err
	}
	// set the account into state using the 'prefixed' key for the account
	if err = s.Set(KeyForAccount(address), bz); err != nil {
		return err
	}
	return nil
}

// SetAccount() upserts multiple accounts into the state
func (s *StateMachine) SetAccounts(accounts []*Account, supply *Supply) (err lib.ErrorI) {
	// for each account
	for _, acc := range accounts {
		// add the account amount to the supply object
		supply.Total += acc.Amount
		// set the account in state
		if err = s.SetAccount(acc); err != nil {
			return
		}
	}
	return
}

// AccountDeductFees() removes fees from a specific address and adds them to the Canopy reward pool
func (s *StateMachine) AccountDeductFees(address crypto.AddressI, fee uint64) lib.ErrorI {
	// deduct the fee from the account
	if err := s.AccountSub(address, fee); err != nil {
		return err
	}
	// add the fee to the reward pool for the 'self' chain id
	return s.PoolAdd(s.Config.ChainId, fee)
}

// AccountAdd() adds tokens to an Account
func (s *StateMachine) AccountAdd(address crypto.AddressI, amountToAdd uint64) lib.ErrorI {
	// ensure no unnecessary database updates
	if amountToAdd == 0 {
		return nil
	}
	// get the account from state
	account, err := s.GetAccount(address)
	if err != nil {
		return err
	}
	// add the tokens to the account structure
	account.Amount += amountToAdd
	// set the account back in state
	return s.SetAccount(account)
}

// AccountSub() removes tokens from an Account
func (s *StateMachine) AccountSub(address crypto.AddressI, amountToSub uint64) lib.ErrorI {
	// ensure no unnecessary database updates
	if amountToSub == 0 {
		return nil
	}
	// get the account from the state
	account, err := s.GetAccount(address)
	if err != nil {
		return err
	}
	// if the account amount is less than the amount to subtract; return insufficient funds
	if account.Amount < amountToSub {
		return ErrInsufficientFunds()
	}
	// subtract from the account amount
	account.Amount -= amountToSub
	// set the account in state
	return s.SetAccount(account)
}

// unmarshalAccount() converts bytes into an Account structure
func (s *StateMachine) unmarshalAccount(bz []byte) (*Account, lib.ErrorI) {
	// create a new account structure to ensure we never have 'nil' accounts
	acc := new(Account)
	// unmarshal the bytes into the account structure
	if err := lib.Unmarshal(bz, acc); err != nil {
		return nil, err
	}
	// return the account
	return acc, nil
}

// marshalAccount() converts an Account structure into bytes
func (s *StateMachine) marshalAccount(account *Account) ([]byte, lib.ErrorI) {
	return lib.Marshal(account)
}

// POOL CODE BELOW

/*
	Pools are owner-less designation funds that are 'earmarked' for a purpose
	NOTE: A distinct structure for pools are used instead of a 'hard-coded account address'
	to simply prove that no-one owns the private key for that account
*/

// GetPool() returns a Pool structure for a specific ID
func (s *StateMachine) GetPool(id uint64) (*Pool, lib.ErrorI) {
	// get the pool bytes from the state using the Key a specific id
	bz, err := s.Get(KeyForPool(id))
	if err != nil {
		return nil, err
	}
	// convert the bytes into a pool structure
	pool, err := s.unmarshalPool(bz)
	if err != nil {
		return nil, err
	}
	// set the pool id from the key
	pool.Id = id
	// return the pool
	return pool, nil
}

// GetPools() returns all Pool structures in the state
func (s *StateMachine) GetPools() (result []*Pool, err lib.ErrorI) {
	// get an iterator for the pool group
	it, err := s.Iterator(PoolPrefix())
	if err != nil {
		return
	}
	defer it.Close()
	// for each item of the iterator
	for ; it.Valid(); it.Next() {
		var p *Pool
		p, err = s.unmarshalPool(it.Value())
		if err != nil {
			return
		}
		// append the pool to the result slice
		result = append(result, p)
	}
	return
}

// GetPoolsPaginated() returns a particular page of Pool structures in the state
func (s *StateMachine) GetPoolsPaginated(p lib.PageParams) (page *lib.Page, err lib.ErrorI) {
	// create a new pool page
	res, page := make(PoolPage, 0), lib.NewPage(p, PoolPageName)
	// populate the pool page using the pool prefix
	err = page.Load(PoolPrefix(), false, &res, s.store, func(_, b []byte) (err lib.ErrorI) {
		acc, err := s.unmarshalPool(b)
		if err == nil {
			res = append(res, acc)
		}
		return
	})
	return
}

// GetPoolBalance() returns the balance of a Pool at an ID
func (s *StateMachine) GetPoolBalance(id uint64) (uint64, lib.ErrorI) {
	// get the pool from state
	pool, err := s.GetPool(id)
	if err != nil {
		return 0, err
	}
	// return the pool amount
	return pool.Amount, nil
}

// SetPool() upserts a Pool structure into the state
func (s *StateMachine) SetPool(pool *Pool) (err lib.ErrorI) {
	// if the pool has a 0 balance
	if pool.Amount == 0 {
		return s.Delete(KeyForPool(pool.Id))
	}
	// convert the pool to bytes
	bz, err := s.marshalPool(pool)
	if err != nil {
		return
	}
	// set the pool bytes in state using the pool id
	if err = s.Set(KeyForPool(pool.Id), bz); err != nil {
		return
	}
	return
}

// SetPools() upserts multiple Pool structures into the state
func (s *StateMachine) SetPools(pools []*Pool, supply *Supply) (err lib.ErrorI) {
	// for each pool
	for _, pool := range pools {
		// add the pool amount to the total supply
		supply.Total += pool.Amount
		// set the pool in state
		if err = s.SetPool(pool); err != nil {
			return
		}
	}
	return
}

// MintToPool() adds newly created tokens to the Pool structure
func (s *StateMachine) MintToPool(id uint64, amount uint64) lib.ErrorI {
	// track the newly created inflation with the supply structure
	if err := s.AddToTotalSupply(amount); err != nil {
		return err
	}
	// update the pools balance with the new inflation
	return s.PoolAdd(id, amount)
}

// PoolAdd() adds tokens to the Pool structure
func (s *StateMachine) PoolAdd(id uint64, amountToAdd uint64) lib.ErrorI {
	// get the pool from the
	pool, err := s.GetPool(id)
	if err != nil {
		return err
	}
	pool.Amount += amountToAdd
	return s.SetPool(pool)
}

// PoolSub() removes tokens from the Pool structure
func (s *StateMachine) PoolSub(id uint64, amountToSub uint64) lib.ErrorI {
	// get the pool from the state using the 'id'
	pool, err := s.GetPool(id)
	if err != nil {
		return err
	}
	// if the pool amount is less than the subtracted amount; return insufficient funds
	if pool.Amount < amountToSub {
		return ErrInsufficientFunds()
	}
	// subtract from the pool balance
	pool.Amount -= amountToSub
	// update the pool in state
	return s.SetPool(pool)
}

// unmarshalPool() coverts bytes into a Pool structure
func (s *StateMachine) unmarshalPool(bz []byte) (*Pool, lib.ErrorI) {
	// create a new pool object reference to ensure no 'nil' pools are used
	pool := new(Pool)
	// populate the pool object with the bytes
	if err := lib.Unmarshal(bz, pool); err != nil {
		return nil, err
	}
	// return the pool
	return pool, nil
}

// marshalPool() coverts a Pool structure into bytes
func (s *StateMachine) marshalPool(pool *Pool) ([]byte, lib.ErrorI) {
	return lib.Marshal(pool)
}

// SUPPLY CODE BELOW

/*
	Supply structure provides an organized view of the overall financial status,
    showing both the total amount available and how it's distributed among various pools and purposes.
*/

// AddToTotalSupply() adds to the total supply count
func (s *StateMachine) AddToTotalSupply(amount uint64) lib.ErrorI {
	// get the supply tracker
	supply, err := s.GetSupply()
	if err != nil {
		return err
	}
	// add to the total supply
	supply.Total += amount
	// set the supply back in state
	return s.SetSupply(supply)
}

// AddToStakedSupply() adds to the staked supply count (staked + delegated)
func (s *StateMachine) AddToStakedSupply(amount uint64) lib.ErrorI {
	// get the supply tracker from the state
	supply, err := s.GetSupply()
	if err != nil {
		return err
	}
	// add to the staked amount in the supply tracker
	supply.Staked += amount
	// set the supply tracker back in state
	return s.SetSupply(supply)
}

// AddToStakedSupply() adds to the staked supply count
func (s *StateMachine) AddToDelegateSupply(amount uint64) lib.ErrorI {
	// get the supply from the state
	supply, err := s.GetSupply()
	if err != nil {
		return err
	}
	// add to the delegation only amount in the supply tracker
	supply.DelegatedOnly += amount
	// set the supply structure back in state
	return s.SetSupply(supply)
}

// AddToCommitteeSupplyForChain() adds to the committee staked supply count
func (s *StateMachine) AddToCommitteeSupplyForChain(chainId uint64, amount uint64) lib.ErrorI {
	return s.addToSupplyPool(chainId, amount, CommitteesWithDelegations)
}

// AddToDelegateSupplyForChain() adds to the delegate staked supply count
func (s *StateMachine) AddToDelegateSupplyForChain(chainId uint64, amount uint64) lib.ErrorI {
	return s.addToSupplyPool(chainId, amount, DelegationsOnly)
}

// SubFromTotalSupply() removes from the total supply count
func (s *StateMachine) SubFromTotalSupply(amount uint64) lib.ErrorI {
	// get the supply tracker
	supply, err := s.GetSupply()
	if err != nil {
		return err
	}
	// ensure there's enough supply to subtract
	if supply.Total < amount {
		return ErrInsufficientSupply()
	}
	// reduce the total supply
	supply.Total -= amount
	// set the supply tracker in the state
	return s.SetSupply(supply)
}

// SubFromStakedSupply() removes from the staked supply count (staked + delegated)
func (s *StateMachine) SubFromStakedSupply(amount uint64) lib.ErrorI {
	// get the supply tracker
	supply, err := s.GetSupply()
	if err != nil {
		return err
	}
	// ensure there's enough staked supply to subtract
	if supply.Staked < amount {
		return ErrInsufficientSupply()
	}
	// subtract the amount from the staked supply
	supply.Staked -= amount
	// set the supply in state
	return s.SetSupply(supply)
}

// SubFromDelegateSupply() removes from the delegated supply count
func (s *StateMachine) SubFromDelegateSupply(amount uint64) lib.ErrorI {
	// get the supply tracker
	supply, err := s.GetSupply()
	if err != nil {
		return err
	}
	// ensure there's enough delegation only supply
	if supply.DelegatedOnly < amount {
		return ErrInsufficientSupply()
	}
	// subtract the delegation only amount
	supply.DelegatedOnly -= amount
	// set the supply in state
	return s.SetSupply(supply)
}

// SubFromCommitteeStakedSupplyForChain() removes from the committee staked supply count
func (s *StateMachine) SubFromCommitteeStakedSupplyForChain(chainId uint64, amount uint64) lib.ErrorI {
	return s.subFromSupplyPool(chainId, amount, CommitteesWithDelegations)
}

// SubFromDelegateStakedSupplyForChain() removes from the delegate committee staked supply count
func (s *StateMachine) SubFromDelegateStakedSupplyForChain(chainId uint64, amount uint64) lib.ErrorI {
	return s.subFromSupplyPool(chainId, amount, DelegationsOnly)
}

// GetCommitteeStakedSupplyForChain() retrieves the committee staked supply count
func (s *StateMachine) GetCommitteeStakedSupplyForChain(chainId uint64) (p *Pool, err lib.ErrorI) {
	return s.getSupplyPool(chainId, CommitteesWithDelegations)
}

// GetFromDelegateStakedSupply() retrieves the delegate committee staked supply count
func (s *StateMachine) GetDelegateStakedSupplyForChain(chainId uint64) (p *Pool, err lib.ErrorI) {
	return s.getSupplyPool(chainId, DelegationsOnly)
}

// GetSupply() returns the Supply structure held in the state
func (s *StateMachine) GetSupply() (*Supply, lib.ErrorI) {
	// get the supply tracker bytes from the state
	bz, err := s.Get(SupplyPrefix())
	if err != nil {
		return nil, err
	}
	// convert the supply tracker bytes into an object
	return s.unmarshalSupply(bz)
}

// SetSupply() upserts the Supply structure into the state
func (s *StateMachine) SetSupply(supply *Supply) lib.ErrorI {
	// convert the supply tracker object to bytes
	bz, err := s.marshalSupply(supply)
	if err != nil {
		return err
	}
	// set the bytes in state under the 'supply prefix'
	if err = s.Set(SupplyPrefix(), bz); err != nil {
		return err
	}
	return nil
}

// unmarshalSupply() converts bytes into the supply
func (s *StateMachine) unmarshalSupply(bz []byte) (*Supply, lib.ErrorI) {
	supply := new(Supply)
	// convert the supply bytes into a supply object
	if err := lib.Unmarshal(bz, supply); err != nil {
		return nil, err
	}
	// return the object
	return supply, nil
}

// marshalSupply() converts the Supply into bytes
func (s *StateMachine) marshalSupply(supply *Supply) ([]byte, lib.ErrorI) {
	return lib.Marshal(supply)
}

// addToSupplyPool() adds to a supply pool using an addition callback with 'executeOnSupplyPool'
func (s *StateMachine) addToSupplyPool(chainId, amount uint64, targetType SupplyPoolType) lib.ErrorI {
	// execute the callback on the supply pool that has a certain chainID and type
	return s.executeOnSupplyPool(chainId, targetType, func(s *Supply, p *Pool) (err lib.ErrorI) {
		// add to the supply pool amount
		p.Amount += amount
		return
	})
}

// subFromSupplyPool() subtracts from a supply pool using a subtraction callback with 'executeOnSupplyPool'
func (s *StateMachine) subFromSupplyPool(chainId, amount uint64, targetType SupplyPoolType) lib.ErrorI {
	// execute the callback on the supply pool that has a certain chainID and type
	return s.executeOnSupplyPool(chainId, targetType, func(s *Supply, p *Pool) (err lib.ErrorI) {
		// ensure no nil or insufficient supply
		if p == nil || p.Amount < amount {
			return ErrInsufficientSupply()
		}
		// subtract from the supply pool
		p.Amount -= amount
		return
	})
}

// getSupplyPool() returns the supply pool based on the target type
func (s *StateMachine) getSupplyPool(chainId uint64, targetType SupplyPoolType) (p *Pool, err lib.ErrorI) {
	// get the supply pools for the given type
	poolList, _, err := s.getSupplyPools(targetType)
	if err != nil {
		return
	}
	// find or insert the pool for the chainId
	p = s.findOrCreateSupplyPool(poolList, chainId)
	return
}

// getSupplyPools retrieves a particular pool based on the target type
func (s *StateMachine) getSupplyPools(targetType SupplyPoolType) (poolList *[]*Pool, supply *Supply, err lib.ErrorI) {
	// get the supply object from state
	supply, err = s.GetSupply()
	if err != nil {
		return
	}
	// determine the type of the target
	switch targetType {
	case CommitteesWithDelegations:
		poolList = &supply.CommitteeStaked
	case DelegationsOnly:
		poolList = &supply.CommitteeDelegatedOnly
	}
	return
}

// executeOnSupplyPool() finds a target pool using the target type and chainId and executes a callback on it
func (s *StateMachine) executeOnSupplyPool(chainId uint64, targetType SupplyPoolType, callback func(s *Supply, p *Pool) lib.ErrorI) lib.ErrorI {
	arr, supply, err := s.getSupplyPools(targetType)
	if err != nil {
		return err
	}
	// locate the target pool
	targetPool := s.findOrCreateSupplyPool(arr, chainId)
	// execute the business logic callback
	if err = callback(supply, targetPool); err != nil {
		return err
	}
	// filter zeroes and sort the pool
	// this prevents dead committees from bloating the supply structure
	FilterAndSortPool(arr)
	// finally set the supply
	return s.SetSupply(supply)
}

// findOrCreateSupplyPool() searches for a pool by chainId or creates a new one if not found
func (s *StateMachine) findOrCreateSupplyPool(poolArr *[]*Pool, chainId uint64) (pool *Pool) {
	// iterate through the list looking for the supply pool
	for _, pool = range *poolArr {
		if pool.Id == chainId {
			return
		}
	}
	// if pool not found
	// 1. set pool return variable
	pool = &Pool{Id: chainId}
	// 2. add it to the list
	*poolArr = append(*poolArr, pool)
	return
}

// This file extends the structures generated by protobuf found in `account.pb.go`
// Here you'll find additional methods that assist the business logic of the objects

// Sort() sorts the list of Pools for committees and delegations by amount (stake) high to low
func (x *Supply) Sort() {
	x.SortCommittees()
	x.SortDelegations()
}

// SortCommittees() sorts the committees list by amount (stake) high to low
func (x *Supply) SortCommittees() { FilterAndSortPool(&x.CommitteeStaked) }

// SortCommittees() sorts the delegations list by amount (stake) high to low
func (x *Supply) SortDelegations() { FilterAndSortPool(&x.CommitteeDelegatedOnly) }

// filterAndSort() removes zero and nil elements from the pool slice and then sorts the slice by amount
// finally setting the result to the pointer from the parameter
func FilterAndSortPool(x *[]*Pool) {
	// check if the pool is nil
	if x == nil {
		return
	}
	// filter zero elements
	result := make([]*Pool, 0, len(*x))
	for _, v := range *x {
		if v != nil && v.Amount != 0 {
			result = append(result, v)
		}
	}
	// sort the slice by amount
	sort.Slice(result, func(i, j int) bool {
		// second order sort is ascending by Ids
		if result[i].Amount == result[j].Amount {
			return result[i].Id < result[j].Id
		}
		// descending by amount
		return result[i].Amount > result[j].Amount
	})
	// set to the pointer
	*x = result
}

// ACCOUNT AND POOL HELPERS BELOW
func init() {
	// Register the pages for converting bytes of Page into the correct Page object
	lib.RegisteredPageables[PoolPageName] = new(PoolPage)
	lib.RegisteredPageables[AccountsPageName] = new(AccountPage)
}

const (
	PoolPageName     = "pools"    // name for page of 'Pools'
	AccountsPageName = "accounts" // name for page of 'Accounts'
)

type PoolPage []*Pool

// PoolPage satisfies the Page interface
func (p *PoolPage) New() lib.Pageable { return &PoolPage{} }

type AccountPage []*Account

// AccountPage satisfies the Page interface
func (p *AccountPage) New() lib.Pageable { return &AccountPage{} }

type SupplyPoolType int

const (
	CommitteesWithDelegations SupplyPoolType = 0
	DelegationsOnly           SupplyPoolType = 1
)

// account is the json.Marshaller and json.Unmarshaler implementation for the Account object
type account struct {
	Address lib.HexBytes `json:"address,omitempty"`
	Amount  uint64       `json:"amount,omitempty"`
}

// MarshalJSON() is the json.Marshaller implementation for the Account object
func (x *Account) MarshalJSON() ([]byte, error) {
	return json.Marshal(account{x.Address, x.Amount})
}

// UnmarshalJSON() is the json.Unmarshaler implementation for the Account object
func (x *Account) UnmarshalJSON(bz []byte) (err error) {
	a := new(account)
	if err = json.Unmarshal(bz, a); err != nil {
		return err
	}
	x.Address, x.Amount = a.Address, a.Amount
	return
}

// pool is the json.Marshaller and json.Unmarshaler implementation for the Pool object
type pool struct {
	ID     uint64 `json:"id"`
	Amount uint64 `json:"amount"`
}

// MarshalJSON() is the json.Marshaller implementation for the Pool object
func (x *Pool) MarshalJSON() ([]byte, error) {
	return json.Marshal(pool{x.Id, x.Amount})
}

// UnmarshalJSON() is the json.Unmarshaler implementation for the Pool object
func (x *Pool) UnmarshalJSON(bz []byte) (err error) {
	a := new(pool)
	if err = json.Unmarshal(bz, a); err != nil {
		return err
	}
	x.Id, x.Amount = a.ID, a.Amount
	return
}

// GetPointsFor() returns the amount of points an address has
func (x *Pool) GetPointsFor(address []byte) (points uint64, err lib.ErrorI) {
	// add to existing if found
	for _, lp := range x.Points {
		// if the address is found
		if bytes.Equal(lp.Address, address) {
			// exit
			return lp.Points, nil
		}
	}
	// exit
	return 0, lib.ErrPointHolderNotFound()
}

// AddPoints() converts a 'percent control' to points using N = (t × P) / (1 - t)
// Where N is new_points, t = the desired ownership fraction, and P is the initial pool size
func (x *Pool) AddPoints(address []byte, points uint64) lib.ErrorI {
	// add to total points
	x.TotalPoolPoints += points
	// add to existing if found
	for _, lp := range x.Points {
		// if the address is found
		if bytes.Equal(lp.Address, address) {
			// add points
			lp.Points += points
			// exit
			return nil
		}
	}
	// add to the points
	x.Points = append(x.Points, &PoolPoints{Address: address, Points: points})
	// exit
	return nil
}

// RemovePoints() removes liquidity points for a certain provider in the pool
func (x *Pool) RemovePoints(address []byte, points uint64) (err lib.ErrorI) {
	// add to existing if found
	for i, lp := range x.Points {
		// if the address is found
		if bytes.Equal(lp.Address, address) {
			// update total points
			x.TotalPoolPoints -= points
			// calculate the remaining lp points
			x.Points[i].Points -= points
			// check if should evict from slice
			if x.Points[i].Points == 0 {
				// remove from the slice
				x.Points = append(x.Points[:i], x.Points[i+1:]...)
			}
			// check for zero pool
			if x.TotalPoolPoints == 0 {
				return lib.ErrZeroLiquidityPool()
			}
			// exit
			return
		}
	}
	// exit
	return lib.ErrPointHolderNotFound()
}
