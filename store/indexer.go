package store

import (
	"bytes"
	"encoding/binary"
	"errors"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/ethereum/go-ethereum/core/types"
	lru "github.com/hashicorp/golang-lru/v2"
)

var _ lib.RWIndexerI = &Indexer{}

var (
	txHashPrefix       = []byte{1}  // store key prefix for transaction by hash
	txHeightPrefix     = []byte{2}  // store key prefix for transactions by height
	txSenderPrefix     = []byte{3}  // store key prefix for transactions from sender
	txRecipientPrefix  = []byte{4}  // store key prefix for transaction by recipient
	blockHashPrefix    = []byte{5}  // store key prefix for block by hash
	blockHeightPrefix  = []byte{6}  // store key prefix for block by height
	qcHeightPrefix     = []byte{7}  // store key prefix for quorum certificate by height
	doubleSignerPrefix = []byte{8}  // store key prefix for double signers by height
	checkPointPrefix   = []byte{9}  // store key prefix for checkpoints for committee chains
	eventAddressPrefix = []byte{10} // store key prefix for events by address
	eventHeightPrefix  = []byte{11} // store key prefix for events by block height
	eventChainIdPrefix = []byte{12} // store key prefix for events by chainId
	eventHashPrefix    = []byte{13} // store key prefix for events by event hash (concept just used for indexing)
	stateChangePrefix  = []byte{14} // state keys written at a particular committed version
	// create indexer cache
	blockCache, _ = lru.New[uint64, *lib.BlockResult](64)
	//qcCache, _ = lru.New[uint64, *lib.QuorumCertificate](4) TODO add back
)

// The marker distinguishes two cases that otherwise look identical:
//
//  1. The version was journaled, but no account keys changed.
//  2. The version predates journaling, so no journal data exists.
var stateChangeMarker = []byte{1}

// Indexer: the part of the DB that stores transactions, blocks, and quorum certificates
type Indexer struct {
	db     *Txn
	config lib.Config
}

// StateChangeKeys() returns state keys written while committing version, optionally
// restricted to a state-key prefix. The available result distinguishes a
// journaled version with no matching changes from a pre-journal version
func (t *Indexer) StateChangeKeys(version uint64, prefix []byte) (keys [][]byte, available bool, err lib.ErrorI) {
	// retrieve the state change version prefix
	versionPrefix := t.stateChangeVersionPrefix(version)
	// retrieve the marker
	marker, err := t.db.Get(versionPrefix)
	if err != nil || len(marker) == 0 {
		return nil, false, err
	}
	// retrieve the search prefix
	searchPrefix := lib.Append(versionPrefix, prefix)
	// iterate through that prefix
	it, err := t.db.Iterator(searchPrefix)
	if err != nil {
		return nil, false, err
	}
	defer it.Close()
	// for each key
	for ; it.Valid(); it.Next() {
		k := it.Key()
		if len(k) <= len(versionPrefix) {
			continue
		}
		// append to key list
		keys = append(keys, bytes.Clone(k[len(versionPrefix):]))
	}
	return keys, true, nil
}

// indexStateChangeKeys() records the commit marker even when keys is empty so
// readers can safely use an empty delta without falling back to a full scan
func (t *Indexer) indexStateChangeKeys(version uint64, keys [][]byte) lib.ErrorI {
	versionPrefix := t.stateChangeVersionPrefix(version)
	if err := t.db.Set(versionPrefix, stateChangeMarker); err != nil {
		return err
	}
	for _, k := range keys {
		if err := t.db.Set(lib.Append(versionPrefix, k), stateChangeMarker); err != nil {
			return err
		}
	}
	return nil
}

// stateChangeVersionPrefix() returns the stateChangePrefix + version (big endian)
func (t *Indexer) stateChangeVersionPrefix(version uint64) []byte {
	return t.key(stateChangePrefix, t.encodeBigEndian(version), nil)
}

// BLOCKS CODE BELOW

// IndexBlock() turns the block into bytes, indexes the block by hash and height
// and then indexes the transactions
func (t *Indexer) IndexBlock(b *lib.BlockResult) lib.ErrorI {
	// marshal result to get the size
	resultBz, err := lib.Marshal(b)
	if err != nil {
		return err
	}
	// set meta stats for the block
	b.Meta = &lib.BlockResultMeta{Size: uint64(len(resultBz))}
	blockCache.Add(b.BlockHeader.Height, b)
	// get bytes of block header
	bz, err := lib.Marshal(b.BlockHeader)
	if err != nil {
		return err
	}
	var eg errgroup.Group
	eg.SetLimit(32)
	// index block header in its own goroutine
	eg.Go(func() error {
		hashKey, err := t.indexBlockByHash(b.BlockHeader.Hash, bz)
		if err != nil {
			return err
		}
		return t.indexBlockByHeight(b.BlockHeader.Height, hashKey)
	})
	// index transactions in parallel
	for _, transaction := range b.Transactions {
		tx := transaction // capture range variable
		eg.Go(func() error {
			return t.IndexTx(tx)
		})
	}
	// index events in parallel
	for i, event := range b.Events {
		e, idx := event, i // capture range variable
		eg.Go(func() error {
			return t.IndexEvent(e, idx)
		})
	}
	// wait for all goroutines to finish
	if err := eg.Wait(); err != nil {
		return ErrIndexBlock(err)
	}
	return nil
}

