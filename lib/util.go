package lib

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/canopy-network/canopy/lib/crypto"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	MaxAllowedPort = 65535 // maxAllowedPort is the maximum port number allowed.
	MinAllowedPort = 1025  // minAllowedPort is the minimum port number allowed to ensure it avoids commonly reserved system ports.

	protoMaxListLen      = 100000
	protoMaxMapLen       = 100000
	protoMaxRecursion    = 32
	protoMaxFieldBytes   = 32 * 1024 * 1024 // 32MB per length-delimited field
	protoMaxMessageBytes = 64 * 1024 * 1024 // 64MB overall per message
)

// PAGE CODE BELOW

// RegisteredPageables is a global slice of registered pageables for generic unmarshalling
var RegisteredPageables = make(map[string]Pageable)

func init() {
	RegisteredPageables[TxResultsPageName] = new(TxResults)      // preregister the page type for unmarshalling
	RegisteredPageables[PendingResultsPageName] = new(TxResults) // preregister the page type for unmarshalling
	RegisteredPageables[FailedTxsPageName] = new(FailedTxs)      // preregister the page type for unmarshalling
	RegisteredPageables[EventsPageName] = new(Events)            // preregister the page type for unmarshalling
}

// Page is a pagination wrapper over a slice of data
type Page struct {
	PageParams          // the input parameters for the page
	Results    Pageable `json:"results"`    // the actual returned array of items
	Type       string   `json:"type"`       // the type of the page
	Count      int      `json:"count"`      // count of items included in the page
	TotalPages int      `json:"totalPages"` // number of pages that exist based on these page parameters
	TotalCount int      `json:"totalCount"` // count of items that exist
}

// PageParams are the input parameters to calculate the proper page
type PageParams struct {
	PageNumber int `json:"pageNumber"`
	PerPage    int `json:"perPage"`
}

// Pageable() is a simple interface that represents Page structures
type Pageable interface{ New() Pageable }

// NewPage() returns a new instance of the Page object from the params and pageType
// Load() or LoadArray() is the likely next function call
func NewPage(p PageParams, pageType string) *Page { return &Page{PageParams: p, Type: pageType} }

// Load() fills a page from an IteratorI
func (p *Page) Load(storePrefix []byte, reverse bool, results Pageable, db RStoreI, callback func(k, v []byte) ErrorI) (err ErrorI) {
	// create a new iterator object to hold the store iterator
	var it IteratorI
	// set the page results so that even if it's a zero page, it will have a castable type
	p.Results = results
	// prefix keys with numbers in big endian ensure that reverse iteration
	// is highest to lowest and vise versa
	switch reverse {
	case true:
		// use a reverse iterator
		it, err = db.RevIterator(storePrefix)
	case false:
		// use a normal iterator
		it, err = db.Iterator(storePrefix)
	}
	// if an error occurred during iterator construction
	if err != nil {
		// exit with error
		return
	}
	// close the iterator once function completes for memory recovery
	defer it.Close()
	// skip to index makes the starting point appropriate based on the page params
	// initialize variable to indicate if the loop is counting only or actually populating
	pageStartIndex, countOnly := p.skipToIndex(), false
	// execute the loop
	for ; it.Valid(); it.Next() {
		// pre-increment total count to ensure each iteration of the loop is counted including if !it.Valid() or `countOnly`
		p.TotalCount++
		// while count is below the start page index (LTE because we pre-increment)
		if p.TotalCount <= pageStartIndex || countOnly {
			// TODO investigate how to optimize skips (turn pre-fetching off etc.)
			continue
		}
		// if reached end of the desired page (+1 because we pre-increment)
		if p.TotalCount == pageStartIndex+p.PerPage+1 {
			// switch to only counts
			countOnly = true
			// continue the next
			continue
		}
		// execute the callback; passing key and value
		if e := callback(it.Key(), it.Value()); e != nil {
			return e
		}
		// set the results and increment the count
		p.Results = results
		// increment the count
		p.Count++
	}
	// calculate total pages
	p.TotalPages = int(math.Ceil(float64(p.TotalCount) / float64(p.PerPage)))
	// exit
	return
}

