package rpc

import (
	"net/http"

	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/store"
	"github.com/julienschmidt/httprouter"
	"github.com/nsf/jsondiff"
)

func getStateMachineFromHeightParams(w http.ResponseWriter, r *http.Request, ptr queryWithHeight) (sm *fsm.StateMachine, ok bool) {
	if ok = unmarshal(w, r, ptr); !ok {
		return
	}
	return getStateMachineWithHeight(ptr.GetHeight(), w)
}

func getDoubleStateMachineFromHeightParams(w http.ResponseWriter, r *http.Request, p httprouter.Params) (sm1, sm2 *fsm.StateMachine, o *jsondiff.Options, ok bool) {
	request, opts := new(heightsRequest), jsondiff.Options{}
	switch r.Method {
	case http.MethodGet:
		opts = jsondiff.DefaultHTMLOptions()
		opts.ChangedSeparator = " <- "
		if err := r.ParseForm(); err != nil {
			ok = false
			write(w, err, http.StatusBadRequest)
			return
		}
		request.Height = parseUint64FromString(r.Form.Get("height"))
		request.StartHeight = parseUint64FromString(r.Form.Get("startHeight"))
	case http.MethodPost:
		opts = jsondiff.DefaultConsoleOptions()
		if ok = unmarshal(w, r, request); !ok {
			return
		}
	}
	sm1, ok = getStateMachineWithHeight(request.Height, w)
	if !ok {
		return
	}
	if request.StartHeight == 0 {
		request.StartHeight = sm1.Height() - 1
	}
	sm2, ok = getStateMachineWithHeight(request.StartHeight, w)
	o = &opts
	return
}

func getStateMachineWithHeight(height uint64, w http.ResponseWriter) (sm *fsm.StateMachine, ok bool) {
	return setupStateMachine(height, w)
}

func setupStore(w http.ResponseWriter) (lib.StoreI, bool) {
	s, err := store.NewStoreWithDB(db, logger, false)
	if err != nil {
		write(w, ErrNewStore(err), http.StatusInternalServerError)
		return nil, false
	}
	return s, true
}

// TODO likely a memory leak here from un-discarded stores
// Investiage memory leak
func setupStateMachine(height uint64, w http.ResponseWriter) (*fsm.StateMachine, bool) {

	// Investigate  memory use of state. State.Discard needs to be called
	state, err := app.FSM.TimeMachine(height)
	if err != nil {
		write(w, ErrTimeMachine(err), http.StatusInternalServerError)
		return nil, false
	}
	return state, true
}
