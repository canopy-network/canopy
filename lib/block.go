package lib

import (
	"bytes"
	"encoding/json"
	"github.com/canopy-network/canopy/lib/codec"
	"github.com/canopy-network/canopy/lib/crypto"
)

/*
	This file contains the default implementation of a 'Block' which is used as 'block bytes' in a quorum certificate
*/

// BLOCK HEADER CODE BELOW

// Check() 'sanity checks' the block header
func (x *BlockHeader) Check(networkID, chainId uint64) ErrorI {
	// rejects empty block header
	if x == nil {
		return ErrNilBlockHeader()
	}
	// check proposer address size
	if len(x.ProposerAddress) != crypto.AddressSize {
		return ErrInvalidBlockProposerAddress()
	}
	// check BlockHash size
	if len(x.Hash) != crypto.HashSize {
		return ErrWrongLengthBlockHash()
	}
	// check StateRoot hash size
	if len(x.StateRoot) != crypto.HashSize {
		return ErrWrongLengthStateRoot()
	}
	// check TransactionRoot hash size
	if len(x.TransactionRoot) != crypto.HashSize {
		return ErrWrongLengthTransactionRoot()
	}
	// check ValidatorRoot hash size
	if len(x.ValidatorRoot) != crypto.HashSize {
		return ErrWrongLengthValidatorRoot()
	}
	// check NextValidatorRoot hash size
	if len(x.NextValidatorRoot) != crypto.HashSize {
		return ErrWrongLengthNextValidatorRoot()
	}
	// check LastBlockHash hash size
	if len(x.LastBlockHash) != crypto.HashSize {
		return ErrWrongLengthLastBlockHash()
	}
	// if after the first block
	if x.Height > 1 {
		// check the LastQuorumCertificate
		if err := x.LastQuorumCertificate.CheckBasic(); err != nil {
			return err
		}
		// ensure the last quorum certificate has the proper network id
		if x.LastQuorumCertificate.Header.NetworkId != networkID {
			return ErrWrongNetworkID()
		}
		// check the last quorum certificate has the proper chain id
		if x.LastQuorumCertificate.Header.ChainId != chainId {
			return ErrWrongChainId()
		}
	}
	// check network id
	if uint64(x.NetworkId) != networkID {
		return ErrWrongNetworkID()
	}
	// check for non-zero BlockTime
	if x.Time == 0 {
		return ErrNilBlockTime()
	}
	// check for non-zero NetworkID
	if x.NetworkId == 0 {
		return ErrNilNetworkID()
	}
	// save hash in a temp variable
	tmp := x.Hash
	// set hash to nil
	x.Hash = nil
	// get the header bytes
	bz, err := Marshal(x)
	// if an error occurred when converting to bytes
	if err != nil {
		// exit with error
		return err
	}
	// reset the hash
	x.Hash = tmp
	// check got vs expected
	if !bytes.Equal(x.Hash, crypto.Hash(bz)) {
		return ErrMismatchHeaderBlockHash()
	}
	// exit
	return nil
}

// SetHash() computes and sets the BlockHash to BlockHeader.Hash
func (x *BlockHeader) SetHash() ([]byte, ErrorI) {
	// set the hash to empty
	x.Hash = nil
	// convert the block header object reference to bytes
	bz, err := Marshal(x)
	// if an error occurred during the bytes conversion
	if err != nil {
		// exit with error
		return nil, err
	}
	// set the hash to the hash of the block header bytes
	x.Hash = crypto.Hash(bz)
	// exit with the block header bytes
	return x.Hash, nil
}

// jsonBlockHeader is the BlockHeader implementation of json.Marshaller and json.Unmarshaler
type jsonBlockHeader struct {
	Height                uint64             `json:"height,omitempty"`
	Hash                  HexBytes           `json:"hash,omitempty"`
	NetworkId             uint32             `json:"networkID,omitempty"`
	Time                  uint64             `json:"time,omitempty"`
	NumTxs                uint64             `json:"numTxs,omitempty"`
	TotalTxs              uint64             `json:"totalTxs,omitempty"`
	TotalVdfIterations    uint64             `json:"totalVDFIterations,omitempty"`
	LastBlockHash         HexBytes           `json:"lastBlockHash,omitempty"`
	StateRoot             HexBytes           `json:"stateRoot,omitempty"`
	TransactionRoot       HexBytes           `json:"transactionRoot,omitempty"`
	ValidatorRoot         HexBytes           `json:"validatorRoot,omitempty"`
	NextValidatorRoot     HexBytes           `json:"nextValidatorRoot,omitempty"`
	ProposerAddress       HexBytes           `json:"proposerAddress,omitempty"`
	VDF                   *crypto.VDF        `json:"vdf,omitempty"`
	LastQuorumCertificate *QuorumCertificate `json:"lastQuorumCertificate,omitempty"`
}

// MarshalJSON() implements the json.Marshaller interface
func (x BlockHeader) MarshalJSON() ([]byte, error) {
	return json.Marshal(jsonBlockHeader{
		Height:                x.Height,
		Hash:                  x.Hash,
		NetworkId:             x.NetworkId,
		Time:                  x.Time,
		NumTxs:                x.NumTxs,
		TotalTxs:              x.TotalTxs,
		TotalVdfIterations:    x.TotalVdfIterations,
		LastBlockHash:         x.LastBlockHash,
		StateRoot:             x.StateRoot,
		TransactionRoot:       x.TransactionRoot,
		ValidatorRoot:         x.ValidatorRoot,
		NextValidatorRoot:     x.NextValidatorRoot,
		ProposerAddress:       x.ProposerAddress,
		VDF:                   x.Vdf,
		LastQuorumCertificate: x.LastQuorumCertificate,
	})
}

