package rpc

import (
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/canopy-network/canopy/fsm/types"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/canopy/lib/crypto"
	"github.com/julienschmidt/httprouter"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
	"github.com/shirou/gopsutil/process"
)

func newKeystore(w http.ResponseWriter, path string) (k *crypto.Keystore, ok bool) {
	k, err := crypto.NewKeystoreFromFile(path)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	ok = true
	return
}

func (s *Server) keystoreHandler(w http.ResponseWriter, r *http.Request, callback func(keystore *crypto.Keystore, ptr *keystoreRequest) (any, error)) {
	keystore, ok := newKeystore(w, s.config.DataDirPath)
	if !ok {
		return
	}
	ptr := new(keystoreRequest)
	if ok = unmarshal(w, r, ptr); !ok {
		return
	}
	p, err := callback(keystore, ptr)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	write(w, p, http.StatusOK)
}

func (s *Server) Keystore(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	keystore, err := crypto.NewKeystoreFromFile(s.config.DataDirPath)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	write(w, keystore, http.StatusOK)
}

func (s *Server) KeystoreNewKey(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	s.keystoreHandler(w, r, func(k *crypto.Keystore, ptr *keystoreRequest) (any, error) {
		pk, err := crypto.NewBLS12381PrivateKey()
		if err != nil {
			return nil, err
		}
		address, err := k.ImportRaw(pk.Bytes(), ptr.Password, crypto.ImportRawOpts{
			Nickname: ptr.Nickname,
		})
		if err != nil {
			return nil, err
		}
		return address, k.SaveToFile(s.config.DataDirPath)
	})
}

func (s *Server) KeystoreImport(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	s.keystoreHandler(w, r, func(k *crypto.Keystore, ptr *keystoreRequest) (any, error) {
		if err := k.Import(&ptr.EncryptedPrivateKey, crypto.ImportOpts{
			Address:  ptr.Address,
			Nickname: ptr.Nickname,
		}); err != nil {
			return nil, err
		}
		return ptr.Address, k.SaveToFile(s.config.DataDirPath)
	})
}

func (s *Server) KeystoreImportRaw(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	s.keystoreHandler(w, r, func(k *crypto.Keystore, ptr *keystoreRequest) (any, error) {
		address, err := k.ImportRaw(ptr.PrivateKey, ptr.Password, crypto.ImportRawOpts{
			Nickname: ptr.Nickname,
		})
		if err != nil {
			return nil, err
		}
		return address, k.SaveToFile(s.config.DataDirPath)
	})
}

func (s *Server) KeystoreDelete(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	s.keystoreHandler(w, r, func(k *crypto.Keystore, ptr *keystoreRequest) (any, error) {
		k.DeleteKey(crypto.DeleteOpts{
			Address:  ptr.Address,
			Nickname: ptr.Nickname,
		})
		return ptr.Address, k.SaveToFile(s.config.DataDirPath)
	})
}

func (s *Server) KeystoreGetKeyGroup(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	s.keystoreHandler(w, r, func(k *crypto.Keystore, ptr *keystoreRequest) (any, error) {
		return k.GetKeyGroup(ptr.Password, crypto.GetKeyGroupOpts{
			Address:  ptr.Address,
			Nickname: ptr.Nickname,
		})
	})
}

func (s *Server) TransactionSend(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	s.txHandler(w, r, func(p crypto.PrivateKeyI, ptr *txRequest) (lib.TransactionI, error) {
		toAddress, err := crypto.NewAddressFromString(ptr.Output)
		if err != nil {
			return nil, err
		}
		if err = getFeeFromState(w, ptr, types.MessageSendName); err != nil {
			return nil, err
		}
		return types.NewSendTransaction(p, toAddress, ptr.Amount, s.config.NetworkID, s.config.ChainId, ptr.Fee, s.controller.ChainHeight(), ptr.Memo)
	})
}