// DeleteBlockForHeight() deletes the block & transaction data for a certain height
func (t *Indexer) DeleteBlockForHeight(height uint64) lib.ErrorI {
	// remove from cache
	blockCache.Remove(height)
	// get the height key
	heightKey := t.blockHeightKey(height)
	// get the hash key (was indexed by height key)
	hashKey, err := t.db.Get(heightKey)
	if err != nil {
		return err
	}
	// delete the reference to the hash key
	if err = t.db.Delete(heightKey); err != nil {
		return err
	}
	// delete the transactions for the height
	if err = t.DeleteTxsForHeight(height); err != nil {
		return err
	}
	// delete the header by the hash key
	return t.db.Delete(hashKey)
}

// GetBlockByHash() returns the block result object from the hash key
func (t *Indexer) GetBlockByHash(hash []byte) (*lib.BlockResult, lib.ErrorI) {
	return t.getBlock(t.blockHashKey(hash), true)
}

// GetBlockByHeight() returns the block result by height key
func (t *Indexer) GetBlockByHeight(height uint64) (*lib.BlockResult, lib.ErrorI) {
	// check cache
	if got, found := blockCache.Get(height); found {
		return got, nil
	}
	// height key points to hash key
	hashKey, err := t.db.Get(t.blockHeightKey(height))
	if err != nil {
		return nil, err
	}
	// get block from hash key
	block, err := t.getBlock(hashKey, true)
	if err != nil {
		return nil, err
	}
	// populate cache on read so historical blocks are warm after a restart
	blockCache.Add(height, block)
	return block, nil
}

// GetBlockHeaderByHeight() returns the block result without transactions
func (t *Indexer) GetBlockHeaderByHeight(height uint64) (*lib.BlockResult, lib.ErrorI) {
	// check cache (full block result may be cached from GetBlockByHeight or IndexBlock)
	if got, found := blockCache.Get(height); found {
		return got, nil
	}
	// height key points to hash key
	hashKey, err := t.db.Get(t.blockHeightKey(height))
	if err != nil {
		return nil, err
	}
	// get block from hash key
	block, err := t.getBlock(hashKey, false)
	if err != nil {
		return nil, err
	}
	// populate cache on read so historical blocks are warm after a restart
	blockCache.Add(height, block)
	return block, nil
}

// GetBlocks() returns a page of blocks based on the page parameters
// blocks are indexed contiguously by height, so the page params are derived from the
// oldest and newest indexed heights instead of walking the entire block index
func (t *Indexer) GetBlocks(p lib.PageParams) (page *lib.Page, err lib.ErrorI) {
	if p.PerPage > 100 {
		p.PerPage = 100
	}
	results, page := make(lib.BlockResults, 0), lib.NewPage(p, lib.BlockResultsPageName)
	// get the height boundaries of the index
	oldest, newest, found, err := t.blockHeightBounds()
	if err != nil {
		return
	}
	// the total count is the size of the height range (0 when nothing is indexed)
	totalCount := 0
	if found {
		totalCount = int(newest - oldest + 1)
	}
	// blocks are ordered newest to oldest, so the item at index i is the block at newest-i
	err = page.LoadCounted(totalCount, &results, func(index int) lib.ErrorI {
		block, e := t.getBlockForPage(newest-uint64(index), true)
		if e != nil {
			return e
		}
		// a cached block result carries the size of the block including its events, so the
		// size is recalculated to report the same value whether or not the block was cached
		size, e := blockResultSize(block)
		if e != nil {
			return e
		}
		// the block result may be shared with the block cache, so a shallow copy holding its
		// own metadata is added to the page to keep the 'took' calculation from mutating it
		results = append(results, &lib.BlockResult{
			BlockHeader:  block.BlockHeader,
			Transactions: block.Transactions,
			Events:       block.Events,
			Meta:         &lib.BlockResultMeta{Size: size},
		})
		return nil
	})
	if err != nil {
		return
	}
	// fill in the block time metadata now that the page is loaded
	err = t.setBlocksTook(results, oldest)
	return
}

// blockHeightBounds() returns the oldest and newest heights present in the block index
func (t *Indexer) blockHeightBounds() (oldest, newest uint64, found bool, err lib.ErrorI) {
	// seek to the highest indexed height
	newest, found, err = t.seekBlockHeight(true)
	// exit early if the index is empty or errored
	if err != nil || !found {
		return
	}
	// seek to the lowest indexed height
	oldest, found, err = t.seekBlockHeight(false)
	return
}