// LoadArray() fills a page from a slice
func (p *Page) LoadArray(slice any, results Pageable, callback func(item any) ErrorI) (err ErrorI) {
	// if the slice is not type of reflect
	arr := reflect.ValueOf(slice)
	// if the type is not a slice
	if arr.Kind() != reflect.Slice {
		// exit with invalid argument
		return ErrInvalidArgument()
	}
	// skip to index makes the starting point appropriate based on the page params
	pageStartIndex, size := p.skipToIndex(), arr.Len()
	// initialize variable to indicate if the loop is counting only or actually populating
	countOnly := false
	// for each element in the slice
	for p.TotalCount < size {
		// pre-increment total count to ensure each iteration of the loop is counted including if p.TotalCount > size or `countOnly`
		p.TotalCount++
		// while count is below the start page index (LTE because we pre-increment)
		if p.TotalCount <= pageStartIndex || countOnly {
			// go to next iteration
			continue
		}
		// convert the element at the index to an 'any'
		a := arr.Index(p.TotalCount - 1).Interface()
		// if reached end of the desired page (+1 because we pre-increment)
		if p.TotalCount-1 == pageStartIndex+p.PerPage {
			// switch to only counts
			countOnly = true
			// continue with next iteration
			continue
		}
		// pass the 'any' to the callback
		if err = callback(a); err != nil {
			// exit with error
			return
		}
		// set the results and increment the count
		p.Results = results
		// increment the count
		p.Count++
	}
	// calculate total pages
	p.TotalPages = int(math.Ceil(float64(p.TotalCount) / float64(p.PerPage)))
	// exit
	return
}

// skipToIndex() sanity checks params and then determines the first index of the page
func (p *PageParams) skipToIndex() int {
	// set the defaults
	defaultPerPage, maxPerPage := 10, 5000
	// if the perPage isn't set
	if p.PerPage == 0 {
		// use the default
		p.PerPage = defaultPerPage
	}
	// if the per page exceeds the max per page
	if p.PerPage > maxPerPage {
		// if the perPage exceeds the max, use the max
		p.PerPage = maxPerPage
	}
	// start page count at 1 not 0
	if p.PageNumber == 0 {
		// set to page 1
		p.PageNumber = 1
	}
	// if on the first page
	if p.PageNumber == 1 {
		// return 0 index
		return 0
	}
	// calculate the previous page number
	lastPage := p.PageNumber - 1
	// set the start to the index after the last page
	return lastPage * p.PerPage
}

// UnmarshalJSON() overrides the unmarshalling logic of the
// Page for generic structure assignment (registered pageables) and custom formatting
func (p *Page) UnmarshalJSON(jsonBytes []byte) (err error) {
	// create a new json object reference to ensure a non-nil result
	j := new(jsonPage)
	// populate the json page with json bytes
	if err = json.Unmarshal(jsonBytes, &j); err != nil {
		// exit with error
		return
	}
	// extract the pageable implementation from the previously registered pageable type
	m, found := RegisteredPageables[j.Type]
	// if not found among the registered
	if !found {
		return ErrUnknownPageable(j.Type)
	}
	// create a new instance of the page
	pageable := m.New()
	// populate the results with json bytes
	if err = json.Unmarshal(j.Results, pageable); err != nil {
		// exit with error
		return
	}
	//
	*p = Page{
		PageParams: j.PageParams,
		Results:    pageable,
		Type:       j.Type,
		Count:      j.Count,
		TotalPages: j.TotalPages,
		TotalCount: j.TotalCount,
	}
	// exit
	return
}

// jsonPage is the internal structure for custom json for the Page structure
type jsonPage struct {
	PageParams
	Results    json.RawMessage `json:"results"`
	Type       string          `json:"type"`
	Count      int             `json:"count"`
	TotalPages int             `json:"totalPages"`
	TotalCount int             `json:"totalCount"`
}

var marshaller = proto.MarshalOptions{Deterministic: true}

// Marshal() serializes a proto.Message into a byte slice
func Marshal(message any) ([]byte, ErrorI) {
	// convert the message into proto bytes using the proto marshaller
	protoBytes, err := marshaller.Marshal(message.(proto.Message))
	// if an error occurred during the conversion process
	if err != nil {
		// exit with a wrapped error
		return nil, ErrMarshal(err)
	}
	// exit
	return protoBytes, nil
}