func (s *Server) TransactionStake(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	s.txHandler(w, r, func(p crypto.PrivateKeyI, ptr *txRequest) (lib.TransactionI, error) {
		outputAddress, err := crypto.NewAddressFromString(ptr.Output)
		if err != nil {
			return nil, err
		}
		committees, err := stringToCommittees(ptr.Committees)
		if err != nil {
			return nil, err
		}
		pk, err := crypto.NewPublicKeyFromString(ptr.PubKey)
		if err != nil {
			return nil, err
		}
		if err = getFeeFromState(w, ptr, types.MessageStakeName); err != nil {
			return nil, err
		}
		return types.NewStakeTx(p, pk.Bytes(), outputAddress, ptr.NetAddress, committees, ptr.Amount, s.config.NetworkID, s.config.ChainId, ptr.Fee, s.controller.ChainHeight(), ptr.Delegate, ptr.EarlyWithdrawal, ptr.Memo)
	})
}

func (s *Server) TransactionEditStake(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	s.txHandler(w, r, func(p crypto.PrivateKeyI, ptr *txRequest) (lib.TransactionI, error) {
		outputAddress, err := crypto.NewAddressFromString(ptr.Output)
		if err != nil {
			return nil, err
		}
		committees, err := stringToCommittees(ptr.Committees)
		if err != nil {
			return nil, err
		}
		if err = getFeeFromState(w, ptr, types.MessageEditStakeName); err != nil {
			return nil, err
		}
		return types.NewEditStakeTx(p, crypto.NewAddress(ptr.Address), outputAddress, ptr.NetAddress, committees, ptr.Amount, s.config.NetworkID, s.config.ChainId, ptr.Fee, s.controller.ChainHeight(), ptr.EarlyWithdrawal, ptr.Memo)
	})
}

func (s *Server) TransactionUnstake(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	s.txHandler(w, r, func(p crypto.PrivateKeyI, ptr *txRequest) (lib.TransactionI, error) {
		if err := getFeeFromState(w, ptr, types.MessageUnstakeName); err != nil {
			return nil, err
		}
		return types.NewUnstakeTx(p, crypto.NewAddress(ptr.Address), s.config.NetworkID, s.config.ChainId, ptr.Fee, s.controller.ChainHeight(), ptr.Memo)
	})
}

func (s *Server) TransactionPause(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	s.txHandler(w, r, func(p crypto.PrivateKeyI, ptr *txRequest) (lib.TransactionI, error) {
		if err := getFeeFromState(w, ptr, types.MessagePauseName); err != nil {
			return nil, err
		}
		return types.NewPauseTx(p, crypto.NewAddress(ptr.Address), s.config.NetworkID, s.config.ChainId, ptr.Fee, s.controller.ChainHeight(), ptr.Memo)
	})
}

func (s *Server) TransactionUnpause(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	s.txHandler(w, r, func(p crypto.PrivateKeyI, ptr *txRequest) (lib.TransactionI, error) {
		if err := getFeeFromState(w, ptr, types.MessageUnpauseName); err != nil {
			return nil, err
		}
		return types.NewUnpauseTx(p, crypto.NewAddress(ptr.Address), s.config.NetworkID, s.config.ChainId, ptr.Fee, s.controller.ChainHeight(), ptr.Memo)
	})
}

func (s *Server) TransactionChangeParam(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	s.txHandler(w, r, func(p crypto.PrivateKeyI, ptr *txRequest) (lib.TransactionI, error) {
		ptr.ParamSpace = types.FormatParamSpace(ptr.ParamSpace)
		if err := getFeeFromState(w, ptr, types.MessageChangeParameterName); err != nil {
			return nil, err
		}
		if ptr.ParamKey == types.ParamProtocolVersion {
			return types.NewChangeParamTxString(p, ptr.ParamSpace, ptr.ParamKey, ptr.ParamValue, ptr.StartBlock, ptr.EndBlock, s.config.NetworkID, s.config.ChainId, ptr.Fee, s.controller.ChainHeight(), ptr.Memo)
		}
		paramValue, err := strconv.ParseUint(ptr.ParamValue, 10, 64)
		if err != nil {
			return nil, err
		}
		return types.NewChangeParamTxUint64(p, ptr.ParamSpace, ptr.ParamKey, paramValue, ptr.StartBlock, ptr.EndBlock, s.config.NetworkID, s.config.ChainId, ptr.Fee, s.controller.ChainHeight(), ptr.Memo)
	})
}

