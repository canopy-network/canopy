package rpc

import (
	"net/http"

	"github.com/canopy-network/canopy/fsm/types"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
)

func heightIndexer(w http.ResponseWriter, r *http.Request, callback func(s lib.StoreI, h uint64, p lib.PageParams) (any, lib.ErrorI)) {
	req := new(paginatedHeightRequest)
	if ok := unmarshal(w, r, req); !ok {
		return
	}
	s, ok := setupStore(w)
	if !ok {
		return
	}
	defer s.Discard()
	if req.Height == 0 {
		req.Height = s.Version() - 1
	}
	p, err := callback(s, req.Height, req.PageParams)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	write(w, p, http.StatusOK)
}

func heightAndAddrIndexer(w http.ResponseWriter, r *http.Request, callback func(s lib.StoreI, h uint64, address lib.HexBytes) (any, lib.ErrorI)) {
	req := new(heightAndAddressRequest)
	if ok := unmarshal(w, r, req); !ok {
		return
	}
	s, ok := setupStore(w)
	if !ok {
		return
	}
	defer s.Discard()
	if req.Height == 0 {
		req.Height = s.Version() - 1
	}
	p, err := callback(s, req.Height, req.Address)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	write(w, p, http.StatusOK)
}

func heightAndIdIndexer(w http.ResponseWriter, r *http.Request, callback func(s lib.StoreI, h, id uint64) (any, lib.ErrorI)) {
	req := new(heightAndIdRequest)
	if ok := unmarshal(w, r, req); !ok {
		return
	}
	s, ok := setupStore(w)
	if !ok {
		return
	}
	defer s.Discard()
	if req.Height == 0 {
		req.Height = s.Version() - 1
	}
	p, err := callback(s, req.Height, req.ID)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	write(w, p, http.StatusOK)
}

func hashIndexer(w http.ResponseWriter, r *http.Request, callback func(s lib.StoreI, h lib.HexBytes) (any, lib.ErrorI)) {
	req := new(hashRequest)
	if ok := unmarshal(w, r, req); !ok {
		return
	}
	s, ok := setupStore(w)
	if !ok {
		return
	}
	defer s.Discard()
	bz, err := lib.StringToBytes(req.Hash)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	p, err := callback(s, bz)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	write(w, p, http.StatusOK)
}

func addrIndexer(w http.ResponseWriter, r *http.Request, callback func(s lib.StoreI, a crypto.AddressI, p lib.PageParams) (any, lib.ErrorI)) {
	req := new(paginatedAddressRequest)
	if ok := unmarshal(w, r, req); !ok {
		return
	}
	s, ok := setupStore(w)
	if !ok {
		return
	}
	defer s.Discard()
	if req.Address == nil {
		write(w, types.ErrAddressEmpty(), http.StatusBadRequest)
		return
	}
	p, err := callback(s, crypto.NewAddressFromBytes(req.Address), req.PageParams)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	write(w, p, http.StatusOK)
}

func pageIndexer(w http.ResponseWriter, r *http.Request, callback func(s lib.StoreI, a crypto.AddressI, p lib.PageParams) (any, lib.ErrorI)) {
	req := new(paginatedAddressRequest)
	if ok := unmarshal(w, r, req); !ok {
		return
	}
	s, ok := setupStore(w)
	if !ok {
		return
	}
	defer s.Discard()
	p, err := callback(s, crypto.NewAddressFromBytes(req.Address), req.PageParams)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	write(w, p, http.StatusOK)
}
