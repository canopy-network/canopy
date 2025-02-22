package rpc

import (
	"time"

	"github.com/canopy-network/canopy/fsm/types"
	"github.com/canopy-network/canopy/lib"
)

// updatePollResults() updates the poll results based on the current token power
func (s *Server) updatePollResults() {
	for {
		p := new(types.ActivePolls)
		if err := func() (err error) {
			if err = p.NewFromFile(s.config.DataDirPath); err != nil {
				return
			}
			sm, err := s.controller.FSM.TimeMachine(0)
			if err != nil {
				return err
			}
			// cleanup old polls
			p.Cleanup(sm.Height())
			if err = p.SaveToFile(s.config.DataDirPath); err != nil {
				return
			}
			// convert the poll to a result
			result, err := sm.PollsToResults(p)
			if err != nil || len(result) == 0 {
				return
			}
			// update the rpc accessible version
			s.pollMux.Lock()
			s.poll = result
			s.pollMux.Unlock()
			return
		}(); err != nil {
			logger.Error(err.Error())
		}
		time.Sleep(time.Second * 3)
	}
}

// PollRootChainInfo() retrieves the information from the root-Chain required for consensus
func (s *Server) pollRootChainInfo() {
	var rootChainHeight uint64
	// execute the loop every conf.RootChainPollMS duration
	ticker := time.NewTicker(time.Duration(s.config.RootChainPollMS) * time.Millisecond)
	for range ticker.C {
		if err := func() (err error) {
			state, err := s.controller.FSM.TimeMachine(0)
			if err != nil {
				return
			}
			defer state.Discard()
			// get the consensus params from the app
			consParams, err := state.GetParamsCons()
			if err != nil {
				return
			}
			// get the url for the root chain as set by the state
			var rootChainUrl string
			for _, chain := range s.config.RootChain {
				if chain.ChainId == consParams.RootChainId {
					rootChainUrl = chain.Url
				}
			}
			// check if root chain url isn't empty
			if rootChainUrl == "" {
				logger.Errorf("Config.JSON missing RootChainID=%d failed with", consParams.RootChainId)
				return lib.ErrEmptyChainId()
			}
			// create a rpc client
			rpcClient := NewClient(rootChainUrl, "", "")
			// set the apps callbacks
			s.controller.RootChainInfo.RemoteCallbacks = &lib.RemoteCallbacks{
				Checkpoint:            rpcClient.Checkpoint,
				ValidatorSet:          rpcClient.ValidatorSet,
				IsValidDoubleSigner:   rpcClient.IsValidDoubleSigner,
				Transaction:           rpcClient.Transaction,
				LastProposers:         rpcClient.LastProposers,
				MinimumEvidenceHeight: rpcClient.MinimumEvidenceHeight,
				CommitteeData:         rpcClient.CommitteeData,
				Lottery:               rpcClient.Lottery,
				Orders:                rpcClient.Orders,
			}
			// query the base chain height
			height, err := rpcClient.Height()
			if err != nil {
				logger.Errorf("GetRootChainHeight failed with err")
				return err
			}
			// check if a new height was received
			if *height <= rootChainHeight {
				return
			}
			// update the base chain height
			rootChainHeight = *height
			// if a new height received
			logger.Infof("New RootChain height %d detected!", rootChainHeight)
			// execute the requests to get the base chain information
			for retry := lib.NewRetry(s.config.RootChainPollMS, 3); retry.WaitAndDoRetry(); {
				// retrieve the root-Chain info
				rootChainInfo, e := rpcClient.RootChainInfo(rootChainHeight, s.config.ChainId)
				if e == nil {
					// update the controller with new root-Chain info
					s.controller.UpdateRootChainInfo(rootChainInfo)
					logger.Info("Updated RootChain information")
					break
				}
				logger.Errorf("GetRootChainInfo failed with err %s", e.Error())
				// update with empty root-Chain info to stop consensus
				s.controller.UpdateRootChainInfo(&lib.RootChainInfo{
					Height:           rootChainHeight,
					ValidatorSet:     lib.ValidatorSet{},
					LastValidatorSet: lib.ValidatorSet{},
					LastProposers:    &lib.Proposers{},
					LotteryWinner:    &lib.LotteryWinner{},
					Orders:           &lib.OrderBook{},
				})
			}
			return
		}(); err != nil {
			logger.Warnf(err.Error())
		}
	}
}