// seekBlockHeight() returns the first height in the block index in the requested direction
func (t *Indexer) seekBlockHeight(newest bool) (height uint64, found bool, err lib.ErrorI) {
	var it lib.IteratorI
	if newest {
		it, err = t.db.RevIterator(lib.JoinLenPrefix(blockHeightPrefix))
	} else {
		it, err = t.db.Iterator(lib.JoinLenPrefix(blockHeightPrefix))
	}
	if err != nil {
		return
	}
	defer it.Close()
	// no blocks are indexed
	if !it.Valid() {
		return
	}
	// extract the height from the key whose layout is <blockHeightPrefix><height>
	segments := lib.DecodeLengthPrefixed(it.Key())
	if len(segments) != 2 {
		return 0, false, ErrInvalidKey()
	}
	return t.decodeBigEndian(segments[1]), true, nil
}

// setBlocksTook() fills the 'took' metadata of each block using the delta between its time
// and the time of the block below it; the results are ordered newest to oldest.
func (t *Indexer) setBlocksTook(results lib.BlockResults, oldest uint64) lib.ErrorI {
	for i, block := range results {
		var previousTime uint64
		if i+1 < len(results) {
			// the next result is the block directly below this one
			previousTime = results[i+1].BlockHeader.Time
		} else {
			// the last result of the page needs the block below the page
			height := block.BlockHeader.Height
			// the oldest indexed block has nothing below it to compare against
			if oldest >= height {
				continue
			}
			// only the header is needed since the block below the page isn't part of the results
			previous, err := t.getBlockForPage(height-1, false)
			if err != nil {
				return err
			}
			previousTime = previous.BlockHeader.Time
		}
		// calculate and set block "took" time
		blockTime := time.UnixMicro(int64(block.BlockHeader.Time))
		prevBlkTime := time.UnixMicro(int64(previousTime))
		block.Meta.Took = uint64(blockTime.Sub(prevBlkTime).Milliseconds())
	}
	return nil
}

// blockResultSize() returns the size of the block header and its transactions, matching the
// size getBlock() reports for a block that wasn't served by the cache
func blockResultSize(block *lib.BlockResult) (uint64, lib.ErrorI) {
	bz, err := lib.Marshal(&lib.BlockResult{
		BlockHeader:  block.BlockHeader,
		Transactions: block.Transactions,
	})
	if err != nil {
		return 0, err
	}
	return uint64(len(bz)), nil
}

// getBlockForPage() returns the block at the height
func (t *Indexer) getBlockForPage(height uint64, transactions bool) (*lib.BlockResult, lib.ErrorI) {
	// use the cached block result if it's already loaded
	if got, found := blockCache.Get(height); found {
		return got, nil
	}
	// height key points to hash key
	hashKey, err := t.db.Get(t.blockHeightKey(height))
	if err != nil {
		return nil, err
	}
	// get the block from the hash key
	return t.getBlock(hashKey, transactions)
}

// QUORUM CERTIFICATE CODE BELOW

// IndexQC() indexes the quorum certificate by height
func (t *Indexer) IndexQC(qc *lib.QuorumCertificate) lib.ErrorI {
	// add to cache
	//t.qcCache.Add(qc.Header.Height, qc)
	// convert to bytes
	bz, err := lib.Marshal(&lib.QuorumCertificate{
		Header:      qc.Header,
		Results:     qc.Results,
		ResultsHash: qc.ResultsHash,
		BlockHash:   qc.BlockHash,
		ProposerKey: qc.ProposerKey,
		Signature:   qc.Signature,
	})
	if err != nil {
		return err
	}
	return t.indexQCByHeight(qc.Header.Height, bz)
}

// GetQCByHeight() returns the quorum certificate by height key
func (t *Indexer) GetQCByHeight(height uint64) (*lib.QuorumCertificate, lib.ErrorI) {
	// check cache
	//if qc, found := t.qcCache.Get(height); found && qc.Block != nil {
	//	return qc, nil
	//}
	// unlike blocks, QCs are stored by hash key
	qc, err := t.getQC(t.qcHeightKey(height))
	if err != nil {
		return nil, err
	}
	// get the block by height key
	blkResult, err := t.GetBlockByHeight(height)
	if err != nil {
		return nil, err
	}
	// just take the block part of the result
	block, err := blkResult.ToBlock()
	if err != nil {
		return nil, err
	}
	// store it in the proposal object as bytes
	qc.Block, err = lib.Marshal(block)
	if err != nil {
		return nil, err
	}
	return qc, err
}

// DeleteQCForHeight() deletes the Quorum Certificate by height
func (t *Indexer) DeleteQCForHeight(height uint64) lib.ErrorI {
	//t.qcCache.Remove(height)
	return t.db.Delete(t.qcHeightKey(height))
}

// TRANSACTION CODE BELOW

// IndexTx() indexes the transaction by hash, height, sender and receiver
// the tx bytes is indexed by hash and then that hash is indexed by height, sender, and receiver
func (t *Indexer) IndexTx(result *lib.TxResult) lib.ErrorI {
	// convert the tx to bytes
	bz, err := lib.Marshal(result)
	if err != nil {
		return err
	}
	hashes, err := indexedTxHashes(result)
	if err != nil {
		return err
	}
	hashKey, err := t.indexTxByHash(hashes[0], bz)
	if err != nil {
		return err
	}
	for _, hash := range hashes[1:] {
		if _, err = t.indexTxByHash(hash, bz); err != nil {
			return err
		}
	}
	// store the hash key by height.index
	heightAndIndexKey := t.txHeightAndIndexKey(result.GetHeight(), result.GetIndex())
	if err = t.indexTxByHeightAndIndex(heightAndIndexKey, hashKey); err != nil {
		return err
	}
	// index by accounts indicates if the indexer should index by sender/receiver
	if t.config.IndexByAccount {
		// store the hash key by sender
		if err = t.indexTxBySender(result.GetSender(), heightAndIndexKey, hashKey); err != nil {
			return err
		}

		// store the hash key by recipient
		if err = t.indexTxByRecipient(result.GetRecipient(), heightAndIndexKey, hashKey); err != nil {
			return err
		}
	}
	return nil
}

