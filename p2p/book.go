package p2p

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/canopy-network/canopy/lib"
)

var (
	MaxFailedDialAttempts        = int32(1)         // maximum times a peer may fail a churn management dial attempt before evicted from the peer book
	MaxPeersExchanged            = 1                // maximum number of peers per chain that may be sent/received during a peer exchange
	MaxPeerBookRequestsPerWindow = 2                // maximum peer book request per window
	PeerBookRequestWindowS       = 30               // seconds in a peer book request
	CrawlAndCleanBookFrequency   = time.Minute * 10 // how often the book is cleaned and crawled
	SaveBookFrequency            = time.Minute * 5  // how often the book is saved to a file
)

// PeerBook is a persisted structure that maintains information on potential peers
type PeerBook struct {
	Book      []*BookPeer  `json:"book"`     // persisted list of peers
	BookSize  int          `json:"bookSize"` // number of peers in the book
	publicKey []byte       // self public key
	path      string       // path to write the peer book json file to
	l         sync.RWMutex // thread safety to update the list
	log       lib.LoggerI  // logger
}

// NewPeerBook() instantiates a PeerBook object from a file, it creates a file if none exist
func NewPeerBook(publicKey []byte, c lib.Config, l lib.LoggerI) *PeerBook {
	pb := &PeerBook{
		Book:      make([]*BookPeer, 0),
		BookSize:  0,
		publicKey: publicKey,
		l:         sync.RWMutex{},
		path:      path.Join(c.DataDirPath, "book.json"),
		log:       l,
	}
	// check if json file exist, if not create one
	if _, err := os.Stat(pb.path); errors.Is(err, os.ErrNotExist) {
		l.Infof("Creating %s file", pb.path)
		if err = pb.WriteToFile(); err != nil {
			l.Fatal(err.Error())
		}
	}
	// read the json file
	bz, err := os.ReadFile(pb.path)
	if err != nil {
		l.Fatalf("unable to read peer book: %s", err.Error())
	}
	// load the bytes into the peer book object
	if err = json.Unmarshal(bz, pb); err != nil {
		l.Fatalf("unable to unmarshal peer book: %s", err.Error())
	}
	return pb
}

// SendPeerBookRequests() is the requesting service of the peer exchange
// Sends a peer request out to a random peer and waits PeerBookRequestTimeoutS for a response
func (p *P2P) SendPeerBookRequests() {
	// sleep time is roundUp(window / request per window), round up to ensure there's never a situation where we exceed the send rate
	secondsPerReq := int(math.Ceil(float64(PeerBookRequestWindowS) / float64(MaxPeerBookRequestsPerWindow)))
	for sleepTime := 0; ; sleepTime = secondsPerReq {
		// rate limit with sleep timer
		time.Sleep(time.Duration(sleepTime) * time.Second)
		// send the request
		if err := p.SendToPeers(lib.Topic_PEERS_REQUEST, &PeerBookRequestMessage{}); err != nil {
			p.log.Errorf("Error %s happened in send peer book requests", err.Error())
			continue
		}
		p.log.Debugf("Sent peer book request to all peers")
	}
}