// Unmarshal() deserializes a byte slice into a proto.Message
func Unmarshal(protoBytes []byte, ptr any) ErrorI {
	// if protoBytes are empty or ptr is nil
	if protoBytes == nil || ptr == nil {
		// return with no error
		return nil
	}
	msg, ok := ptr.(proto.Message)
	if !ok || reflect.ValueOf(ptr).Kind() != reflect.Ptr {
		return ErrUnmarshal(fmt.Errorf("target must be a pointer to a proto.Message"))
	}
	// defensive global proto size check
	if len(protoBytes) > protoMaxMessageBytes {
		return ErrUnmarshal(fmt.Errorf("proto message exceeds max size: %d > %d", len(protoBytes), protoMaxMessageBytes))
	}
	// detect if the message is a 'critical' message
	var isCritical bool
	switch ptr.(type) {
	case *Block, *Transaction, *QuorumCertificate:
		isCritical = true
		if err := preflightProtoBytes(protoBytes); err != nil {
			return ErrUnmarshal(err)
		}
	default:
	}
	// populate the ptr with the proto bytes
	opts := proto.UnmarshalOptions{AllowPartial: false}
	if err := opts.Unmarshal(protoBytes, msg); err != nil {
		// exit with wrapped error
		return ErrUnmarshal(err)
	}
	// reject unknown fields for critical messages
	if isCritical {
		if err := rejectUnknownForCriticalMessages(msg); err != nil {
			return ErrUnmarshal(err)
		}
	}
	// exit
	return nil
}

// rejectUnknownForCriticalMessages() prevents bloat attacks by enforcing 'no unknown fields' for critical messages
func rejectUnknownForCriticalMessages(msg proto.Message) error {
	switch msg.(type) {
	case *QuorumCertificate, *Block, *Transaction:
		return detectUnknownProtoFields(msg.ProtoReflect(), 0)
	default:
		return nil
	}
}

// detectUnknownProtoFields() checks for unknown protobuf messages, caps list/map sizes, and enforces recursion limits
func detectUnknownProtoFields(m protoreflect.Message, depth int) error {
	if depth > protoMaxRecursion {
		return fmt.Errorf("protobuf recursion depth exceeded (%d)", protoMaxRecursion)
	}
	if len(m.GetUnknown()) > 0 {
		return fmt.Errorf("unknown protobuf fields present in %s", m.Descriptor().FullName())
	}
	var walkErr error
	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		kind := fd.Kind()
		if fd.IsList() {
			if v.List().Len() > protoMaxListLen {
				walkErr = fmt.Errorf("protobuf list %s length %d exceeds max %d", fd.FullName(), v.List().Len(), protoMaxListLen)
				return false
			}
		}
		if fd.IsMap() {
			if v.Map().Len() > protoMaxMapLen {
				walkErr = fmt.Errorf("protobuf map %s length %d exceeds max %d", fd.FullName(), v.Map().Len(), protoMaxMapLen)
				return false
			}
		}
		if kind == protoreflect.MessageKind || kind == protoreflect.GroupKind {
			switch {
			case fd.IsList():
				list := v.List()
				for i := 0; i < list.Len(); i++ {
					if err := detectUnknownProtoFields(list.Get(i).Message(), depth+1); err != nil {
						walkErr = err
						return false
					}
				}
			case fd.IsMap():
				if fd.MapValue().Kind() == protoreflect.MessageKind {
					v.Map().Range(func(_ protoreflect.MapKey, mv protoreflect.Value) bool {
						if err := detectUnknownProtoFields(mv.Message(), depth+1); err != nil {
							walkErr = err
							return false
						}
						return true
					})
					if walkErr != nil {
						return false
					}
				}
			default:
				if err := detectUnknownProtoFields(v.Message(), depth+1); err != nil {
					walkErr = err
					return false
				}
			}
		}
		return true
	})
	return walkErr
}

// preflightProtoBytes performs a lightweight scan for oversized length-delimited fields before decoding.
func preflightProtoBytes(b []byte) error {
	offset := 0
	for offset < len(b) {
		_, wireType, n := protowire.ConsumeTag(b[offset:])
		if n < 0 {
			return fmt.Errorf("invalid protobuf tag at offset %d", offset)
		}
		offset += n

		switch wireType {
		case protowire.VarintType:
			_, n = protowire.ConsumeVarint(b[offset:])
			if n < 0 {
				return fmt.Errorf("invalid varint at offset %d", offset)
			}
			offset += n
		case protowire.Fixed32Type:
			if offset+4 > len(b) {
				return fmt.Errorf("truncated fixed32 field")
			}
			offset += 4
		case protowire.Fixed64Type:
			if offset+8 > len(b) {
				return fmt.Errorf("truncated fixed64 field")
			}
			offset += 8
		case protowire.BytesType:
			l, n := protowire.ConsumeVarint(b[offset:])
			if n < 0 {
				return fmt.Errorf("invalid length-delimited size at offset %d", offset)
			}
			offset += n
			if l > protoMaxFieldBytes {
				return fmt.Errorf("length-delimited field exceeds max size: %d > %d", l, protoMaxFieldBytes)
			}
			if l < 0 || offset+int(l) > len(b) {
				return fmt.Errorf("length-delimited field exceeds buffer bounds")
			}
			offset += int(l)
		default:
			return fmt.Errorf("unsupported wire type %d", wireType)
		}
	}
	return nil
}