func (s *Server) TransactionDAOTransfer(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	s.txHandler(w, r, func(p crypto.PrivateKeyI, ptr *txRequest) (lib.TransactionI, error) {
		if err := getFeeFromState(w, ptr, types.MessageDAOTransferName); err != nil {
			return nil, err
		}
		return types.NewDAOTransferTx(p, ptr.Amount, ptr.StartBlock, ptr.EndBlock, s.config.NetworkID, s.config.ChainId, ptr.Fee, s.controller.ChainHeight(), ptr.Memo)
	})
}

func (s *Server) TransactionSubsidy(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	s.txHandler(w, r, func(p crypto.PrivateKeyI, ptr *txRequest) (lib.TransactionI, error) {
		chainId := uint64(0)
		if c, err := stringToCommittees(ptr.Committees); err == nil {
			chainId = c[0]
		}
		if err := getFeeFromState(w, ptr, types.MessageSubsidyName); err != nil {
			return nil, err
		}
		return types.NewSubsidyTx(p, ptr.Amount, chainId, ptr.OpCode, s.config.NetworkID, s.config.ChainId, ptr.Fee, s.controller.ChainHeight(), ptr.Memo)
	})
}

func (s *Server) TransactionCreateOrder(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	s.txHandler(w, r, func(p crypto.PrivateKeyI, ptr *txRequest) (lib.TransactionI, error) {
		chainId := uint64(0)
		if c, err := stringToCommittees(ptr.Committees); err == nil {
			chainId = c[0]
		}
		if err := getFeeFromState(w, ptr, types.MessageCreateOrderName); err != nil {
			return nil, err
		}
		return types.NewCreateOrderTx(p, ptr.Amount, ptr.ReceiveAmount, chainId, ptr.ReceiveAddress, s.config.NetworkID, s.config.ChainId, ptr.Fee, s.controller.ChainHeight(), ptr.Memo)
	})
}

func (s *Server) TransactionEditOrder(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	s.txHandler(w, r, func(p crypto.PrivateKeyI, ptr *txRequest) (lib.TransactionI, error) {
		chainId := uint64(0)
		if c, err := stringToCommittees(ptr.Committees); err == nil {
			chainId = c[0]
		}
		if err := getFeeFromState(w, ptr, types.MessageEditOrderName); err != nil {
			return nil, err
		}
		return types.NewEditOrderTx(p, ptr.OrderId, ptr.Amount, ptr.ReceiveAmount, chainId, ptr.ReceiveAddress, s.config.NetworkID, s.config.ChainId, ptr.Fee, s.controller.ChainHeight(), ptr.Memo)
	})
}

func (s *Server) TransactionDeleteOrder(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	s.txHandler(w, r, func(p crypto.PrivateKeyI, ptr *txRequest) (lib.TransactionI, error) {
		chainId := uint64(0)
		if c, err := stringToCommittees(ptr.Committees); err == nil {
			chainId = c[0]
		}
		if err := getFeeFromState(w, ptr, types.MessageDeleteOrderName); err != nil {
			return nil, err
		}
		return types.NewDeleteOrderTx(p, ptr.OrderId, chainId, s.config.NetworkID, s.config.ChainId, ptr.Fee, s.controller.ChainHeight(), ptr.Memo)
	})
}