// indexedTxHashes() returns the primary Canopy tx hash plus any persisted lookup aliases.
func indexedTxHashes(result *lib.TxResult) ([][]byte, lib.ErrorI) {
	hash, err := lib.StringToBytes(result.GetTxHash())
	if err != nil {
		return nil, err
	}
	hashes := [][]byte{hash}
	ethHash := ethTxHash(result.Transaction)
	if len(ethHash) != 0 && !bytes.Equal(ethHash, hash) {
		hashes = append(hashes, ethHash)
	}
	intentID, err := result.Transaction.GetMultisigIntentID()
	if err != nil {
		return nil, err
	}
	if len(intentID) != 0 && !bytes.Equal(intentID, hash) && !bytes.Equal(intentID, ethHash) {
		hashes = append(hashes, intentID)
	}
	return hashes, nil
}

// ethTxHash() returns the canonical Ethereum tx hash for an RLP-backed transaction.
func ethTxHash(tx *lib.Transaction) []byte {
	if tx == nil || !lib.IsRLPMemo(tx.Memo) || tx.Signature == nil || len(tx.Signature.Signature) == 0 {
		return nil
	}
	var ethTx types.Transaction
	if err := ethTx.UnmarshalBinary(tx.Signature.Signature); err != nil {
		return nil
	}
	return ethTx.Hash().Bytes()
}

// GetTxByHash() returns the tx by hash
func (t *Indexer) GetTxByHash(hash []byte) (*lib.TxResult, lib.ErrorI) {
	return t.getTx(t.txHashKey(hash))
}

// GetTxsByHeight() returns a page of transactions for a height
func (t *Indexer) GetTxsByHeight(height uint64, newestToOldest bool, p lib.PageParams) (*lib.Page, lib.ErrorI) {
	return t.getTxs(t.txHeightKey(height), newestToOldest, p)
}

// GetTxsByHeightNonPaginated() returns a slice of transactions ordered by index for a height
func (t *Indexer) GetTxsByHeightNonPaginated(height uint64, newestToOldest bool) ([]*lib.TxResult, lib.ErrorI) {
	return t.getTxsNonPaginated(t.txHeightKey(height), newestToOldest)
}

// GetTxsBySender() returns a slice of transactions ordered by height and index for a sender
func (t *Indexer) GetTxsBySender(address crypto.AddressI, newestToOldest bool, p lib.PageParams) (*lib.Page, lib.ErrorI) {
	return t.getTxs(t.txSenderKey(address.Bytes(), nil), newestToOldest, p)
}

// GetTxsByRecipient() returns a slice of transactions ordered by height and index for a recipient
func (t *Indexer) GetTxsByRecipient(address crypto.AddressI, newestToOldest bool, p lib.PageParams) (*lib.Page, lib.ErrorI) {
	return t.getTxs(t.txRecipientKey(address.Bytes(), nil), newestToOldest, p)
}

// DeleteTxsForHeight() deletes the transaction object for a specific height
func (t *Indexer) DeleteTxsForHeight(height uint64) lib.ErrorI {
	txs, err := t.GetTxsByHeightNonPaginated(height, false)
	if err != nil {
		return err
	}
	for _, tx := range txs {
		heightAndIndexKey := t.txHeightAndIndexKey(tx.GetHeight(), tx.GetIndex())
		hashes, e := indexedTxHashes(tx)
		if e != nil {
			return e
		}
		for _, hash := range hashes {
			if e = t.db.Delete(t.txHashKey(hash)); e != nil {
				return e
			}
		}
		if t.config.IndexByAccount {
			if e = t.db.Delete(t.txSenderKey(tx.GetSender(), heightAndIndexKey)); e != nil {
				return e
			}
			if recipient := tx.GetRecipient(); recipient != nil {
				if e = t.db.Delete(t.txRecipientKey(recipient, heightAndIndexKey)); e != nil {
					return e
				}
			}
		}
	}
	if err = t.deleteAll(t.txHeightKey(height)); err != nil {
		return err
	}
	return nil
}

// DOUBLE SIGNER CODE BELOW

// IndexDoubleSigner() indexes the double signer by a height
func (t *Indexer) IndexDoubleSigner(address []byte, height uint64) lib.ErrorI {
	return t.indexDoubleSignerByHeight(address, height)
}

