package p2p

import (
	"fmt"
	lib "github.com/ginchuco/ginchu/types"
	"google.golang.org/protobuf/proto"
)

func ErrUnknownP2PMsg(t proto.Message) lib.ErrorI {
	return lib.NewError(lib.CodeUnknownP2PMessage, lib.P2PModule, fmt.Sprintf("unknown p2p message: %T", t))
}

func ErrFailedRead(err error) lib.ErrorI {
	return lib.NewError(lib.CodeFailedRead, lib.P2PModule, fmt.Sprintf("read() failed with err: %s", err.Error()))
}

func ErrFailedReadFull(err error) lib.ErrorI {
	return lib.NewError(lib.CodeFailedReadFull, lib.P2PModule, fmt.Sprintf("readFull() failed with err: %s", err.Error()))
}

func ErrFailedWrite(err error) lib.ErrorI {
	return lib.NewError(lib.CodeFailedWrite, lib.P2PModule, fmt.Sprintf("write() failed with err: %s", err.Error()))
}

func ErrMaxMessageSize() lib.ErrorI {
	return lib.NewError(lib.CodeMaxMessageSize, lib.P2PModule, "max message size")
}

func ErrIsBlacklisted() lib.ErrorI {
	return lib.NewError(lib.CodeBlacklisted, lib.P2PModule, "blacklisted man-in-the-middle id")
}

func ErrPongTimeout() lib.ErrorI {
	return lib.NewError(lib.CodePongTimeout, lib.P2PModule, "pong timeout")
}

func ErrErrorGroup(err error) lib.ErrorI {
	return lib.NewError(lib.CodeErrorGroup, lib.P2PModule, fmt.Sprintf("error group failed with err: %s", err.Error()))
}

func ErrConnDecryptFailed(err error) lib.ErrorI {
	return lib.NewError(lib.CodeConnDecrypt, lib.P2PModule, fmt.Sprintf("conn.decrypt failed with err: %s", err.Error()))
}

func ErrPeerAlreadyExists(s string) lib.ErrorI {
	return lib.NewError(lib.CodePeerAlreadyExists, lib.P2PModule, fmt.Sprintf("peer %s already exists", s))
}

func ErrPeerNotFound(s string) lib.ErrorI {
	return lib.NewError(lib.CodePeerNotFound, lib.P2PModule, fmt.Sprintf("peer %s not found", s))
}

func ErrChunkLargerThanMax() lib.ErrorI {
	return lib.NewError(lib.CodeChunkLargerThanMax, lib.P2PModule, "chunk larger than max")
}

func ErrFailedChallenge() lib.ErrorI {
	return lib.NewError(lib.CodeFailedChallenge, lib.P2PModule, "failed challenge")
}

func ErrFailedDiffieHellman(err error) lib.ErrorI {
	return lib.NewError(lib.CodeFailedDiffieHellman, lib.P2PModule, fmt.Sprintf("diffie hellman failed with err: %s", err.Error()))
}

func ErrFailedHKDF(err error) lib.ErrorI {
	return lib.NewError(lib.CodeFailedHKDF, lib.P2PModule, fmt.Sprintf("hkdf failed with err: %s", err.Error()))
}

func ErrFailedDial(err error) lib.ErrorI {
	return lib.NewError(lib.CodeFailedDial, lib.P2PModule, fmt.Sprintf("net.dial failed with err: %s", err.Error()))
}

func ErrMismatchPeerPublicKey(expected, got []byte) lib.ErrorI {
	return lib.NewError(lib.CodeMismatchPeerPublicKey, lib.P2PModule, fmt.Sprintf("mismatch peer public key: expected %s, got %s", lib.BytesToString(expected), lib.BytesToString(got)))
}

func ErrFailedListen(err error) lib.ErrorI {
	return lib.NewError(lib.CodeFailedListen, lib.P2PModule, fmt.Sprintf("net.listen() failed with err: %s", err.Error()))
}

func ErrHostAndPortFromRemote(err error) lib.ErrorI {
	return lib.NewError(lib.CodeHostAndPortFromRemote, lib.P2PModule, fmt.Sprintf("net.hostAndPortFromRemote() failed with err: %s", err.Error()))
}

func ErrIPLookup(err error) lib.ErrorI {
	return lib.NewError(lib.CodeIPLookup, lib.P2PModule, fmt.Sprintf("net.ipLookup() failed with err: %s", err.Error()))
}

func ErrBannedIP(s string) lib.ErrorI {
	return lib.NewError(lib.CodeBannedIP, lib.P2PModule, fmt.Sprintf("banned IP attempted to connect: %s", s))
}

func ErrBannedID(s string) lib.ErrorI {
	return lib.NewError(lib.CodeBannedID, lib.P2PModule, fmt.Sprintf("banned ID attempted to connect: %s", s))
}

func ErrNonTCPAddress() lib.ErrorI {
	return lib.NewError(lib.CodeNonTCPAddr, lib.P2PModule, "non tcp address")
}

func ErrMaxOutbound() lib.ErrorI {
	return lib.NewError(lib.CodeMaxOutbound, lib.P2PModule, "max outbound peers")
}

func ErrMaxInbound() lib.ErrorI {
	return lib.NewError(lib.CodeMaxInbound, lib.P2PModule, "max inbound peers")
}