func (s *Server) TransactionBuyOrder(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	s.txHandler(w, r, func(p crypto.PrivateKeyI, ptr *txRequest) (lib.TransactionI, error) {
		if err := getFeeFromState(w, ptr, types.MessageSendName, true); err != nil {
			return nil, err
		}
		return types.NewBuyOrderTx(p, lib.BuyOrder{OrderId: ptr.OrderId, BuyerSendAddress: p.PublicKey().Address().Bytes(), BuyerReceiveAddress: ptr.ReceiveAddress}, s.config.NetworkID, s.config.ChainId, s.controller.ChainHeight(), ptr.Fee)
	})
}

func (s *Server) TransactionStartPoll(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	s.txHandler(w, r, func(p crypto.PrivateKeyI, ptr *txRequest) (lib.TransactionI, error) {
		if err := getFeeFromState(w, ptr, types.MessageSendName); err != nil {
			return nil, err
		}
		return types.NewStartPollTransaction(p, ptr.PollJSON, s.config.NetworkID, s.config.ChainId, s.controller.ChainHeight(), ptr.Fee)
	})
}

func (s *Server) TransactionVotePoll(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	s.txHandler(w, r, func(p crypto.PrivateKeyI, ptr *txRequest) (lib.TransactionI, error) {
		if err := getFeeFromState(w, ptr, types.MessageSendName); err != nil {
			return nil, err
		}
		return types.NewVotePollTransaction(p, ptr.PollJSON, ptr.PollApprove, s.config.NetworkID, s.config.ChainId, s.controller.ChainHeight(), ptr.Fee)
	})
}

func (s *Server) ConsensusInfo(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if err := r.ParseForm(); err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	summary, err := s.controller.ConsensusSummary()
	if err != nil {
		write(w, err, http.StatusInternalServerError)
		return
	}
	w.Header().Set(ContentType, ApplicationJSON)
	w.WriteHeader(http.StatusOK)
	if _, e := w.Write(summary); e != nil {
		logger.Error(e.Error())
	}
}

func (s *Server) PeerInfo(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	peers, numInbound, numOutbound := s.controller.P2P.GetAllInfos()
	write(w, &peerInfoResponse{
		ID:          s.controller.P2P.ID(),
		NumPeers:    numInbound + numOutbound,
		NumInbound:  numInbound,
		NumOutbound: numOutbound,
		Peers:       peers,
	}, http.StatusOK)
}

func (s *Server) PeerBook(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	write(w, s.controller.P2P.GetBookPeers(), http.StatusOK)
}

func (s *Server) Config(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	write(w, s.config, http.StatusOK)
}

func (s *Server) txHandler(w http.ResponseWriter, r *http.Request, callback func(privateKey crypto.PrivateKeyI, ptr *txRequest) (lib.TransactionI, error)) {
	ptr := new(txRequest)
	if ok := unmarshal(w, r, ptr); !ok {
		return
	}
	keystore, ok := newKeystore(w, s.config.DataDirPath)
	if !ok {
		return
	}
	getAddressFromNickname(ptr, keystore)

	signer := ptr.Signer
	if len(signer) == 0 {
		signer = ptr.Address
	}
	privateKey, err := keystore.GetKey(signer, ptr.Password)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	ptr.PubKey = privateKey.PublicKey().String()
	p, err := callback(privateKey, ptr)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	if ptr.Submit {
		s.submitTx(w, p)
	} else {
		bz, e := lib.MarshalJSONIndent(p)
		if e != nil {
			write(w, e, http.StatusBadRequest)
			return
		}
		if _, err = w.Write(bz); err != nil {
			logger.Error(err.Error())
			return
		}
	}
}

func (s *Server) AddVote(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	proposals := make(types.GovProposals)
	if err := proposals.NewFromFile(s.config.DataDirPath); err != nil {
		write(w, err, http.StatusInternalServerError)
		return
	}
	j := new(voteRequest)
	if !unmarshal(w, r, j) {
		return
	}
	prop, err := types.NewProposalFromBytes(j.Proposal)
	if err != nil || prop.GetEndHeight() == 0 {
		write(w, err, http.StatusBadRequest)
		return
	}
	if err = proposals.Add(prop, j.Approve); err != nil {
		write(w, err, http.StatusInternalServerError)
		return
	}
	if err = proposals.SaveToFile(s.config.DataDirPath); err != nil {
		write(w, err, http.StatusInternalServerError)
		return
	}
	write(w, j, http.StatusOK)
}