// MarshalJSON() serializes a message into a JSON byte slice
func MarshalJSON(message any) ([]byte, ErrorI) {
	// convert the message to json bytes
	jsonBytes, err := json.Marshal(message)
	// if an error occurred during the conversion
	if err != nil {
		// exit with wrapped error
		return nil, ErrJSONMarshal(err)
	}
	// exit with json bytes
	return jsonBytes, nil
}

// MustMarshalJSON() serializes a message into a JSON without error checking
func MustMarshalJSON(message any) []byte {
	// convert the message to json bytes
	jsonBytes, _ := json.Marshal(message)
	// exit with json bytes
	return jsonBytes
}

// UnmarshalJSON() deserializes a JSON byte slice into the specified object
func UnmarshalJSON(jsonBytes []byte, ptr any) ErrorI {
	// populate the pointer with json bytes
	if err := json.Unmarshal(jsonBytes, ptr); err != nil {
		// exit with error
		return ErrJSONUnmarshal(err)
	}
	// exit
	return nil
}

// MarshalJSONIndent() serializes a message into an indented JSON byte slice
func MarshalJSONIndent(message any) ([]byte, ErrorI) {
	// convert the message to pretty json bytes
	bz, err := json.MarshalIndent(message, "", "  ")
	// if an error occurred during the conversion
	if err != nil {
		// exit with wrapped error
		return nil, ErrJSONMarshal(err)
	}
	// exit with pretty json bytes
	return bz, nil
}

// MarshalJSONIndentString() serializes a message into an indented JSON string
func MarshalJSONIndentString(message any) (string, ErrorI) {
	// convert the message to pretty json bytes
	bz, err := MarshalJSONIndent(message)
	// convert to string and exit
	return string(bz), err
}

// NewJSONFromFile() reads a json file into an object
func NewJSONFromFile(o any, dataDirPath, filePath string) ErrorI {
	// read the json file into bytes
	jsonFileBytes, err := os.ReadFile(filepath.Join(dataDirPath, filePath))
	// if an error occurred during the read
	if err != nil {
		// exit with error
		return ErrReadFile(err)
	}
	// populate the object using the json file bytes
	return UnmarshalJSON(jsonFileBytes, &o)
}

// SaveJSONToFile() saves a json object to a file
func SaveJSONToFile(j any, dataDirPath, filePath string) (err ErrorI) {
	// convert the object into json bytes
	jsonBytes, err := MarshalJSONIndent(j)
	// if an error occurred during the conversion
	if err != nil {
		// exit with error
		return
	}
	// attempt to write the json bytes to a json file at the path
	if e := os.WriteFile(filepath.Join(dataDirPath, filePath), jsonBytes, os.ModePerm); e != nil {
		// exit with error
		return ErrWriteFile(e)
	}
	// exit
	return
}

// NewAny() converts a proto.Message into an anypb.Any type
func NewAny(message proto.Message) (*anypb.Any, ErrorI) {
	// convert the message to a proto any
	a, err := anypb.New(message)
	// if an error occurred during the conversion
	if err != nil {
		// exit with error
		return nil, ErrToAny(err)
	}
	// exit with any
	return a, nil
}

// FromAny() converts an anypb.Any type back into a proto.Message
func FromAny(any *anypb.Any) (proto.Message, ErrorI) {
	// convert the proto any into a proto message
	msg, err := anypb.UnmarshalNew(any, proto.UnmarshalOptions{})
	// if an error occurred during the conversion
	if err != nil {
		// exit with error
		return nil, ErrFromAny(err)
	}
	// exit with the proto message
	return msg, nil
}

