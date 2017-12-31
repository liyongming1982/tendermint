package types

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	crypto "github.com/tendermint/go-crypto"
	data "github.com/tendermint/go-wire/data"
	"github.com/tendermint/tendermint/types"
)

//-------------------------------------

// LastSignedInfo contains information about the latest
// data signed by a validator to help prevent double signing.
type LastSignedInfo struct {
	LastHeight    int64            `json:"last_height"`
	LastRound     int              `json:"last_round"`
	LastStep      int8             `json:"last_step"`
	LastSignature crypto.Signature `json:"last_signature,omitempty"` // so we dont lose signatures
	LastSignBytes data.Bytes       `json:"last_signbytes,omitempty"` // so we dont lose signatures
}

func NewLastSignedInfo() *LastSignedInfo {
	return &LastSignedInfo{
		LastStep: -1,
	}
}

func (info *LastSignedInfo) String() string {
	return fmt.Sprintf("LH:%v, LR:%v, LS:%v", info.LastHeight, info.LastRound, info.LastStep)
}

// Verify returns an error if there is a height/round/step regression
// or if the HRS matches but there are no LastSignBytes.
// It returns true if HRS matches exactly and the LastSignature exists.
// It panics if the HRS matches, the LastSignBytes are not empty, but the LastSignature is empty.
func (info LastSignedInfo) Verify(height int64, round int, step int8) (bool, error) {
	if info.LastHeight > height {
		return false, errors.New("Height regression")
	}

	if info.LastHeight == height {
		if info.LastRound > round {
			return false, errors.New("Round regression")
		}

		if info.LastRound == round {
			if info.LastStep > step {
				return false, errors.New("Step regression")
			} else if info.LastStep == step {
				if info.LastSignBytes != nil {
					if info.LastSignature.Empty() {
						panic("info: LastSignature is nil but LastSignBytes is not!")
					}
					return true, nil
				}
				return false, errors.New("No LastSignature found")
			}
		}
	}
	return false, nil
}

// Set height/round/step and signature on the info
func (info *LastSignedInfo) Set(height int64, round int, step int8,
	signBytes []byte, sig crypto.Signature) {

	info.LastHeight = height
	info.LastRound = round
	info.LastStep = step
	info.LastSignature = sig
	info.LastSignBytes = signBytes
}

func (info *LastSignedInfo) Reset() {
	info.LastHeight = 0
	info.LastRound = 0
	info.LastStep = 0
	info.LastSignature = crypto.Signature{}
	info.LastSignBytes = nil
}

//-------------------------------------

type checkOnlyDifferByTimestamp func([]byte, []byte) bool

// returns true if the only difference in the votes is their timestamp
func checkVotesOnlyDifferByTimestamp(lastSignBytes, newSignBytes []byte) bool {
	var lastVote, newVote types.CanonicalJSONOnceVote
	if err := json.Unmarshal(lastSignBytes, &lastVote); err != nil {
		panic(fmt.Sprintf("LastSignBytes cannot be unmarshalled into vote: %v", err))
	}
	if err := json.Unmarshal(newSignBytes, &newVote); err != nil {
		panic(fmt.Sprintf("signBytes cannot be unmarshalled into vote: %v", err))
	}

	// set the times to the same value and check equality
	now := types.CanonicalTime(time.Now())
	lastVote.Vote.Timestamp = now
	newVote.Vote.Timestamp = now
	lastVoteBytes, _ := json.Marshal(lastVote)
	newVoteBytes, _ := json.Marshal(newVote)

	return bytes.Equal(newVoteBytes, lastVoteBytes)
}

// returns true if the only difference in the proposals is their timestamp
func checkProposalsOnlyDifferByTimestamp(lastSignBytes, newSignBytes []byte) bool {
	var lastProposal, newProposal types.CanonicalJSONOnceProposal
	if err := json.Unmarshal(lastSignBytes, &lastProposal); err != nil {
		panic(fmt.Sprintf("LastSignBytes cannot be unmarshalled into proposal: %v", err))
	}
	if err := json.Unmarshal(newSignBytes, &newProposal); err != nil {
		panic(fmt.Sprintf("signBytes cannot be unmarshalled into proposal: %v", err))
	}

	// set the times to the same value and check equality
	now := types.CanonicalTime(time.Now())
	lastProposal.Proposal.Timestamp = now
	newProposal.Proposal.Timestamp = now
	lastProposalBytes, _ := json.Marshal(lastProposal)
	newProposalBytes, _ := json.Marshal(newProposal)

	return bytes.Equal(newProposalBytes, lastProposalBytes)
}
