package invariants

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// FuzzReferenceModel drives the reference model (T3/T4/T5) with a random
// sequence of operations decoded from fuzz bytes. After every successful
// transition the snapshot must satisfy CheckAll. A failing seed indicates
// either (a) a reference-model bug, (b) a predicate bug, or (c) a SPEC
// inconsistency — any of which is a real finding.
//
// Run with:
//   go test -run=^$ -fuzz=FuzzReferenceModel -fuzztime=30s ./invariants/
//
// The byte stream is decoded as a sequence of ops, each framed as:
//   opByte: 0 -> Advance(new hash byte, has-move)
//           1 -> Invalidate(hash byte)
//           2 -> Rewind
//           3..255 -> no-op (skip)
// Any out-of-bound consumption ends the sequence.
func FuzzReferenceModel(f *testing.F) {
	// Seeds chosen to exercise T5, T4, T3, and mixed sequences.
	f.Add([]byte{0x00, 0x01}) // single advance
	f.Add([]byte{0x00, 0x01, 0x00, 0x02, 0x00, 0x03})          // three advances
	f.Add([]byte{0x00, 0x01, 0x00, 0x02, 0x02})                // advance x2, rewind
	f.Add([]byte{0x00, 0x01, 0x01, 0x99, 0x02})                // advance, invalidate, rewind
	f.Add([]byte{0x00, 0x01, 0x00, 0x02, 0x01, 0x99, 0x00, 0x03})

	f.Fuzz(func(t *testing.T, ops []byte) {
		// Cap the number of operations processed to keep per-seed memory
		// and runtime bounded. Without this, a pathological input of 10k+
		// Advance bytes can OOM the fuzz worker before any assertion is
		// evaluated.
		const maxOps = 512
		if len(ops) > maxOps*2 {
			ops = ops[:maxOps*2]
		}

		s := NewInitialSnapshot(100, []eth.ChainID{chainA()})

		// Track a monotonic block-number counter so Advance always produces
		// a properly-chained successor. The "new hash byte" from the fuzz
		// input is the identity of the new block; the parent is whatever
		// the current LogsDB tail is.
		var nextNum uint64 = 1
		var currentTime uint64 = 100
		prevHashByte := byte(0) // genesis parent hash

		i := 0
		applied := 0
		for i < len(ops) && applied < maxOps {
			applied++
			op := ops[i]
			i++
			switch op {
			case 0: // Advance
				if i >= len(ops) {
					return
				}
				newHashByte := ops[i]
				i++

				newL1 := blockID(byte(nextNum%255), 1000+nextNum)
				newBlock := block(newHashByte, prevHashByte, nextNum, currentTime)

				next, err := ApplyAdvance(s, newL1,
					map[eth.ChainID]BlockWithLogs{chainA(): newBlock})
				if err != nil {
					// Overflow or degenerate input; stop exploration.
					return
				}
				if inv := CheckAll(next); inv != nil {
					t.Fatalf("invariants violated after advance hash=%d: %v", newHashByte, inv)
				}
				s = next
				nextNum++
				currentTime++
				prevHashByte = newHashByte

			case 1: // Invalidate
				if i >= len(ops) {
					return
				}
				invalidHashByte := ops[i]
				i++
				if len(s.Verified) == 0 {
					continue // invalid; would error
				}
				next, err := ApplyInvalidate(s, map[eth.ChainID]eth.BlockID{
					chainA(): blockID(invalidHashByte, nextNum),
				})
				if err != nil {
					return
				}
				if inv := CheckAll(next); inv != nil {
					t.Fatalf("invariants violated after invalidate hash=%d: %v", invalidHashByte, inv)
				}
				s = next

			case 2: // Rewind
				if len(s.Verified) == 0 {
					continue
				}
				next, err := ApplyRewind(s)
				if err != nil {
					return
				}
				if inv := CheckAll(next); inv != nil {
					t.Fatalf("invariants violated after rewind: %v", inv)
				}
				// After rewind, the LogsDB tail may have been popped.
				// Re-sync the fuzzer's local counters.
				logs := next.LogsDB[chainA()]
				if len(logs) == 0 {
					nextNum = 1
					currentTime = 100
					prevHashByte = 0
				} else {
					tail := logs[len(logs)-1]
					nextNum = tail.Ref.ID.Number + 1
					currentTime = tail.Ref.Time + 1
					// Recover prevHashByte from the tail ID (last byte).
					prevHashByte = tail.Ref.ID.Hash[31]
				}
				s = next

			default:
				// skip / no-op
			}
		}
	})
}