func (s *Server) DelVote(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	proposals := make(types.GovProposals)
	if err := proposals.NewFromFile(s.config.DataDirPath); err != nil {
		write(w, err, http.StatusInternalServerError)
		return
	}
	j := new(voteRequest)
	if !unmarshal(w, r, j) {
		return
	}
	prop, err := types.NewProposalFromBytes(j.Proposal)
	if err != nil {
		write(w, err, http.StatusBadRequest)
		return
	}
	proposals.Del(prop)
	if err = proposals.SaveToFile(s.config.DataDirPath); err != nil {
		write(w, err, http.StatusInternalServerError)
		return
	}
	write(w, j, http.StatusOK)
}

func (s *Server) ResourceUsage(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	pm, err := mem.VirtualMemory() // os memory
	if err != nil {
		write(w, err, http.StatusInternalServerError)
		return
	}
	c, err := cpu.Times(false) // os cpu
	if err != nil {
		write(w, err, http.StatusInternalServerError)
		return
	}
	cp, err := cpu.Percent(0, false) // os cpu percent
	if err != nil {
		write(w, err, http.StatusInternalServerError)
		return
	}
	d, err := disk.Usage("/") // os disk
	if err != nil {
		write(w, err, http.StatusInternalServerError)
		return
	}
	p, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		write(w, err, http.StatusInternalServerError)
		return
	}
	name, err := p.Name()
	if err != nil {
		write(w, err, http.StatusInternalServerError)
		return
	}
	cpuPercent, err := p.CPUPercent()
	if err != nil {
		write(w, err, http.StatusInternalServerError)
		return
	}
	ioCounters, err := net.IOCounters(false)
	if err != nil {
		write(w, err, http.StatusInternalServerError)
		return
	}
	status, err := p.Status()
	if err != nil {
		write(w, err, http.StatusInternalServerError)
		return
	}
	fds, err := fdCount(p.Pid)
	if err != nil {
		write(w, err, http.StatusInternalServerError)
		return
	}
	numThreads, err := p.NumThreads()
	if err != nil {
		write(w, err, http.StatusInternalServerError)
		return
	}
	memPercent, err := p.MemoryPercent()
	if err != nil {
		write(w, err, http.StatusInternalServerError)
		return
	}
	utc, err := p.CreateTime()
	if err != nil {
		write(w, err, http.StatusInternalServerError)
		return
	}
	write(w, resourceUsageResponse{
		Process: ProcessResourceUsage{
			Name:          name,
			Status:        status,
			CreateTime:    time.Unix(utc, 0).Format(time.RFC822),
			FDCount:       uint64(fds),
			ThreadCount:   uint64(numThreads),
			MemoryPercent: float64(memPercent),
			CPUPercent:    cpuPercent,
		},
		System: SystemResourceUsage{
			TotalRAM:        pm.Total,
			AvailableRAM:    pm.Available,
			UsedRAM:         pm.Used,
			UsedRAMPercent:  pm.UsedPercent,
			FreeRAM:         pm.Free,
			UsedCPUPercent:  cp[0],
			UserCPU:         c[0].User,
			SystemCPU:       c[0].System,
			IdleCPU:         c[0].Idle,
			TotalDisk:       d.Total,
			UsedDisk:        d.Used,
			UsedDiskPercent: d.UsedPercent,
			FreeDisk:        d.Free,
			ReceivedBytesIO: ioCounters[0].BytesRecv,
			WrittenBytesIO:  ioCounters[0].BytesSent,
		},
	}, http.StatusOK)
}