// BytesToString() converts a byte slice to a hexadecimal string
func BytesToString(b []byte) string {
	// hex encode the bytes into a string
	return hex.EncodeToString(b)
}

// StringToBytes() converts a hexadecimal string back into a byte slice
func StringToBytes(s string) ([]byte, ErrorI) {
	// decode the hex string into bytes
	b, err := hex.DecodeString(s)
	// if an error occurred during the decode
	if err != nil {
		// exit with error
		return nil, ErrStringToBytes(err)
	}
	// exit with bytes
	return b, nil
}

// BytesToTruncatedString() converts a byte slice to a truncated hexadecimal string
func BytesToTruncatedString(b []byte) string {
	// if the bytes are LTE the truncation
	if len(b) <= 10 {
		// simply return the string version
		return hex.EncodeToString(b)
	}
	// return the truncated string version
	return hex.EncodeToString(b[:10])
}

// PublicKeyFromBytes() converts a byte slice into a BLS public key
func PublicKeyFromBytes(pubKey []byte) (crypto.PublicKeyI, ErrorI) {
	// convert the public key bytes into a public key object
	publicKey, err := crypto.NewPublicKeyFromBytes(pubKey)
	// if an error occurred during the conversion
	if err != nil {
		// exit with error
		return nil, ErrPubKeyFromBytes(err)
	}
	// exit with the public key object
	return publicKey, nil
}

// MerkleTree() generates a Merkle tree and its root from a list of items
func MerkleTree(items [][]byte) (root []byte, tree [][]byte, err ErrorI) {
	// convert the items into a merkle tree
	root, tree, er := crypto.MerkleTree(items)
	// if an error occurred during the conversion
	if er != nil {
		// exit with error
		return nil, nil, ErrMerkleTree(er)
	}
	// exit
	return
}

// BigLess() compares two big.Int values and returns true if the first is less
func BigLess(a *big.Int, b *big.Int) bool { return a.Cmp(b) == -1 }

// Uint64PercentageDiv() calculates the percentage from dividend/divisor
func Uint64PercentageDiv(dividend, divisor uint64) (percent uint64) {
	// if either the dividend or the divisor are 0
	if dividend == 0 || divisor == 0 {
		// exit with 0
		return 0
	}
	// calculate the percent
	percent = (dividend * 100) / divisor
	// ensure the percent can't exceed 100
	if percent > 100 {
		// cap the percent at 100
		percent = 100
	}
	// exit
	return percent
}

// Uint64Percentage() calculates the percentage of an amount
func Uint64Percentage(total uint64, percentage uint64) (res uint64) {
	// if either the total or the percentage is 0
	if percentage == 0 || total == 0 {
		// exit with 0
		return 0
	}
	// if the percent is GTE 100%
	if percentage >= 100 {
		// exit with the full total
		return total
	}
	// exit with a fraction of the total
	return (total * percentage) / 100
}

// Uint64ReducePercentage() reduces an amount by a specified percentage
func Uint64ReducePercentage(fullAmount, percentage uint64) (res uint64) {
	// if the percent exceeds 100 or full amount is 0
	if percentage >= 100 || fullAmount == 0 {
		// exit with 0
		return 0
	}
	// if the percent is 0
	if percentage == 0 {
		// exit with the full amount
		return fullAmount
	}
	// exit with a reduced amount
	return (fullAmount * (100 - percentage)) / 100
}

// Uint64ToBigFloat() converts a uint64 to a big.Float
func Uint64ToBigFloat(u uint64) *big.Float {
	return new(big.Float).SetUint64(u)
}

// HexBytes represents a byte slice that can be marshaled and unmarshalled as hex strings
type HexBytes []byte

// NewHexBytesFromString() converts a hexadecimal string into HexBytes
func NewHexBytesFromString(s string) (HexBytes, ErrorI) {
	// convert the hex string into bytes
	bz, err := hex.DecodeString(s)
	// if an error occurred during the conversion
	if err != nil {
		// exit with error
		return nil, ErrJSONUnmarshal(err)
	}
	// exit with hex bytes
	return bz, nil
}

// String() returns the HexBytes as a hexadecimal string
func (x HexBytes) String() string {
	return BytesToString(x)
}

// MarshalJSON() serializes the HexBytes to a JSON byte slice
func (x HexBytes) MarshalJSON() ([]byte, error) {
	return json.Marshal(BytesToString(x))
}