func (p *P2P) ListenForPeerBookResponses() {
	// limit the number of inbound PeerBook requests per requester and by total number of requests
	l := lib.NewLimiter(MaxPeerBookRequestsPerWindow, p.MaxPossiblePeers()*MaxPeerBookRequestsPerWindow, PeerBookRequestWindowS, "PEER_BOOK_RESPONSE", p.log)
	for {
		select {
		// fires when received the response to the request
		case msg := <-p.Inbox(lib.Topic_PEERS_RESPONSE):
			// p.log.Debugf("Received peer book response from %s", lib.BytesToTruncatedString(msg.Sender.Address.PublicKey))
			senderID := msg.Sender.Address.PublicKey
			// rate limit per requester
			blocked, totalBlock := l.NewRequest(lib.BytesToString(senderID))
			// if requester blocked
			if blocked {
				p.log.Warnf("too many peer book responses from %s", lib.BytesToTruncatedString(msg.Sender.Address.PublicKey))
				p.ChangeReputation(senderID, ExceedMaxPBReqRep)
				continue
			}
			// if blocked by total number of requests
			if totalBlock {
				// p.log.Warnf("blocked by total block in book responses from %s", lib.BytesToTruncatedString(msg.Sender.Address.PublicKey))
				continue // dos defensive
			}
			// ensure PeerBookResponse message type
			peerBookResponseMsg := new(PeerBookResponseMessage)
			if err := lib.Unmarshal(msg.Message, peerBookResponseMsg); err != nil {
				p.log.Warnf("Invalid peer book response from %s", lib.BytesToTruncatedString(msg.Sender.Address.PublicKey))
				p.ChangeReputation(senderID, InvalidMsgRep)
				continue
			}
			// if they sent too many peers
			if len(peerBookResponseMsg.Book) > MaxPeersExchanged {
				//p.log.Warnf("Too many peers sent from %s", lib.BytesToTruncatedString(msg.Sender.Address.PublicKey))
				//p.ChangeReputation(senderID, ExceedMaxPBLenRep)
				continue
			}
			// add each peer to the book (deduplicated upon adding)
			for _, bp := range peerBookResponseMsg.Book {
				// skip empty
				if bp == nil || bp.Address == nil || bp.Address.PeerMeta == nil {
					//p.log.Warnf("empty book response message from %s", lib.BytesToTruncatedString(msg.Sender.Address.PublicKey))
					continue
				}
				// skip max dial failed
				if bp.ConsecutiveFailedDial >= MaxFailedDialAttempts {
					//p.log.Warnf("max consecutibe failed dials from %s", lib.BytesToTruncatedString(msg.Sender.Address.PublicKey))
					continue
				}
				// skip if already connected
				if p.Has(bp.Address.PublicKey) {
					//p.log.Warnf("public key already connected from %s", lib.BytesToTruncatedString(msg.Sender.Address.PublicKey))
					continue
				}
				// try to dial, now async so we don't block processing messages'
				go func(address *lib.PeerAddress) {
					if err := p.DialAndDisconnect(address, true); err == nil {
						p.book.Add(bp)
					}
				}(bp.Address)
			}
			p.ChangeReputation(senderID, GoodPeerBookRespRep)
		case <-l.TimeToReset(): // fires when the limiter should reset
			//p.log.Info("Limiter reset in book responses")
			l.Reset()
		}
	}
}

// ListenForPeerBookRequests()
func (p *P2P) ListenForPeerBookRequests() {
	// limit the number of inbound PeerBook requests per requester and by total number of requests
	l := lib.NewLimiter(MaxPeerBookRequestsPerWindow, p.MaxPossiblePeers()*MaxPeerBookRequestsPerWindow, PeerBookRequestWindowS, "PEER_BOOK_REQUEST", p.log)
	for {
		select {
		// fires after receiving a peer request
		case msg := <-p.Inbox(lib.Topic_PEERS_REQUEST):
			// p.log.Debugf("Received peer book request from %s", lib.BytesToTruncatedString(msg.Sender.Address.PublicKey))
			requesterID := msg.Sender.Address.PublicKey
			// rate limit per requester
			blocked, totalBlock := l.NewRequest(lib.BytesToString(requesterID))
			// if requester blocked
			if blocked {
				p.log.Warnf("too many peer book requests from %s", lib.BytesToTruncatedString(msg.Sender.Address.PublicKey))
				p.ChangeReputation(requesterID, ExceedMaxPBReqRep)
				continue
			}
			// if blocked by total number of requests
			if totalBlock {
				//p.log.Warnf("blocked by total block in book requests from %s", lib.BytesToTruncatedString(msg.Sender.Address.PublicKey))
				continue // dos defensive
			}
			// only should be PeerBookMessage in this channel
			if err := lib.Unmarshal(msg.Message, new(PeerBookRequestMessage)); err != nil {
				p.log.Warnf("Received invalid peer book request from %s", lib.BytesToString(msg.Sender.Address.PublicKey))
				p.ChangeReputation(requesterID, InvalidMsgRep)
				continue
			}
			var response []*BookPeer
			// grab up to MaxPeerExchangePerChain number of peers for that specific chain
			for i := 0; i <= MaxPeersExchanged; i++ {
				toBeAdded := p.book.GetRandom()
				if toBeAdded == nil {
					//p.log.Warnf("nil to be added from %s", lib.BytesToTruncatedString(msg.Sender.Address.PublicKey))
					break
				}
				if !slices.ContainsFunc(response, func(p *BookPeer) bool { // ensure no duplicates
					return bytes.Equal(p.Address.PublicKey, toBeAdded.Address.PublicKey)
				}) {
					response = append(response, toBeAdded) // add BookPeer to response
				}
			}
			// send response to the requester
			err := p.SendTo(requesterID, lib.Topic_PEERS_RESPONSE, &PeerBookResponseMessage{Book: response})
			if err != nil {
				p.log.Errorf("Error %s in sendTo from %s", err.Error(), lib.BytesToTruncatedString(msg.Sender.Address.PublicKey)) // log error
			}
		case <-l.TimeToReset(): // fires when the limiter should reset
			//p.log.Info("Limiter reset in book requests")
			l.Reset()
		}
	}
}

