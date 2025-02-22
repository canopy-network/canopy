package rpc

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"github.com/alecthomas/units"
	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/fsm/types"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
)

func unmarshal(w http.ResponseWriter, r *http.Request, ptr interface{}) bool {
	bz, err := io.ReadAll(io.LimitReader(r.Body, int64(units.MB)))
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return false
	}
	defer func() { _ = r.Body.Close() }()
	if err = json.Unmarshal(bz, ptr); err != nil {
		write(w, err, http.StatusBadRequest)
		return false
	}
	return true
}

func write(w http.ResponseWriter, payload interface{}, code int) {
	w.Header().Set(ContentType, ApplicationJSON)
	w.WriteHeader(code)
	bz, _ := json.MarshalIndent(payload, "", "  ")
	if _, err := w.Write(bz); err != nil {
		logger.Error(err.Error())
	}
}

func parseUint64FromString(s string) uint64 {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return uint64(i)
}

func orderParams(w http.ResponseWriter, r *http.Request, callback func(s *fsm.StateMachine, request *orderRequest) (any, lib.ErrorI)) {
	req := new(orderRequest)
	state, ok := getStateMachineFromHeightParams(w, r, req)
	if !ok {
		return
	}
	p, err := callback(state, req)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	write(w, p, http.StatusOK)
}

func heightParams(w http.ResponseWriter, r *http.Request, callback func(s *fsm.StateMachine) (any, lib.ErrorI)) {
	req := new(heightRequest)
	state, ok := getStateMachineFromHeightParams(w, r, req)
	if !ok {
		return
	}
	p, err := callback(state)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	write(w, p, http.StatusOK)
}

func heightPaginated(w http.ResponseWriter, r *http.Request, callback func(s *fsm.StateMachine, p *paginatedHeightRequest) (any, lib.ErrorI)) {
	req := new(paginatedHeightRequest)
	state, ok := getStateMachineFromHeightParams(w, r, req)
	if !ok {
		return
	}
	p, err := callback(state, req)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	write(w, p, http.StatusOK)
}

func heightAndAddressParams(w http.ResponseWriter, r *http.Request, callback func(*fsm.StateMachine, lib.HexBytes) (any, lib.ErrorI)) {
	req := new(heightAndAddressRequest)
	state, ok := getStateMachineFromHeightParams(w, r, req)
	if !ok {
		return
	}
	if req.Address == nil {
		write(w, types.ErrAddressEmpty(), http.StatusBadRequest)
		return
	}
	p, err := callback(state, req.Address)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	write(w, p, http.StatusOK)
}

func heightAndIdParams(w http.ResponseWriter, r *http.Request, callback func(*fsm.StateMachine, uint64) (any, lib.ErrorI)) {
	req := new(heightAndIdRequest)
	state, ok := getStateMachineFromHeightParams(w, r, req)
	if !ok {
		return
	}
	p, err := callback(state, req.ID)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	write(w, p, http.StatusOK)
}

func getFeeFromState(w http.ResponseWriter, ptr *txRequest, messageName string, buyOrder ...bool) lib.ErrorI {
	state, ok := getStateMachineWithHeight(0, w)
	if !ok {
		return ErrTimeMachine(fmt.Errorf("getStateMachineWithHeight failed"))
	}
	defer state.Discard()
	minimumFee, err := state.GetFeeForMessageName(messageName)
	if err != nil {
		return err
	}
	if len(buyOrder) == 1 && buyOrder[0] == true {
		params, e := state.GetParamsVal()
		if e != nil {
			return e
		}
		minimumFee *= params.BuyOrderFeeMultiplier
	}
	if ptr.Fee == 0 {
		ptr.Fee = minimumFee
	}
	if ptr.Fee < minimumFee {
		return types.ErrTxFeeBelowStateLimit()
	}
	return nil
}

func stringToCommittees(s string) (committees []uint64, error error) {
	i, err := strconv.ParseUint(s, 10, 64) // single int is an option for subsidy txn
	if err == nil {
		return []uint64{i}, nil
	}
	commaSeparatedArr := strings.Split(strings.ReplaceAll(s, " ", ""), ",")
	if len(commaSeparatedArr) == 0 {
		return nil, ErrStringToCommittee(s)
	}
	for _, c := range commaSeparatedArr {
		ui, e := strconv.ParseUint(c, 10, 64)
		if e != nil {
			return nil, e
		}
		committees = append(committees, ui)
	}
	return
}

func getAddressFromNickname(ptr *txRequest, keystore *crypto.Keystore) {
	if len(ptr.Signer) == 0 && ptr.SignerNickname != "" {
		addressString := keystore.NicknameMap[ptr.SignerNickname]
		addressBytes, _ := hex.DecodeString(addressString)
		ptr.Signer = addressBytes
	}

	if len(ptr.Address) == 0 && ptr.Nickname != "" {
		addressString := keystore.NicknameMap[ptr.Nickname]
		addressBytes, _ := hex.DecodeString(addressString)
		ptr.Address = addressBytes
	}
}

func fdCount(pid int32) (int, error) {
	cmd := []string{"-a", "-n", "-P", "-p", strconv.Itoa(int(pid))}
	out, err := _exec("lsof", cmd...)
	if err != nil {
		return 0, err
	}
	lines := strings.Split(string(out), "\n")
	var ret []string
	for _, l := range lines[1:] {
		if len(l) == 0 {
			continue
		}
		ret = append(ret, l)
	}
	return len(ret), nil
}

func _exec(name string, arg ...string) ([]byte, error) {
	cmd := exec.Command(name, arg...)

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	if err := cmd.Start(); err != nil {
		return buf.Bytes(), err
	}

	if err := cmd.Wait(); err != nil {
		return buf.Bytes(), err
	}

	return buf.Bytes(), nil
}