// UnmarshalJSON() deserializes a JSON byte slice into HexBytes
func (x *HexBytes) UnmarshalJSON(jsonBytes []byte) (err error) {
	// create a new object ref to ensure a non nil result
	s := new(string)
	// populate the string object ref with the json bytes
	if err = json.Unmarshal(jsonBytes, s); err != nil {
		// exit with error
		return
	}
	// populate the underlying object by converting the hex string to hex bytes
	*x, err = StringToBytes(*s)
	// exit
	return
}

// ValidNetURLInput() validates the input netURL via regex
// Allow:
// - optional tcp:// prefix
// - valid hostname
// - valid ip4 and ip6 address
// - optional port (e.g., :80)
//
// Disallow:
// - Sub-paths (e.g., /path)
func ValidNetURLInput(netURL string) bool {
	// updated regex with optional port
	regex := `^(?:tcp:\/\/)?(?:localhost|(?:[a-zA-Z0-9-]+\.)*[a-zA-Z0-9-]+|(?:\d{1,3}\.){3}\d{1,3}|\[[0-9a-fA-F:]+\])(?:\:\d{1,5})?$`
	matched, err := regexp.MatchString(regex, netURL)
	if err != nil {
		return false
	}
	return matched
}

// AddToPort() adds some number to the port ensuring it doesn't exceed the max port
func AddToPort(portStr string, add uint64) (string, ErrorI) {
	// remove the colon if present
	portStr = strings.ReplaceAll(portStr, ":", "")
	// convert the port to
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return "", ErrBadPort()
	}
	// check if port is valid, should be higher than 1024 to avoid port conflicts
	if port < MinAllowedPort {
		return "", ErrBadPortLowLimit()
	}
	// add the given number to the port
	newPort := port + int(add)
	// ensure the new port doesn't exceed the max port number (65535)
	if newPort > MaxAllowedPort {
		return "", ErrMaxPort()
	}
	return fmt.Sprintf("%d", newPort), nil
}

// NewTimer() creates a 0 value initialized instance of a timer
func NewTimer() *time.Timer {
	// create a new timer with duration set to 0
	t := time.NewTimer(0)
	// discard the initial fire
	<-t.C
	// return the ready timer
	return t
}

// ResetTimer() stops the existing timer, and resets with the new duration
func ResetTimer(t *time.Timer, d time.Duration) {
	// stop the timer
	StopTimer(t)
	// reset with a new duration
	t.Reset(d)
}

// StopTimer() stops the existing timer
func StopTimer(t *time.Timer) {
	// if the timer isn't empty
	if t != nil {
		// stop the timer and check if 'already fired'
		if !t.Stop() {
			// if already fired
			select {
			// discard the trigger
			case <-t.C:
			// non blocking
			default:
			}
		}
	}
}

// CatchPanic() catches any panic in the function call or child function calls
func CatchPanic(l LoggerI) {
	if r := recover(); r != nil {
		l.Errorf(string(debug.Stack()))
	}
}

// JoinLenPrefix() appends the items together separated by a single byte to represent the length of the segment
func JoinLenPrefix(toAppend ...[]byte) []byte {
	// calculate total length first
	totalLen := 0
	for _, item := range toAppend {
		// if the item isn't empty, calculate the size
		if item != nil {
			// 1 byte for length + item length
			totalLen += 1 + len(item)
		}
	}
	// make the proper size buffer
	res := make([]byte, 0, totalLen)
	// iterate through each 'segment' and append it
	for _, item := range toAppend {
		// if item is empty, skip
		if item == nil {
			continue
		}
		// length
		res = append(res, byte(len(item)))
		// item
		res = append(res, item...)
	}
	// return the result
	return res
}

// DecodeLengthPrefixed() decodes a key that is delimited by the length of the segment in a single byte
func DecodeLengthPrefixed(key []byte) (segments [][]byte) {
	// create a variable to be 're-used' to track the length part of each prefix
	var length int
	// until the end of 'key'
	for i := 0; i < len(key); i += length {
		// read the length prefix
		length = int(key[i])
		// increment the index
		i++
		// do a sanity check on the key
		if i+length > len(key) {
			panic("corrupt or incomplete key")
		}
		// add this portion to the segments list
		segments = append(segments, key[i:i+length])
	}
	// exit
	return
}

// Retry is a simple exponential backoff retry structure in the form of doubling the timeout
type Retry struct {
	waitTimeMS uint64 // time to wait in milliseconds
	maxLoops   uint64 // the maximum number of loops before quitting
	loopCount  uint64 // the loop count itself
}

