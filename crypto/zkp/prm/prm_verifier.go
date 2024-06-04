package zkpprm

import (
	"errors"

	big "github.com/binance-chain/tss-lib/common/int"
)

type PrmProofVerifier struct {
	semaphore chan interface{}
}

func NewPrmProofVerifier(concurrency int) (*PrmProofVerifier, error) {
	if concurrency == 0 {
		return nil, errors.New("NewPrmProofVerifier: concurrency level must not be zero")
	}

	semaphore := make(chan interface{}, concurrency)

	return &PrmProofVerifier{
		semaphore: semaphore,
	}, nil
}

func (pv *PrmProofVerifier) VerifyWithNonce(pf *ProofPrm, s *big.Int, t *big.Int, N *big.Int, nonce *big.Int, onDone func(bool)) {
	pv.semaphore <- struct{}{}
	go func() {
		defer func() { <-pv.semaphore }()

		onDone(pf.VerifyWithNonce(s, t, N, nonce))
	}()
}