// StartPeerBookService() begins:
// - Peer Exchange service: exchange known peers with currently active set
// - Churn management service: evict inactive peers from the book
// - File save service: persist the book.json file periodically
func (p *P2P) StartPeerBookService() {
	go p.ListenForPeerBookRequests()
	go p.SendPeerBookRequests()
	go p.ListenForPeerBookResponses()
	go p.book.StartChurnManagement(p.DialAndDisconnect)
	go p.book.SaveRoutine()
}

// StartChurnManagement() evicts inactive peers from the PeerBook by periodically attempting to connect with each peer
func (p *PeerBook) StartChurnManagement(dialAndDisconnect func(a *lib.PeerAddress, strictPublicKey bool) lib.ErrorI) {
	for {
		// snapshot the PeerBook
		p.l.RLock()
		bookCopy := make([]*BookPeer, len(p.Book))
		copy(bookCopy, p.Book)
		p.l.RUnlock()
		// create a map to deduplicate based on 'signature'
		pubs, netAddrs := make(map[string]int), make(map[string]int)
		// iterate through the copy
		for _, peer := range bookCopy {
			// deduplicate based on public key, net address, and ips
			pubs[lib.BytesToTruncatedString(peer.Address.PublicKey)]++
			// net addr deduplication
			netAddrs[peer.Address.NetAddress]++
			// deduplicate based on signature
			if err := dialAndDisconnect(peer.Address, true); err != nil {
				p.AddFailedDialAttempt(peer.Address)
			} else {
				// if succeeded, reset failed attempts
				p.ResetFailedDialAttempts(peer.Address)
			}
		}
		// second pass to remove duplicates
		p.l.RLock()
		bookCopy = make([]*BookPeer, len(p.Book))
		copy(bookCopy, p.Book)
		p.l.RUnlock()
		// iterate through the copy
		for _, peer := range bookCopy {
			if pubs[lib.BytesToTruncatedString(peer.Address.PublicKey)] > 1 ||
				netAddrs[peer.Address.NetAddress] > 1 {
				p.DeleteAtIndex(peer.Address)
			}
		}
		time.Sleep(CrawlAndCleanBookFrequency)
	}
}

// GetRandom() returns a random peer from the Book that has a specific chain in the Meta
func (p *PeerBook) GetRandom() *BookPeer {
	p.l.RLock()
	defer p.l.RUnlock()
	peeringCandidates, numCandidates := p.getPeers()
	if numCandidates == 0 {
		return nil
	}
	return peeringCandidates[rand.Intn(numCandidates)]
}

// getPeers() returns peers from the peer book
func (p *PeerBook) getPeers() (peers []*BookPeer, count int) {
	for _, peer := range p.Book {
		count++
		peers = append(peers, peer)
	}
	return
}

// GetAll() returns a snapshot of all peers in the book
func (p *PeerBook) GetAll() (res []*BookPeer) {
	p.l.RLock()
	defer p.l.RUnlock()
	res = append(res, p.Book...)
	return
}

// Add() adds a peer to the book in sorted order by public key
func (p *PeerBook) Add(peer *BookPeer) {
	//p.log.Debugf("Try add book peer %s with self %s", lib.BytesToTruncatedString(peer.Address.PublicKey), lib.BytesToTruncatedString(p.publicKey))
	// if peer is self, ignore
	if bytes.Equal(p.publicKey, peer.Address.PublicKey) {
		//p.log.Debugf("Peer %s is self; ignoring", lib.BytesToTruncatedString(peer.Address.PublicKey))
		return
	}
	// lock for thread safety
	p.l.Lock()
	defer p.l.Unlock()
	// get the index where the peer should be located in the slice
	i, found := p.getIndex(peer.Address)
	// if peer already exists in the slice
	if found {
		p.Book[i] = peer // overwrite existing in case ip changed
		//p.log.Debugf("Peer %s already found", lib.BytesToTruncatedString(peer.Address.PublicKey))
		return
	}
	// if the peer does not yet exist, add it to the slice
	p.BookSize++
	p.Book = append(p.Book, new(BookPeer))
	copy(p.Book[i+1:], p.Book[i:])
	p.Book[i] = peer
	p.log.Debugf("Added book peer %s", lib.BytesToTruncatedString(peer.Address.PublicKey))
}