// FuzzReferenceModelMultiChain drives the reference model with TWO L2
// chains advancing in lockstep, with the second chain's blocks carrying
// cross-chain executing messages that reference the first chain's most
// recent block. Exercises:
//
//   - I3 state-local (initiating-message presence on a different chain)
//   - I3 acyclicity (the iterative DFS over the cross-chain exec graph)
//   - I9 (LogsDB shape across multiple chains; max-derive plumbing)
//   - I6 (per-chain monotone heads while another chain is also moving)
//
// The single-chain fuzzer above caps these paths at zero; this one
// closes that gap. Operations are framed identically (op byte + arg
// byte) so the corpora are independent.
func FuzzReferenceModelMultiChain(f *testing.F) {
	f.Add([]byte{0x00, 0x01})
	f.Add([]byte{0x00, 0x01, 0x00, 0x02, 0x00, 0x03})
	f.Add([]byte{0x00, 0x01, 0x00, 0x02, 0x02})
	f.Add([]byte{0x00, 0x01, 0x01, 0x99, 0x02})
	f.Add([]byte{0x00, 0x01, 0x00, 0x02, 0x01, 0x99, 0x00, 0x03})

	f.Fuzz(func(t *testing.T, ops []byte) {
		const maxOps = 256 // halved vs single-chain: 2x state per op
		if len(ops) > maxOps*2 {
			ops = ops[:maxOps*2]
		}

		chains := []eth.ChainID{chainA(), chainB()}
		s := NewInitialSnapshot(100, chains)

		// Per-chain counters. Both chains advance together at each
		// Advance op so timestamps stay aligned.
		var nextNum uint64 = 1
		var currentTime uint64 = 100
		// Use distinct hash-byte spaces per chain so blocks never alias.
		// Chain A uses (newHashByte | 0x00); chain B uses (0x80).
		const chainBHashBit = 0x80
		prevHashByteA := byte(0)
		prevHashByteB := byte(0)

		i := 0
		applied := 0
		for i < len(ops) && applied < maxOps {
			applied++
			op := ops[i]
			i++
			switch op {
			case 0: // Advance both chains in lockstep
				if i >= len(ops) {
					return
				}
				newHashByte := ops[i]
				i++

				newL1 := blockID(byte(nextNum%127)+1, 1000+nextNum)

				blockA := block(newHashByte&0x7f, prevHashByteA, nextNum, currentTime)
				blockB := block(newHashByte|chainBHashBit, prevHashByteB, nextNum, currentTime)

				// Cross-chain executing message: chain B's new block
				// references chain A's PREVIOUS block (strictly older
				// timestamp). Skipped on the very first advance because
				// chain A has no prior block to point at.
				if nextNum > 1 {
					blockB.ExecMsgs = []ExecutingMessage{{
						ChainID:   chainA(),
						BlockNum:  nextNum - 1,
						LogIdx:    0,
						Timestamp: currentTime - 1,
					}}
				}

				next, err := ApplyAdvance(s, newL1,
					map[eth.ChainID]BlockWithLogs{
						chainA(): blockA,
						chainB(): blockB,
					})
				if err != nil {
					return
				}
				if inv := CheckAll(next); inv != nil {
					t.Fatalf("invariants violated after multichain advance hash=%d: %v", newHashByte, inv)
				}
				// State-local I3 must hold given the cross-chain exec
				// message just inserted.
				if i3 := CheckI3_ExecutingMessageValidity(next, nil); i3 != nil {
					t.Fatalf("I3 violated after multichain advance: %v", i3)
				}
				s = next
				nextNum++
				currentTime++
				prevHashByteA = newHashByte & 0x7f
				prevHashByteB = newHashByte | chainBHashBit

			case 1: // Invalidate chain B's current block
				if i >= len(ops) {
					return
				}
				invalidHashByte := ops[i]
				i++
				if len(s.Verified) == 0 {
					continue
				}
				next, err := ApplyInvalidate(s, map[eth.ChainID]eth.BlockID{
					chainB(): blockID(invalidHashByte|chainBHashBit, nextNum),
				})
				if err != nil {
					return
				}
				if inv := CheckAll(next); inv != nil {
					t.Fatalf("invariants violated after multichain invalidate hash=%d: %v", invalidHashByte, inv)
				}
				s = next

			case 2: // Rewind
				if len(s.Verified) == 0 {
					continue
				}
				next, err := ApplyRewind(s)
				if err != nil {
					return
				}
				if inv := CheckAll(next); inv != nil {
					t.Fatalf("invariants violated after multichain rewind: %v", inv)
				}
				// Re-sync local counters from chain A's tail (both
				// chains advance in lockstep so their lengths match).
				logs := next.LogsDB[chainA()]
				if len(logs) == 0 {
					nextNum = 1
					currentTime = 100
					prevHashByteA = 0
					prevHashByteB = 0
				} else {
					tailA := logs[len(logs)-1]
					nextNum = tailA.Ref.ID.Number + 1
					currentTime = tailA.Ref.Time + 1
					prevHashByteA = tailA.Ref.ID.Hash[31]
					if logsB := next.LogsDB[chainB()]; len(logsB) > 0 {
						prevHashByteB = logsB[len(logsB)-1].Ref.ID.Hash[31]
					} else {
						prevHashByteB = 0
					}
				}
				s = next

			default:
				// skip
			}
		}
	})
}

