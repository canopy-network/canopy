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
	"sort"
	"sync"
	"time"

	"github.com/canopy-network/canopy/lib"
)

const (
	MaxFailedDialAttempts        = 5               // maximum times a peer may fail a churn management dial attempt before evicted from the peer book
	MaxPeersExchanged            = 5               // maximum number of peers per chain that may be sent/received during a peer exchange
	MaxPeerBookRequestsPerWindow = 2               // maximum peer book request per window
	PeerBookRequestWindowS       = 120             // seconds in a peer book request
	PeerBookRequestTimeoutS      = 5               // timeout in seconds of the peer book
	CrawlAndCleanBookFrequency   = time.Hour       // how often the book is cleaned and crawled
	SaveBookFrequency            = time.Minute * 5 // how often the book is saved to a file
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
		peerInfo, err := p.SendToRandPeer(lib.Topic_PEERS_REQUEST, &PeerBookRequestMessage{})
		if peerInfo == nil || err != nil {
			continue
		}
		p.log.Debugf("Sent peer book request to %s", lib.BytesToTruncatedString(peerInfo.Address.PublicKey))
		select {
		// fires when received the response to the request
		case msg := <-p.Inbox(lib.Topic_PEERS_RESPONSE):
			p.log.Debugf("Received peer book response from %s", lib.BytesToTruncatedString(msg.Sender.Address.PublicKey))
			senderID := msg.Sender.Address.PublicKey
			// ensure PeerBookResponse message type
			peerBookResponseMsg, ok := msg.Message.(*PeerBookResponseMessage)
			if !ok {
				p.log.Warnf("Invalid peer book response from %s", lib.BytesToTruncatedString(msg.Sender.Address.PublicKey))
				p.ChangeReputation(senderID, InvalidMsgRep)
				continue
			}
			// ensure it's the expected sender
			if !bytes.Equal(msg.Sender.Address.PublicKey, peerInfo.Address.PublicKey) {
				p.log.Warnf("Unexpected peer book response from %s", lib.BytesToTruncatedString(msg.Sender.Address.PublicKey))
				p.ChangeReputation(senderID, UnexpectedMsgRep)
				continue
			}
			// if they sent too many peers
			if len(peerBookResponseMsg.Book) > MaxPeersExchanged {
				p.log.Warnf("Too many peers sent from %s", lib.BytesToTruncatedString(msg.Sender.Address.PublicKey))
				p.ChangeReputation(senderID, ExceedMaxPBLenRep)
				continue
			}
			// add each peer to the book (deduplicated upon adding)
			for _, bp := range peerBookResponseMsg.Book {
				p.book.Add(bp)
			}
			p.ChangeReputation(senderID, GoodPeerBookRespRep)
			// fires when request times out
		case <-time.After(PeerBookRequestTimeoutS * time.Second):
			p.log.Warnf("Peer book timeout from %s", lib.BytesToTruncatedString(peerInfo.Address.PublicKey))
			p.ChangeReputation(peerInfo.Address.PublicKey, PeerBookReqTimeoutRep)
			continue
		}
	}
}