// Remove() a peer from the book
func (p *PeerBook) Remove(address *lib.PeerAddress) {
	p.l.Lock()
	defer p.l.Unlock()
	// get the index where the peer should be located in the slice
	i, found := p.getIndex(address)
	// if not in the slice, ignore
	if !found {
		//p.log.Debugf("Peer %s from PeerBook not found trying to remove", lib.BytesToString(address.PublicKey))
		return
	}
	p.log.Debugf("Removing peer %s from PeerBook", lib.BytesToString(address.PublicKey))
	// remove at this index
	p.delAtIndex(i)
}

// GetBookSize() returns the book peer count
func (p *PeerBook) GetBookSize() int {
	p.l.RLock()
	defer p.l.RUnlock()
	_, count := p.getPeers()
	return count
}

// ResetFailedDialAttempts() resets the failed dial attempt count for the peer
func (p *PeerBook) ResetFailedDialAttempts(address *lib.PeerAddress) {
	p.l.Lock()
	defer p.l.Unlock()
	// get the peer at a specific index
	i, found := p.getIndex(address)
	// if not in the slice, ignore
	if !found {
		//p.log.Debugf("Address %s not found in reset failed dial attempts", address)
		return
	}
	// set the peer consecutive failed dial attempts to zero
	p.Book[i].ConsecutiveFailedDial = 0
}

// AddFailedDialAttempt() increments the failed dial attempt counter for a BookPeer
func (p *PeerBook) AddFailedDialAttempt(address *lib.PeerAddress) {
	p.l.Lock()
	defer p.l.Unlock()
	// get the peer at a specific index
	i, found := p.getIndex(address)
	// if not in the slice, ignore
	if !found {
		p.log.Warnf("AddFailedDialAttempt: address not found in book")
		return
	}
	// increment the consecutive failed dial attempts for the peer
	p.Book[i].ConsecutiveFailedDial++
	// if the consecutive failed dial attempts exceeds the maximum
	// then remove the peer from the book
	if p.Book[i].ConsecutiveFailedDial >= MaxFailedDialAttempts {
		p.log.Debugf("Removing peer %s from PeerBook after max failed dial", lib.BytesToString(address.PublicKey))
		p.delAtIndex(i)
		return
	}
}

// AddFailedDialAttempt() increments the failed dial attempt counter for a BookPeer
func (p *PeerBook) DeleteAtIndex(address *lib.PeerAddress) {
	p.l.Lock()
	defer p.l.Unlock()
	// get the peer at a specific index
	i, found := p.getIndex(address)
	// if not in the slice, ignore
	if !found {
		p.log.Warnf("DeleteAtIndex: address not found in book")
		return
	}
	// delete the peer at index
	p.delAtIndex(i)
}

// GetBookPeers() returns all peers in the PeerBook
func (p *P2P) GetBookPeers() []*BookPeer { return p.book.GetAll() }

// WriteToFile() saves the peer book object to a json file
func (p *PeerBook) WriteToFile() error {
	p.l.Lock()
	defer p.l.Unlock()
	configBz, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	// create all necessary directories
	if err = os.MkdirAll(filepath.Dir(p.path), os.ModePerm); err != nil {
		return err
	}
	return os.WriteFile(p.path, configBz, os.ModePerm)
}

// SaveRoutine() periodically saves the book to a json file
func (p *PeerBook) SaveRoutine() {
	for {
		time.Sleep(SaveBookFrequency)
		if err := p.WriteToFile(); err != nil {
			p.log.Error(err.Error())
		}
		p.log.Debug("Peer book saved in file")
	}
}

// getIndex() returns the index where the peer should be located within the sorted
// slice, and if the peer exists in the slice or not
func (p *PeerBook) getIndex(address *lib.PeerAddress) (int, bool) {
	// loop through the peer book
	for i, peer := range p.Book {
		// if the addresses are equal
		if peer.Address.Equals(address) {
			// return the index and found
			return i, true
		}
	}
	// return not found
	return 0, false
}

// delAtIndex() deletes the peer from the slice at index i
func (p *PeerBook) delAtIndex(i int) {
	p.BookSize--
	p.Book = append(p.Book[:i], p.Book[i+1:]...)
}