// Property test: seeded RNG equivalent that runs a small deterministic
// sample without relying on Go's fuzz infrastructure. Useful in CI that
// doesn't run -fuzz.
func TestReferenceModel_PropertySample(t *testing.T) {
	seeds := [][]byte{
		{0x00, 0x01},
		{0x00, 0x01, 0x00, 0x02, 0x00, 0x03, 0x00, 0x04},
		{0x00, 0x01, 0x00, 0x02, 0x02},
		{0x00, 0x01, 0x01, 0x99, 0x02},
		{0x00, 0x01, 0x00, 0x02, 0x01, 0x99, 0x00, 0x03},
		{0x00, 0x01, 0x00, 0x02, 0x02, 0x00, 0x03, 0x02, 0x00, 0x04},
	}
	for si, seed := range seeds {
		si := si
		seed := seed
		t.Run("", func(t *testing.T) {
			s := NewInitialSnapshot(100, []eth.ChainID{chainA()})
			var nextNum uint64 = 1
			var currentTime uint64 = 100
			prevHashByte := byte(0)
			for i := 0; i < len(seed); {
				op := seed[i]
				i++
				switch op {
				case 0:
					if i >= len(seed) {
						break
					}
					newHashByte := seed[i]
					i++
					newL1 := blockID(byte(nextNum%255), 1000+nextNum)
					newBlock := block(newHashByte, prevHashByte, nextNum, currentTime)
					next, err := ApplyAdvance(s, newL1,
						map[eth.ChainID]BlockWithLogs{chainA(): newBlock})
					if err != nil {
						return
					}
					if inv := CheckAll(next); inv != nil {
						t.Fatalf("seed=%d: %v", si, inv)
					}
					s = next
					nextNum++
					currentTime++
					prevHashByte = newHashByte
				case 1:
					if i >= len(seed) || len(s.Verified) == 0 {
						break
					}
					invalidHashByte := seed[i]
					i++
					next, err := ApplyInvalidate(s, map[eth.ChainID]eth.BlockID{
						chainA(): blockID(invalidHashByte, nextNum),
					})
					if err != nil {
						return
					}
					if inv := CheckAll(next); inv != nil {
						t.Fatalf("seed=%d: %v", si, inv)
					}
					s = next
				case 2:
					if len(s.Verified) == 0 {
						break
					}
					next, err := ApplyRewind(s)
					if err != nil {
						return
					}
					if inv := CheckAll(next); inv != nil {
						t.Fatalf("seed=%d: %v", si, inv)
					}
					logs := next.LogsDB[chainA()]
					if len(logs) == 0 {
						nextNum = 1
						currentTime = 100
						prevHashByte = 0
					} else {
						tail := logs[len(logs)-1]
						nextNum = tail.Ref.ID.Number + 1
						currentTime = tail.Ref.Time + 1
						prevHashByte = tail.Ref.ID.Hash[31]
					}
					s = next
				}
			}
		})
	}
}