// GetDoubleSigners() gets all double signers saved in the indexer
// IMPORTANT NOTE: this returns double signers in the form of <address> -> <heights> NOT <public_key> -> <heights>
func (t *Indexer) GetDoubleSigners() (ds []*lib.DoubleSigner, err lib.ErrorI) {
	it, err := t.db.Iterator(lib.JoinLenPrefix(doubleSignerPrefix))
	if err != nil {
		return nil, err
	}
	defer it.Close()
	results := make(map[string][]uint64)
	for ; it.Valid(); it.Next() {
		// get the segments of the key
		segments := lib.DecodeLengthPrefixed(it.Key())
		// sanity check the key
		if len(segments) < 3 {
			return nil, ErrInvalidKey()
		}
		// key split should be dsPrefix / height / address
		address, height := segments[1], t.decodeBigEndian(segments[2])
		// add to results
		results[lib.BytesToString(address)] = append(results[lib.BytesToString(address)], height)
	}
	for address, heights := range results {
		addr, e := lib.StringToBytes(address)
		if e != nil {
			return nil, e
		}
		ds = append(ds, &lib.DoubleSigner{
			Id:      addr,
			Heights: heights,
		})
	}
	return
}

// GetDoubleSignersAsOf() gets double signers in the indexer up to and including the provided height
// IMPORTANT NOTE: this returns double signers in the form of <address> -> <heights> NOT <public_key> -> <heights>
func (t *Indexer) GetDoubleSignersAsOf(height uint64) (ds []*lib.DoubleSigner, err lib.ErrorI) {
	it, err := t.db.Iterator(lib.JoinLenPrefix(doubleSignerPrefix))
	if err != nil {
		return nil, err
	}
	defer it.Close()
	results := make(map[string][]uint64)
	for ; it.Valid(); it.Next() {
		segments := lib.DecodeLengthPrefixed(it.Key())
		if len(segments) < 3 {
			return nil, ErrInvalidKey()
		}
		address, dsHeight := segments[1], t.decodeBigEndian(segments[2])
		if dsHeight > height {
			continue
		}
		results[lib.BytesToString(address)] = append(results[lib.BytesToString(address)], dsHeight)
	}
	for address, heights := range results {
		addr, e := lib.StringToBytes(address)
		if e != nil {
			return nil, e
		}
		ds = append(ds, &lib.DoubleSigner{
			Id:      addr,
			Heights: heights,
		})
	}
	return
}

// IsValidDoubleSigner() checks if the double signer byte is set for a height
func (t *Indexer) IsValidDoubleSigner(address []byte, height uint64) (bool, lib.ErrorI) {
	bz, err := t.db.Get(t.doubleSignerHeightKey(address, height))
	if err != nil {
		return false, err
	}
	return !bytes.Equal(bz, doubleSignerPrefix), nil
}

// EVENTS CODE BELOW

// IndexEvent() indexes the event by hash, height, address and chainId
func (t *Indexer) IndexEvent(e *lib.Event, index int) lib.ErrorI {
	// index the event by hash
	hashKey, err := t.indexEventByHash(e)
	if err != nil {
		return err
	}
	// index the event by height and index
	heightAndIndexKey := t.eventHeightAndIndexKey(e.Height, uint64(index))
	// store the hash key by height.index
	if err = t.indexEventByHeightAndIndex(heightAndIndexKey, hashKey); err != nil {
		return err
	}
	// index by chain id
	if e.ChainId != 0 {
		// store the hash key by chainId
		if err = t.indexEventByChainId(e.ChainId, heightAndIndexKey, hashKey); err != nil {
			return err
		}
	}
	// index by accounts indicates if the indexer should index by address
	if t.config.IndexByAccount && e.Address != nil {
		// store the hash key by address
		if err = t.indexEventByAddress(e.Address, heightAndIndexKey, hashKey); err != nil {
			return err
		}
	}
	return nil
}

// GetEventsByAddress() returns a slice of events ordered by height and index for an address
func (t *Indexer) GetEventsByAddress(address crypto.AddressI, newestToOldest bool, p lib.PageParams) (*lib.Page, lib.ErrorI) {
	return t.getEvents(t.eventAddressKey(address.Bytes(), nil), newestToOldest, p)
}

// GetEventsByBlockHeight() returns a slice of events ordered by height and index for a block height
func (t *Indexer) GetEventsByBlockHeight(blockHeight uint64, newestToOldest bool, p lib.PageParams) (*lib.Page, lib.ErrorI) {
	return t.getEvents(t.eventBlockHeightKey(blockHeight), newestToOldest, p)
}

// GetEventsByChainId() returns a slice of events ordered by chainId for an event type
func (t *Indexer) GetEventsByChainId(chainId uint64, newestToOldest bool, p lib.PageParams) (*lib.Page, lib.ErrorI) {
	return t.getEvents(t.eventChainIdKey(chainId, nil), newestToOldest, p)
}

// GetEventsNonPaginated() returns a slice of events ordered by index for a height
func (t *Indexer) GetEventsNonPaginated(height uint64, newestToOldest bool) ([]*lib.Event, lib.ErrorI) {
	return t.getEventsNonPaginated(t.eventHeightKey(height), newestToOldest)
}

