package contract

// This file exposes minimal helpers needed by sibling plugin packages (e.g.
// the canoliq plugin) that cannot satisfy the unexported isPluginToFSM_Payload
// oneof interface from outside the contract package. Keeping these helpers in
// the contract package preserves encapsulation while letting other packages
// build PluginToFSM messages without forking the proto generation.

// NewPluginToFSM constructs a PluginToFSM message with the given request id
// and oneof payload. The accepted payload variants mirror the proto oneof.
func NewPluginToFSM(id uint64, payload any) (*PluginToFSM, *PluginError) {
	switch p := payload.(type) {
	case *PluginToFSM_Config:
		return &PluginToFSM{Id: id, Payload: p}, nil
	case *PluginToFSM_Genesis:
		return &PluginToFSM{Id: id, Payload: p}, nil
	case *PluginToFSM_Begin:
		return &PluginToFSM{Id: id, Payload: p}, nil
	case *PluginToFSM_Check:
		return &PluginToFSM{Id: id, Payload: p}, nil
	case *PluginToFSM_Deliver:
		return &PluginToFSM{Id: id, Payload: p}, nil
	case *PluginToFSM_End:
		return &PluginToFSM{Id: id, Payload: p}, nil
	case *PluginToFSM_StateRead:
		return &PluginToFSM{Id: id, Payload: p}, nil
	case *PluginToFSM_StateWrite:
		return &PluginToFSM{Id: id, Payload: p}, nil
	default:
		return nil, ErrInvalidMessageCast()
	}
}
