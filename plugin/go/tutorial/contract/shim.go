package contract

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// Shim for types and functions defined in plugin.go — test build only.

type Config struct {
	ChainId     uint64 `json:"chainId"`
	DataDirPath string `json:"dataDirPath"`
}

type Plugin struct{}

func (p *Plugin) StateRead(c *Contract, req *PluginStateReadRequest) (*PluginStateReadResponse, *PluginError) {
	return nil, ErrInternal()
}

func (p *Plugin) StateWrite(c *Contract, req *PluginStateWriteRequest) (*PluginStateWriteResponse, *PluginError) {
	return nil, ErrInternal()
}

func FromAny(any *anypb.Any) (proto.Message, *PluginError) {
	msg, err := anypb.UnmarshalNew(any, proto.UnmarshalOptions{})
	if err != nil {
		return nil, ErrFromAny(err)
	}
	return msg, nil
}

func Marshal(message any) ([]byte, *PluginError) {
	m, ok := message.(proto.Message)
	if !ok {
		return nil, ErrInternal()
	}
	b, err := proto.Marshal(m)
	if err != nil {
		return nil, ErrInternal()
	}
	return b, nil
}

func Unmarshal(protoBytes []byte, ptr any) *PluginError {
	m, ok := ptr.(proto.Message)
	if !ok {
		return ErrInternal()
	}
	if err := proto.Unmarshal(protoBytes, m); err != nil {
		return ErrInternal()
	}
	return nil
}

func JoinLenPrefix(toAppend ...[]byte) []byte {
	totalLen := 0
	for _, item := range toAppend {
		if item != nil {
			totalLen += 1 + len(item)
		}
	}
	res := make([]byte, 0, totalLen)
	for _, item := range toAppend {
		if item == nil {
			continue
		}
		res = append(res, byte(len(item)))
		res = append(res, item...)
	}
	return res
}