// NewRetry() constructs a new Retry given parameters
func NewRetry(waitTimeMS, maxLoops uint64) *Retry {
	return &Retry{
		waitTimeMS: waitTimeMS,
		maxLoops:   maxLoops,
	}
}

// WaitAndDoRetry() sleeps the appropriate time and returns false if maxed out retry
func (r *Retry) WaitAndDoRetry() bool {
	// if GTE max loops
	if r.maxLoops < r.loopCount {
		// exit with 'try again'
		return false
	}
	// don't sleep or increment on the first iteration
	if r.loopCount != 0 {
		// sleep the allotted time
		time.Sleep(time.Duration(r.waitTimeMS) * time.Millisecond)
		// double the timeout
		r.waitTimeMS += r.waitTimeMS
	}
	// increment the loop count
	r.loopCount++
	// exit with 'try again'
	return true
}

// TruncateSlice() safely ensures that a slice doesn't exceed the max size
func TruncateSlice[T any](slice []T, max int) []T {
	// if the slice is empty
	if slice == nil {
		// exit
		return nil
	}
	// if the slice is below the max
	if len(slice) <= max {
		// exit with the whole slice
		return slice
	}
	// exit with the truncated slice
	return slice[:max]
}

// DeDuplicator is a generic structure that serves as a simple anti-duplication check
type DeDuplicator[T comparable] struct {
	m map[T]struct{}
}

// NewDeDuplicator constructs a new object reference to a DeDuplicator
func NewDeDuplicator[T comparable]() *DeDuplicator[T] {
	return &DeDuplicator[T]{m: make(map[T]struct{})}
}

// Found() checks for an existing entry and adds it to the map if it's not present
func (d *DeDuplicator[T]) Found(k T) bool {
	// check if the key already exists
	if _, exists := d.m[k]; exists {
		// exit with 'it is a duplicate'
		return true
	}
	// add the key to the map
	d.m[k] = struct{}{}
	// exit with 'not a duplicate'
	return false
}

// Delete() removes the key from the de-duplicator map
func (d *DeDuplicator[T]) Delete(k T) { delete(d.m, k) }

// Map() returns the underlying map to the de-duplicator
func (d *DeDuplicator[T]) Map() map[T]struct{} { return d.m }

// TimeTrack() a utility function to benchmark the time of caller function
func TimeTrack(l LoggerI, start time.Time, logOnMax time.Duration) {
	elapsed, functionName := time.Since(start), "unknown"
	if elapsed < logOnMax {
		return
	}
	pcs := make([]uintptr, 10)
	n := runtime.Callers(2, pcs)
	for _, pc := range pcs[:n] {
		fn := runtime.FuncForPC(pc)
		// skip anon functions
		if fn != nil && !strings.Contains(fn.Name(), ".func") {
			fullName := fn.Name()
			parts := strings.Split(fullName, ".")
			functionName = parts[len(parts)-1]
			break
		}
	}
	l.Errorf("*** %s took %s", functionName, elapsed)
}

func PrintStackTrace(print bool) (fns []string) {
	pc := make([]uintptr, 10) // Get at most 10 stack frames
	n := runtime.Callers(2, pc)
	frames := runtime.CallersFrames(pc[:n])
	if print {
		fmt.Println("Stack trace:")
	}
	for {
		frame, more := frames.Next()
		if print {
			fmt.Printf("%s\n\t%s:%d\n", frame.Function, frame.File, frame.Line)
		}
		fns = append(fns, frame.Function)
		if !more {
			break
		}
	}
	return fns
}

// Append() is a 'safe append' when the caller wants to re-use the 'a' slice
func Append(a, b []byte) []byte {
	out := make([]byte, len(a)+len(b))
	copy(out, a)
	copy(out[len(a):], b)
	return out
}

// AppendWithBuffer() appends a and b into a fresh []byte using a buffer to reduce allocations.
// The result is safe to retain and use independently of a/b/buffer.
func AppendWithBuffer(buf *[]byte, a, b []byte) []byte {
	totalLen := len(a) + len(b)
	if cap(*buf) < totalLen {
		*buf = make([]byte, totalLen)
	}
	*buf = (*buf)[:totalLen]
	copy(*buf, a)
	copy((*buf)[len(a):], b)
	return *buf
}