// UnmarshalJSON() implements the json.Unmarshaler interface
func (x *BlockHeader) UnmarshalJSON(b []byte) (err error) {
	// initialize a new object reference for the block header
	j := new(jsonBlockHeader)
	// convert the block header object reference to bytes
	if err = json.Unmarshal(b, &j); err != nil {
		// exit with error
		return
	}
	// set the underlying object to the new block header
	*x = BlockHeader{
		Height:                j.Height,
		Hash:                  j.Hash,
		NetworkId:             j.NetworkId,
		Time:                  j.Time,
		NumTxs:                j.NumTxs,
		TotalTxs:              j.TotalTxs,
		TotalVdfIterations:    j.TotalVdfIterations,
		LastBlockHash:         j.LastBlockHash,
		StateRoot:             j.StateRoot,
		TransactionRoot:       j.TransactionRoot,
		ValidatorRoot:         j.ValidatorRoot,
		NextValidatorRoot:     j.NextValidatorRoot,
		ProposerAddress:       j.ProposerAddress,
		Vdf:                   j.VDF,
		LastQuorumCertificate: j.LastQuorumCertificate,
	}
	// exit
	return
}

// BLOCK CODE BELOW

// Check() 'sanity checks' the Block structure
func (x *Block) Check(networkID, chainId uint64) ErrorI {
	// ensure the block is non nil
	if x == nil {
		// exit with nil error
		return ErrNilBlock()
	}
	// check for unknown fields
	//if codec.HasUnknown(x) { TODO add once block structure is stable
	//	// exit with unknown fields error
	//	return ErrUnknownFields()
	//}
	// check the header
	return x.BlockHeader.Check(networkID, chainId)
}

// Hash() computes, sets, and returns the BlockHash
func (x *Block) Hash() ([]byte, ErrorI) { return x.BlockHeader.SetHash() }

// BytesToBlockHash() converts block bytes into a block hash
func (x *Block) BytesToBlockHash(blockBytes []byte) (hash []byte, err ErrorI) {
	// ensure the block isn't empty
	if blockBytes == nil {
		// exit with error
		return nil, ErrNilBlock()
	}
	// get the block header field
	blockHeaderWithHash, e := codec.GetRawProtoField(blockBytes, 1)
	if e != nil {
		return nil, ErrProtoParse(e)
	}
	// nullify the hash included in the block header
	blockHeaderWithoutHash, e := codec.NullifyProtoField(blockHeaderWithHash, 2)
	if e != nil {
		return nil, ErrProtoParse(e)
	}
	// exit
	return crypto.Hash(blockHeaderWithoutHash), nil
}

// jsonBlock is the Block implementation of json.Marshaller and json.Unmarshaler
type jsonBlock struct {
	BlockHeader  *BlockHeader `json:"blockHeader,omitempty"`
	Transactions []HexBytes   `json:"transactions,omitempty"`
}

// MarshalJSON() implements the json.Marshaller interface
func (x Block) MarshalJSON() ([]byte, error) {
	// create a list of hex bytes
	var txs []HexBytes
	// for each transaction in the block
	for _, tx := range x.Transactions {
		// add the transaction to the list
		// converting it to hex bytes
		txs = append(txs, tx)
	}
	// convert the block into json bytes
	return json.Marshal(jsonBlock{
		BlockHeader:  x.BlockHeader,
		Transactions: txs,
	})
}

// UnmarshalJSON() implements the json.Unmarshaler interface
func (x *Block) UnmarshalJSON(blockBytes []byte) (err error) {
	// create a new object reference for the json block
	j := new(jsonBlock)
	// populate the json block reference with the block bytes
	if err = json.Unmarshal(blockBytes, j); err != nil {
		// exit with error
		return
	}
	// create a list of bytes
	var txs [][]byte
	// for each transaction in the json transaction object
	for _, hexTx := range j.Transactions {
		// add it to the list
		// converting it to regular bytes
		txs = append(txs, hexTx)
	}
	// populate the structure with a header and transaction list
	x.BlockHeader, x.Transactions = j.BlockHeader, txs
	// exit
	return
}

// BLOCK RESULTS CODE BELOW

// BlockResults is a collection of Blocks containing their TransactionResults and Meta after commitment
type BlockResults []*BlockResult

// New() Satisfies the pageable interface
func (b *BlockResults) New() Pageable { return &BlockResults{} }

// ToBlock() converts the BlockResult into a Block object
func (x *BlockResult) ToBlock() (*Block, ErrorI) {
	// create a list of bytes
	var txs [][]byte
	// for each transaction in the block results structure
	for _, txResult := range x.Transactions {
		// convert the transaction object to bytes
		txBytes, err := Marshal(txResult.Transaction)
		// if an error occurred during the conversion
		if err != nil {
			// exit with error
			return nil, err
		}
		// add the bytes to the transaction list
		txs = append(txs, txBytes)
	}
	// exit with the populated block structure
	return &Block{
		BlockHeader:  x.BlockHeader,
		Transactions: txs,
	}, nil
}

const (
	BlockResultsPageName = "block-results-page" // BlockResults as a pageable name
)

func init() {
	RegisteredPageables[BlockResultsPageName] = new(BlockResults) // register BlockResults as a pageable
}

var _ Pageable = new(BlockResults) // Pageable interface enforcement