// getEvents() returns a page of events in sorted order by block height
func (t *Indexer) getEvents(prefix []byte, newestToOldest bool, p lib.PageParams) (page *lib.Page, err lib.ErrorI) {
	events, page := make(lib.Events, 0), lib.NewPage(p, "events-page")
	err = page.Load(prefix, newestToOldest, &events, t.db, func(_, b []byte) (e lib.ErrorI) {
		tx, e := t.getEvent(b)
		if e == nil {
			events = append(events, tx)
		}
		return
	})
	return
}

// getEventsNonPaginated() gets the events in sorted order by block.index in a slice format
func (t *Indexer) getEventsNonPaginated(prefix []byte, newestToOldest bool) (results []*lib.Event, err lib.ErrorI) {
	var it lib.IteratorI
	switch newestToOldest {
	case true:
		it, err = t.db.RevIterator(prefix)
	case false:
		it, err = t.db.Iterator(prefix)
	}
	if err != nil {
		return nil, err
	}
	defer it.Close()
	for ; it.Valid(); it.Next() {
		tx, e := t.getEvent(it.Value())
		if e != nil {
			return nil, e
		}
		results = append(results, tx)
	}
	return
}

// getEvent() gets the event bytes from the DB and converts it into Event object
func (t *Indexer) getEvent(hashKey []byte) (*lib.Event, lib.ErrorI) {
	bz, err := t.db.Get(hashKey)
	if err != nil {
		return nil, err
	}
	if len(bz) == 0 {
		return nil, ErrStoreGet(errors.New("event not found"))
	}
	ptr := new(lib.Event)
	if err = lib.Unmarshal(bz, ptr); err != nil {
		return nil, err
	}
	return ptr, nil
}

// indexEventByHash() indexes an event by its hash
func (t *Indexer) indexEventByHash(e *lib.Event) (hashKey []byte, err lib.ErrorI) {
	bz, err := lib.Marshal(e)
	if err != nil {
		return nil, err
	}
	k := t.key(eventHashPrefix, crypto.Hash(bz), nil)
	return k, t.db.Set(k, bz)
}

func (t *Indexer) indexEventByAddress(address, heightAndIndexKey []byte, bz []byte) lib.ErrorI {
	return t.db.Set(t.eventAddressKey(address, heightAndIndexKey), bz)
}

func (t *Indexer) indexEventByBlockHeight(blockHeight uint64, bz []byte) (err lib.ErrorI) {
	k := t.eventBlockHeightKey(blockHeight)
	return t.db.Set(k, bz)
}

func (t *Indexer) indexEventByChainId(chainId uint64, blockHeightAndIdxkey, bz []byte) (err lib.ErrorI) {
	return t.db.Set(t.eventChainIdKey(chainId, blockHeightAndIdxkey), bz)
}

func (t *Indexer) eventChainIdKey(chainId uint64, heightAndIndexKey []byte) []byte {
	return t.key(eventChainIdPrefix, t.encodeBigEndian(chainId), heightAndIndexKey)
}

func (t *Indexer) eventBlockHeightKey(blockHeight uint64) []byte {
	return t.key(eventHeightPrefix, t.encodeBigEndian(blockHeight), nil)
}

func (t *Indexer) eventAddressKey(address, heightAndIndexKey []byte) []byte {
	return t.key(eventAddressPrefix, address, heightAndIndexKey)
}

// CHECKPOINT CODE BELOW

// IndexCheckpoint() indexes a 'checkpoint block hash' for a committee chain at a certain height
// this is for the 'checkpointing as a service' long-range-attack prevention
func (t *Indexer) IndexCheckpoint(chainId uint64, checkpoint *lib.Checkpoint) lib.ErrorI {
	return t.db.Set(t.checkpointKey(chainId, checkpoint.Height), checkpoint.BlockHash)
}

// GetCheckpoint() retrieves a 'checkpoint block hash' for a committee chain at a certain height
// this is for the 'checkpointing as a service' long-range-attack prevention
func (t *Indexer) GetCheckpoint(chainId, height uint64) (blockHash lib.HexBytes, err lib.ErrorI) {
	return t.db.Get(t.checkpointKey(chainId, height))
}

// GetMostRecentCheckpoint() retrieves a 'checkpoint block hash' for a committee chain at the most recent height
// this is for the 'checkpointing as a service' long-range-attack prevention
func (t *Indexer) GetMostRecentCheckpoint(chainId uint64) (checkpoint *lib.Checkpoint, err lib.ErrorI) {
	it, err := t.db.RevIterator(t.checkpointsCommitteeKey(chainId))
	if err != nil {
		return
	}
	defer it.Close()
	if !it.Valid() {
		return &lib.Checkpoint{
			Height:    0,
			BlockHash: nil,
		}, nil
	}
	return t.checkpointFromKeyValue(it.Key(), it.Value())
}

// GetAllCheckpoints() exports all 'checkpoint block hashes' for a committee chain
// this is for the 'checkpointing as a service' long-range-attack prevention
func (t *Indexer) GetAllCheckpoints(chainId uint64) (checkpoints []*lib.Checkpoint, err lib.ErrorI) {
	it, err := t.db.Iterator(t.checkpointsCommitteeKey(chainId))
	if err != nil {
		return
	}
	defer it.Close()
	for ; it.Valid(); it.Next() {
		checkPoint, e := t.checkpointFromKeyValue(it.Key(), it.Value())
		if e != nil {
			return nil, e
		}
		checkpoints = append(checkpoints, checkPoint)
	}
	return
}

