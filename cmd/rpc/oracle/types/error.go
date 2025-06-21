package types

import (
	"fmt"
	"math"

	"github.com/canopy-network/canopy/lib"
)

const (
	NoCode lib.ErrorCode = math.MaxUint32

	// Oracle Module
	OracleModule lib.ErrorModule = "oracle"

	// Oracle Module Error Codes
	CodeReadHeightFile   lib.ErrorCode = 1
	CodeParseHeight      lib.ErrorCode = 2
	CodeWriteHeightFile  lib.ErrorCode = 3
	CodeCreateDirectory  lib.ErrorCode = 4
	CodeGetHomeDirectory lib.ErrorCode = 5
	CodeUnmarshalOrder   lib.ErrorCode = 6
	CodeMarshalOrder     lib.ErrorCode = 7
	CodeOrderNotFound    lib.ErrorCode = 8
	CodeGetOrderBook     lib.ErrorCode = 9
	CodeAmountMismatch   lib.ErrorCode = 10
	CodePersistOrder     lib.ErrorCode = 11
	CodeOrderNotVerified lib.ErrorCode = 12
)

// Error functions for Oracle module
func ErrReadHeightFile(err error) lib.ErrorI {
	return lib.NewError(CodeReadHeightFile, OracleModule, "failed to read height file: "+err.Error())
}

func ErrParseHeight(err error) lib.ErrorI {
	return lib.NewError(CodeParseHeight, OracleModule, "failed to parse height: "+err.Error())
}

func ErrWriteHeightFile(err error) lib.ErrorI {
	return lib.NewError(CodeWriteHeightFile, OracleModule, "failed to write height file: "+err.Error())
}

func ErrCreateDirectory(err error) lib.ErrorI {
	return lib.NewError(CodeCreateDirectory, OracleModule, "failed to create directory: "+err.Error())
}

func ErrGetHomeDirectory(err error) lib.ErrorI {
	return lib.NewError(CodeGetHomeDirectory, OracleModule, "failed to get home directory: "+err.Error())
}

func ErrUnmarshalOrder(err error) lib.ErrorI {
	return lib.NewError(CodeUnmarshalOrder, OracleModule, "failed to unmarshal order: "+err.Error())
}

func ErrMarshalOrder(err error) lib.ErrorI {
	return lib.NewError(CodeMarshalOrder, OracleModule, "failed to marshal order: "+err.Error())
}

func ErrOrderNotFoundInBook(orderId string) lib.ErrorI {
	return lib.NewError(CodeOrderNotFound, OracleModule, "order not found in order book: "+orderId)
}

func ErrGetOrderBook(err error) lib.ErrorI {
	return lib.NewError(CodeGetOrderBook, OracleModule, "failed to get order book: "+err.Error())
}

func ErrAmountMismatch(transferAmount, orderAmount uint64) lib.ErrorI {
	return lib.NewError(CodeAmountMismatch, OracleModule, fmt.Sprintf("transfer amount %d does not match order amount %d", transferAmount, orderAmount))
}

func ErrPersistOrder(err error) lib.ErrorI {
	return lib.NewError(CodePersistOrder, OracleModule, "failed to persist order: "+err.Error())
}

func ErrOrderNotVerified(s string, err error) lib.ErrorI {
	return lib.NewError(CodeOrderNotVerified, OracleModule, "order not verified: "+err.Error())
}
