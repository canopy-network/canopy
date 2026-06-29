package canoliq

import (
	"github.com/canopy-network/go-plugin/contract"
)

// LoadParams reads the canoLiq parameters from state, falling back to
// DefaultParams() if the key is unset. Genesis is responsible for writing
// the initial set so steady-state reads always observe the persisted value.
func (c *Canoliq) LoadParams() (*contract.CanoliqParams, *contract.PluginError) {
	q := qid()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{{QueryId: q, Key: KeyForParams()}},
	})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	// FSM StateRead returns Entry{Value: nil} for missing keys (not zero
	// entries), so we must check the value length too. Unmarshaling an empty
	// byte slice yields a zero-valued CanoliqParams that ValidateParams would
	// (correctly) reject with "split sum 0 != 10000".
	if len(resp.Results) == 0 || len(resp.Results[0].Entries) == 0 ||
		len(resp.Results[0].Entries[0].Value) == 0 {
		return DefaultParams(), nil
	}
	params := new(contract.CanoliqParams)
	if e := contract.Unmarshal(resp.Results[0].Entries[0].Value, params); e != nil {
		return nil, e
	}
	if err := ValidateParams(params); err != nil {
		return nil, err
	}
	return params, nil
}

// SaveParams writes the canoLiq parameters to state after validation.
func (c *Canoliq) SaveParams(params *contract.CanoliqParams) *contract.PluginError {
	if err := ValidateParams(params); err != nil {
		return err
	}
	bz, e := contract.Marshal(params)
	if e != nil {
		return e
	}
	if _, err := c.plugin.StateWrite(c, &contract.PluginStateWriteRequest{
		Sets: []*contract.PluginSetOp{{Key: KeyForParams(), Value: bz}},
	}); err != nil {
		return err
	}
	return nil
}

// LoadGlobals reads the singleton globals record, returning an empty struct
// if it is not yet present. Callers must persist any mutations via SaveGlobals.
func (c *Canoliq) LoadGlobals() (*contract.CanoliqGlobals, *contract.PluginError) {
	q := qid()
	resp, err := c.plugin.StateRead(c, &contract.PluginStateReadRequest{
		Keys: []*contract.PluginKeyRead{{QueryId: q, Key: KeyForGlobals()}},
	})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, resp.Error
	}
	g := new(contract.CanoliqGlobals)
	if len(resp.Results) > 0 && len(resp.Results[0].Entries) > 0 &&
		len(resp.Results[0].Entries[0].Value) > 0 {
		if e := contract.Unmarshal(resp.Results[0].Entries[0].Value, g); e != nil {
			return nil, e
		}
	}
	return g, nil
}

// SaveGlobals persists the globals record under the singleton key.
func (c *Canoliq) SaveGlobals(g *contract.CanoliqGlobals) *contract.PluginError {
	bz, e := contract.Marshal(g)
	if e != nil {
		return e
	}
	if _, err := c.plugin.StateWrite(c, &contract.PluginStateWriteRequest{
		Sets: []*contract.PluginSetOp{{Key: KeyForGlobals(), Value: bz}},
	}); err != nil {
		return err
	}
	return nil
}