// DeleteCheckpointsForChain() removes all checkpoint records for a committee chain
func (t *Indexer) DeleteCheckpointsForChain(chainId uint64) (err lib.ErrorI) {
	it, err := t.db.Iterator(t.checkpointsCommitteeKey(chainId))
	if err != nil {
		return err
	}
	defer it.Close()
	var keys [][]byte
	for ; it.Valid(); it.Next() {
		key := make([]byte, len(it.Key()))
		copy(key, it.Key())
		keys = append(keys, key)
	}
	for _, key := range keys {
		if err = t.db.Delete(key); err != nil {
			return err
		}
	}
	return nil
}

func (t *Indexer) checkpointFromKeyValue(key, value []byte) (*lib.Checkpoint, lib.ErrorI) {
	segments := lib.DecodeLengthPrefixed(key)
	if len(segments) != 3 {
		return nil, ErrInvalidKey()
	}
	height := binary.BigEndian.Uint64(segments[2])
	return &lib.Checkpoint{
		Height:    height,
		BlockHash: value,
	}, nil
}

// HELPER CODE BELOW

// getQC() gets the QC bytes from the DB and converts it into a QC object
func (t *Indexer) getQC(heightKey []byte) (*lib.QuorumCertificate, lib.ErrorI) {
	// get from db
	bz, err := t.db.Get(heightKey)
	if err != nil {
		return nil, err
	}
	if len(bz) == 0 {
		return nil, ErrStoreGet(errors.New("quorum certificate not found"))
	}
	// convert to QC object
	ptr := new(lib.QuorumCertificate)
	if err = lib.Unmarshal(bz, ptr); err != nil {
		return nil, err
	}
	return ptr, nil
}

// getBlock() gets the block bytes from the DB and converts it into a filled BlockResult object including the transactions
func (t *Indexer) getBlock(hashKey []byte, transactions bool) (*lib.BlockResult, lib.ErrorI) {
	bz, err := t.db.Get(hashKey)
	if err != nil {
		return nil, err
	}
	if len(bz) == 0 {
		return nil, ErrStoreGet(errors.New("block not found"))
	}
	ptr := new(lib.BlockHeader)
	if err = lib.Unmarshal(bz, ptr); err != nil {
		return nil, err
	}
	if !transactions {
		result := &lib.BlockResult{
			BlockHeader: ptr,
		}
		resultBz, err := lib.Marshal(result)
		if err != nil {
			return nil, err
		}
		result.Meta = &lib.BlockResultMeta{Size: uint64(len(resultBz))}
		return result, nil
	}
	txs, err := t.GetTxsByHeightNonPaginated(ptr.Height, false)
	if err != nil {
		return nil, err
	}
	events, err := t.GetEventsNonPaginated(ptr.Height, false)
	if err != nil {
		return nil, err
	}
	bz, err = lib.Marshal(&lib.BlockResult{
		BlockHeader:  ptr,
		Transactions: txs,
	})
	if err != nil {
		return nil, err
	}
	return &lib.BlockResult{
		BlockHeader:  ptr,
		Meta:         &lib.BlockResultMeta{Size: uint64(len(bz))},
		Events:       events,
		Transactions: txs,
	}, nil
}

// getTx() gets the tx bytes from the DB and converts it into TxResult object
func (t *Indexer) getTx(key []byte) (*lib.TxResult, lib.ErrorI) {
	bz, err := t.db.Get(key)
	if err != nil {
		return nil, err
	}
	if len(bz) == 0 {
		return nil, nil
	}
	ptr := new(lib.TxResult)
	if err = lib.Unmarshal(bz, ptr); err != nil {
		return nil, err
	}
	return ptr, nil
}

// getTxsNonPaginated() gets the txs in sorted order by block.index in a slice format
func (t *Indexer) getTxsNonPaginated(prefix []byte, newestToOldest bool) (results []*lib.TxResult, err lib.ErrorI) {
	var it lib.IteratorI
	switch newestToOldest {
	case true:
		it, err = t.db.RevIterator(prefix)
	case false:
		it, err = t.db.Iterator(prefix)
	}
	if err != nil {
		return nil, err
	}
	defer it.Close()
	for ; it.Valid(); it.Next() {
		tx, e := t.getTx(it.Value())
		if e != nil {
			return nil, e
		}
		results = append(results, tx)
	}
	return
}

// getTxs() returns a page of transactions in sorted order by block.index
func (t *Indexer) getTxs(prefix []byte, newestToOldest bool, p lib.PageParams) (page *lib.Page, err lib.ErrorI) {
	txResults, page := make(lib.TxResults, 0), lib.NewPage(p, lib.TxResultsPageName)
	err = page.Load(prefix, newestToOldest, &txResults, t.db, func(_, b []byte) (e lib.ErrorI) {
		tx, e := t.getTx(b)
		if e == nil {
			txResults = append(txResults, tx)
		}
		return
	})
	return
}