// ListenForPeerBookRequests()
func (p *P2P) ListenForPeerBookRequests() {
	// limit the number of inbound PeerBook requests per requester and by total number of requests
	l := lib.NewLimiter(MaxPeerBookRequestsPerWindow, p.MaxPossiblePeers()*MaxPeerBookRequestsPerWindow, PeerBookRequestWindowS)
	for {
		select {
		// fires after receiving a peer request
		case msg := <-p.Inbox(lib.Topic_PEERS_REQUEST):
			p.log.Debugf("Received peer book request from %s", lib.BytesToTruncatedString(msg.Sender.Address.PublicKey))
			requesterID := msg.Sender.Address.PublicKey
			// rate limit per requester
			blocked, totalBlock := l.NewRequest(lib.BytesToString(requesterID))
			// if requester blocked
			if blocked {
				p.ChangeReputation(requesterID, ExceedMaxPBReqRep)
				continue
			}
			// if blocked by total number of requests
			if totalBlock {
				continue // dos defensive
			}
			// only should be PeerBookMessage in this channel
			if _, ok := msg.Message.(*PeerBookRequestMessage); !ok {
				p.log.Warnf("Received invalid peer book request from %s", lib.BytesToString(msg.Sender.Address.PublicKey))
				p.ChangeReputation(requesterID, InvalidMsgRep)
				continue
			}
			var response []*BookPeer
			// grab up to MaxPeerExchangePerChain number of peers for that specific chain
			for i := 0; i < MaxPeersExchanged; i++ {
				toBeAdded := p.book.GetRandom()
				if toBeAdded == nil {
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
				p.log.Error(err.Error()) // log error
			}
		case <-l.TimeToReset(): // fires when the limiter should reset
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
		// iterate through the copy
		for _, peer := range bookCopy {
			// try to dial the peer
			if err := dialAndDisconnect(peer.Address, true); err != nil {
				// if failed, add failed attempt
				p.AddFailedDialAttempt(peer.Address.PublicKey)
			} else {
				// if succeeded, reset failed attempts
				p.ResetFailedDialAttempts(peer.Address.PublicKey)
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
	p.log.Debugf("Try add book peer %s", lib.BytesToTruncatedString(peer.Address.PublicKey))
	// if peer is self, ignore
	if bytes.Equal(p.publicKey, peer.Address.PublicKey) {
		return
	}
	// lock for thread safety
	p.l.Lock()
	defer p.l.Unlock()
	// get the index where the peer should be located in the slice
	i, found := p.getIndex(peer.Address.PublicKey)
	// if peer already exists in the slice
	if found {
		p.Book[i] = peer // overwrite existing in case ip changed
		return
	}
	// if the peer does not yet exist, add it to the slice
	p.BookSize++
	p.Book = append(p.Book, new(BookPeer))
	copy(p.Book[i+1:], p.Book[i:])
	p.Book[i] = peer
}

// Remove() a peer from the book
func (p *PeerBook) Remove(publicKey []byte) {
	p.l.Lock()
	defer p.l.Unlock()
	// get the index where the peer should be located in the slice
	i, found := p.getIndex(publicKey)
	// if not in the slice, ignore
	if !found {
		return
	}
	p.log.Debugf("Removing peer %s from PeerBook", lib.BytesToString(publicKey))
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
func (p *PeerBook) ResetFailedDialAttempts(publicKey []byte) {
	p.l.Lock()
	defer p.l.Unlock()
	// get the peer at a specific index
	i, found := p.getIndex(publicKey)
	// if not in the slice, ignore
	if !found {
		return
	}
	// set the peer consecutive failed dial attempts to zero
	p.Book[i].ConsecutiveFailedDial = 0
}

// AddFailedDialAttempt() increments the failed dial attempt counter for a BookPeer
func (p *PeerBook) AddFailedDialAttempt(publicKey []byte) {
	p.l.Lock()
	defer p.l.Unlock()
	// get the peer at a specific index
	i, found := p.getIndex(publicKey)
	// if not in the slice, ignore
	if !found {
		return
	}
	// increment the consecutive failed dial attempts for the peer
	p.Book[i].ConsecutiveFailedDial++
	// if the consecutive failed dial attempts exceeds the maximum
	// then remove the peer from the book
	if p.Book[i].ConsecutiveFailedDial >= MaxFailedDialAttempts {
		p.log.Debugf("Removing peer %s from PeerBook after max failed dial", lib.BytesToString(publicKey))
		p.delAtIndex(i)
		return
	}
}

// GetBookPeers() returns all peers in the PeerBook
func (p *P2P) GetBookPeers() []*BookPeer { return p.book.GetAll() }

// WriteToFile() saves the peer book object to a json file
func (p *PeerBook) WriteToFile() error {
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
	}
}

// getIndex() returns the index where the peer should be located within the sorted
// slice, and if the peer exists in the slice or not
func (p *PeerBook) getIndex(publicKey []byte) (int, bool) {
	// binary search the slice to find the index where the public key should be located
	i := sort.Search(p.BookSize, func(i int) bool {
		return bytes.Compare(p.Book[i].Address.PublicKey, publicKey) >= 0
	})
	// if the index is within the slice and the peer at that index has the target pubkey
	if i != p.BookSize && bytes.Equal(p.Book[i].Address.PublicKey, publicKey) {
		return i, true // found
	}
	return i, false // not found but i == index where it should be
}

// delAtIndex() deletes the peer from the slice at index i
func (p *PeerBook) delAtIndex(i int) {
	p.BookSize--
	p.Book = append(p.Book[:i], p.Book[i+1:]...)
}
