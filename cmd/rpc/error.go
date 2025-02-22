package rpc

import (
	"fmt"

	"github.com/canopy-network/canopy/lib"
)

func ErrServerTimeout() lib.ErrorI {
	return lib.NewError(lib.CodeRPCTimeout, lib.RPCModule, "server timeout")
}

func ErrInvalidParams(err error) lib.ErrorI {
	bz, _ := lib.MarshalJSON(err)
	return lib.NewError(lib.CodeInvalidParams, lib.RPCModule, fmt.Sprintf("invalid params: %s", string(bz)))
}

func ErrNewFSM(err error) lib.ErrorI {
	return lib.NewError(lib.CodeNewFSM, lib.RPCModule, fmt.Sprintf("new fsm failed with err: %s", err.Error()))
}

func ErrNewStore(err error) lib.ErrorI {
	return lib.NewError(lib.CodeNewFSM, lib.RPCModule, fmt.Sprintf("new store failed with err: %s", err.Error()))
}

func ErrTimeMachine(err error) lib.ErrorI {
	return lib.NewError(lib.CodeTimeMachine, lib.RPCModule, fmt.Sprintf("fsm.TimeMachine() failed with err: %s", err.Error()))
}

func ErrPostRequest(err error) lib.ErrorI {
	return lib.NewError(lib.CodePostRequest, lib.RPCModule, fmt.Sprintf("http.Post() failed with err: %s", err.Error()))
}

func ErrGetRequest(err error) lib.ErrorI {
	return lib.NewError(lib.CodeGetRequest, lib.RPCModule, fmt.Sprintf("http.Get() failed with err: %s", err.Error()))
}

func ErrHttpStatus(status string, statusCode int, body []byte) lib.ErrorI {
	return lib.NewError(lib.CodeHttpStatus, lib.RPCModule, fmt.Sprintf("http response bad status %s with code %d and body %s", status, statusCode, body))
}

func ErrReadBody(err error) lib.ErrorI {
	return lib.NewError(lib.CodeReadBody, lib.RPCModule, fmt.Sprintf("io.ReadAll(http.ResponseBody) failed with err: %s", err.Error()))
}

func ErrStringToCommittee(s string) lib.ErrorI {
	return lib.NewError(lib.CodeStringToCommittee, lib.RPCModule, fmt.Sprintf("committee arg %s is invalid, requires a comma separated list of <chainId>=<percent> ex. 0=50,21=25,99=25", s))
}
