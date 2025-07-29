package oracle

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
	CodeReadStateFile    lib.ErrorCode = 1
	CodeParseHeight      lib.ErrorCode = 2
	CodeWriteStateFile   lib.ErrorCode = 3
	CodeCreateDirectory  lib.ErrorCode = 4
	CodeGetHomeDirectory lib.ErrorCode = 5
	CodeUnmarshalOrder   lib.ErrorCode = 6
	CodeMarshalOrder     lib.ErrorCode = 7
	CodeOrderNotFound    lib.ErrorCode = 8
	CodeGetOrderBook     lib.ErrorCode = 9
	CodeAmountMismatch   lib.ErrorCode = 10
	CodeOrderNotVerified lib.ErrorCode = 11
	CodeOrderValidation  lib.ErrorCode = 12
	CodeBlockSequence    lib.ErrorCode = 13
	CodeChainReorg       lib.ErrorCode = 14
	CodeNilOrderBook     lib.ErrorCode = 15

	// OrderStore Module
	OrderStoreModule lib.ErrorModule = "order_store"

	// OracleStore Module Error Codes
	CodeValidateOrder lib.ErrorCode = 1
	CodeVerifyOrder   lib.ErrorCode = 2
	CodeReadOrder     lib.ErrorCode = 3
	CodeRemoveOrder   lib.ErrorCode = 4
	CodeWriteOrder    lib.ErrorCode = 5
)

// Error functions for Order Store module
func ErrValidateOrder(err error) lib.ErrorI {
	return lib.NewError(CodeValidateOrder, OrderStoreModule, "failed to validate order: "+err.Error())
}

func ErrVerifyOrder(err error) lib.ErrorI {
	return lib.NewError(CodeVerifyOrder, OrderStoreModule, "failed to verify order: "+err.Error())
}

func ErrReadOrder(err error) lib.ErrorI {
	return lib.NewError(CodeReadOrder, OrderStoreModule, "failed to read order: "+err.Error())
}

func ErrRemoveOrder(err error) lib.ErrorI {
	return lib.NewError(CodeRemoveOrder, OrderStoreModule, "failed to remove order: "+err.Error())
}

func ErrWriteOrder(err error) lib.ErrorI {
	return lib.NewError(CodeWriteOrder, OrderStoreModule, "failed to write order: "+err.Error())
}

// Error functions for Oracle module
func ErrReadStateFile(err error) lib.ErrorI {
	return lib.NewError(CodeReadStateFile, OracleModule, "failed to read oracle state file: "+err.Error())
}

func ErrParseState(err error) lib.ErrorI {
	return lib.NewError(CodeParseHeight, OracleModule, "failed to parse height: "+err.Error())
}

func ErrWriteStateFile(err error) lib.ErrorI {
	return lib.NewError(CodeWriteStateFile, OracleModule, "failed to oracle state file: "+err.Error())
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

func ErrOrderNotFoundInOrderBook(orderId string) lib.ErrorI {
	return lib.NewError(CodeOrderNotFound, OracleModule, "order not found in order book: "+orderId)
}

func ErrGetOrderBook(err error) lib.ErrorI {
	return lib.NewError(CodeGetOrderBook, OracleModule, "failed to get order book: "+err.Error())
}

func ErrAmountMismatch(transferAmount, orderAmount uint64) lib.ErrorI {
	return lib.NewError(CodeAmountMismatch, OracleModule, fmt.Sprintf("transfer amount %d does not match order amount %d", transferAmount, orderAmount))
}

func ErrOrderNotVerified(s string, err error) lib.ErrorI {
	return lib.NewError(CodeOrderNotVerified, OracleModule, "order not verified: "+err.Error())
}

func ErrOrderValidation(s string) lib.ErrorI {
	return lib.NewError(CodeOrderValidation, OracleModule, "order validation failure: "+s)
}

func ErrBlockSequence(message string) lib.ErrorI {
	return lib.NewError(CodeBlockSequence, OracleModule, "block sequence error: "+message)
}

func ErrChainReorg(message string) lib.ErrorI {
	return lib.NewError(CodeChainReorg, OracleModule, "chain reorganization detected: "+message)
}

func ErrNilOrderBook() lib.ErrorI {
	return lib.NewError(CodeNilOrderBook, OracleModule, "order book is nil")
}