// EqualByteSlices() performs equality check on two byte slices
func EqualByteSlices(a, b [][]byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !bytes.Equal(a[i], b[i]) {
			return false
		}
	}
	return true
}

func SqrtProductUint64(x, y uint64) uint64 {
	// p = x * y (big.Int)
	a := new(big.Int).SetUint64(x)
	b := new(big.Int).SetUint64(y)
	p := new(big.Int).Mul(a, b)

	return new(big.Int).Sqrt(p).Uint64()
}

// SafeMulDiv computes (a * b) / c safely using big.Int.
func SafeMulDiv(a, b, c uint64) uint64 {
	if c == 0 {
		return 0
	}
	bigA := new(big.Int).SetUint64(a)
	bigB := new(big.Int).SetUint64(b)
	bigC := new(big.Int).SetUint64(c)

	num := new(big.Int).Mul(bigA, bigB)
	res := new(big.Int).Div(num, bigC)

	return res.Uint64()
}

// IntSqrt returns the integer square root of n (truncated)
func IntSqrt(n uint64) uint64 {
	if n == 0 {
		return 0
	}
	x := n
	y := (x + 1) / 2
	for y < x {
		x = y
		y = (x + n/x) / 2
	}
	return x
}

// ContainsByteSlice() checks to see if the byte slice is within the list
func ContainsByteSlice(list [][]byte, target []byte) (found bool) {
	for _, item := range list {
		if bytes.Equal(item, target) {
			return
		}
	}
	return
}

type stringStruct struct {
	str unsafe.Pointer
	len int
}

//go:noescape
//go:linkname memhash runtime.memhash
func memhash(p unsafe.Pointer, h, s uintptr) uintptr

// MemHash is the hash function used by go map, it utilizes available hardware instructions(behaves
// as aeshash if aes instruction is available).
// NOTE: The hash seed changes for every process. So, this cannot be used as a persistent hash.
func MemHash(data []byte) uint64 {
	ss := (*stringStruct)(unsafe.Pointer(&data))
	return uint64(memhash(ss.str, 0, uintptr(ss.len)))
}

// RandSlice is a helper to generate a random byte slice of the given size.
func RandSlice(byteSize uint64) []byte {
	value := make([]byte, byteSize)
	rand.Read(value)
	return value
}

// Integer defines a constraint for integer types including big.Int
type Integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

// BigInt converts integer t into a bigInt
func BigInt[T Integer](t T) *big.Int {
	switch v := any(t).(type) {
	case int:
		return big.NewInt(int64(v))
	case int8:
		return big.NewInt(int64(v))
	case int16:
		return big.NewInt(int64(v))
	case int32:
		return big.NewInt(int64(v))
	case int64:
		return big.NewInt(v)
	case uint:
		return new(big.Int).SetUint64(uint64(v))
	case uint8:
		return new(big.Int).SetUint64(uint64(v))
	case uint16:
		return new(big.Int).SetUint64(uint64(v))
	case uint32:
		return new(big.Int).SetUint64(uint64(v))
	case uint64:
		return new(big.Int).SetUint64(v)
	default:
		return new(big.Int).SetUint64(0)
	}
}

func BigIntIsZero(i *big.Int) bool {
	return i.Cmp(big.NewInt(0)) == 0
}

func BigIntSub(x, y *big.Int) *big.Int {
	return new(big.Int).Sub(x, y)
}

// AtomicWriteFile writes data to a file atomically using write-and-move pattern
func AtomicWriteFile(filePath string, data []byte) error {
	// create temporary file in the same directory as the target file
	dir := filepath.Dir(filePath)
	tempFile, err := os.CreateTemp(dir, ".tmp_atomic_*")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	tempFilePath := tempFile.Name()
	// ensure temporary file is cleaned up if something goes wrong
	defer func() {
		tempFile.Close()
		os.Remove(tempFilePath)
	}()
	// write data to temporary file
	_, err = tempFile.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write to temporary file: %w", err)
	}
	// sync to ensure data is written to disk
	err = tempFile.Sync()
	if err != nil {
		return fmt.Errorf("failed to sync temporary file: %w", err)
	}
	// close temporary file before rename
	err = tempFile.Close()
	if err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}
	// atomically move temporary file to final destination
	err = os.Rename(tempFilePath, filePath)
	if err != nil {
		return fmt.Errorf("failed to rename temporary file to final destination: %w", err)
	}
	return nil
}