// deleteAll() deletes all the keys for a prefix
func (t *Indexer) deleteAll(prefix []byte) lib.ErrorI {
	it, err := t.db.Iterator(prefix)
	if err != nil {
		return err
	}
	defer it.Close()
	var keysToDelete [][]byte
	for ; it.Valid(); it.Next() {
		keysToDelete = append(keysToDelete, it.Key())
	}
	for _, k := range keysToDelete {
		if err = t.db.Delete(k); err != nil {
			return err
		}
	}
	return nil
}

func (t *Indexer) indexTxByHash(hash, bz []byte) (hashKey []byte, err lib.ErrorI) {
	k := t.txHashKey(hash)
	return k, t.db.Set(k, bz)
}

func (t *Indexer) indexTxByHeightAndIndex(heightAndIndexKey []byte, bz []byte) lib.ErrorI {
	return t.db.Set(heightAndIndexKey, bz)
}

func (t *Indexer) indexTxBySender(sender, heightAndIndexKey []byte, bz []byte) lib.ErrorI {
	return t.db.Set(t.txSenderKey(sender, heightAndIndexKey), bz)
}

func (t *Indexer) indexTxByRecipient(recipient, heightAndIndexKey []byte, bz []byte) lib.ErrorI {
	if recipient == nil {
		return nil
	}
	return t.db.Set(t.txRecipientKey(recipient, heightAndIndexKey), bz)
}

func (t *Indexer) indexQCByHeight(height uint64, bz []byte) lib.ErrorI {
	return t.db.Set(t.qcHeightKey(height), bz)
}

func (t *Indexer) indexBlockByHash(hash, bz []byte) (hashKey []byte, err lib.ErrorI) {
	k := t.blockHashKey(hash)
	return k, t.db.Set(k, bz)
}

func (t *Indexer) indexBlockByHeight(height uint64, bz []byte) lib.ErrorI {
	return t.db.Set(t.blockHeightKey(height), bz)
}

func (t *Indexer) indexDoubleSignerByHeight(address []byte, height uint64) lib.ErrorI {
	return t.db.Set(t.doubleSignerHeightKey(address, height), doubleSignerPrefix) // using the prefix byte as the 'is set' value
}

func (t *Indexer) txHashKey(hash []byte) []byte {
	return t.key(txHashPrefix, hash, nil)
}

func (t *Indexer) eventHeightKey(height uint64) []byte {
	return t.key(eventHeightPrefix, t.encodeBigEndian(height), nil)
}

func (t *Indexer) eventHeightAndIndexKey(height, index uint64) []byte {
	return t.key(eventHeightPrefix, t.encodeBigEndian(height), t.encodeBigEndian(index))
}

func (t *Indexer) indexEventByHeightAndIndex(heightAndIndexKey []byte, bz []byte) lib.ErrorI {
	return t.db.Set(heightAndIndexKey, bz)
}

func (t *Indexer) txHeightAndIndexKey(height, index uint64) []byte {
	return t.key(txHeightPrefix, t.encodeBigEndian(height), t.encodeBigEndian(index))
}

func (t *Indexer) txHeightKey(height uint64) []byte {
	return t.key(txHeightPrefix, t.encodeBigEndian(height), nil)
}

func (t *Indexer) txSenderKey(address, heightAndIndexKey []byte) []byte {
	return t.key(txSenderPrefix, address, heightAndIndexKey)
}

func (t *Indexer) txRecipientKey(address, heightAndIndexKey []byte) []byte {
	return t.key(txRecipientPrefix, address, heightAndIndexKey)
}

func (t *Indexer) blockHashKey(hash []byte) []byte {
	return t.key(blockHashPrefix, hash, nil)
}

func (t *Indexer) blockHeightKey(height uint64) []byte {
	return t.key(blockHeightPrefix, t.encodeBigEndian(height), nil)
}

func (t *Indexer) qcHeightKey(height uint64) []byte {
	return t.key(qcHeightPrefix, t.encodeBigEndian(height), nil)
}

func (t *Indexer) checkpointsCommitteeKey(committee uint64) []byte {
	return t.key(checkPointPrefix, t.encodeBigEndian(committee), nil)
}

func (t *Indexer) checkpointKey(committee, height uint64) []byte {
	return t.key(checkPointPrefix, t.encodeBigEndian(committee), t.encodeBigEndian(height))
}

func (t *Indexer) doubleSignerHeightKey(address []byte, height uint64) []byte {
	return t.key(doubleSignerPrefix, address, t.encodeBigEndian(height))
}

func (t *Indexer) key(prefix, param1, param2 []byte) []byte {
	return lib.JoinLenPrefix(prefix, param1, param2)
}

// encodeBigEndian() encodes a number such that default DB order
// (lexicographical) will sort properly when iterating by height
func (t *Indexer) encodeBigEndian(i uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, i)
	return b
}

// decodeBigEndian() decodes a number from big endian bytes
func (t *Indexer) decodeBigEndian(b []byte) uint64 {
	return binary.BigEndian.Uint64(b)
}

func (t *Indexer) setDB(db *Txn) { t.db = db }
