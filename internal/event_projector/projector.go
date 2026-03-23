package eventprojector

import (
	"fmt"
	"log"
	"time"
)

func StartProjector(repo ProjectorRepository) {
	for {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Println("projector panic recovered:", r)
				}
			}()

			if err := processBatch(repo); err != nil {
				log.Println("projector error:", err)
			}
		}()

		time.Sleep(1 * time.Second)
	}
}

func processBatch(repo ProjectorRepository) (err error) {
	tx, err := repo.BeginTx()
	if err != nil {
		return err
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			err = fmt.Errorf("panic: %v", r)
			return
		}
		if err != nil {
			tx.Rollback()
		}
	}()
	//Lock offset
	lastSeq, err := repo.GetOffsetForUpdate(tx, "payment_projector")
	if err != nil {
		return err
	}
	//Fetch batch
	entries, err := repo.FetchLedger(tx, lastSeq, 100)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return tx.Commit()
	}
	//collect payment ID
	paymentSet := make(map[string]struct{})
	for _, e := range entries {
		paymentSet[e.PaymentID] = struct{}{}
	}

	paymentIDs := make([]string, 0, len(paymentSet))
	for id := range paymentSet {
		paymentIDs = append(paymentIDs, id)
	}
	//get status in bulk
	states, err := repo.GetPaymentStates(tx, paymentIDs)
	if err != nil {
		return err
	}
	originalStates := make(map[string]State, len(states))
	for k, v := range states {
		originalStates[k] = v
	}
	//mutate the state based on the ledger(streaming)
	for _, e := range entries {
		state := states[e.PaymentID]

		//idempotent check
		if e.Seq <= state.LastAppliedSeq {
			continue
		}

		switch e.EntryType {
		case "PAYMENT":
			state.CapturedAmount += e.Amount
		case "REFUND":
			state.RefundedAmount += e.Amount
		}

		state.LastAppliedSeq = e.Seq
		states[e.PaymentID] = state
	}

	for paymentID, state := range states {
		old := originalStates[paymentID]
		if state.CapturedAmount == old.CapturedAmount &&
			state.RefundedAmount == old.RefundedAmount {
			continue
		}
		newStatus := deriveStatus(state.CapturedAmount, state.RefundedAmount)
		if err := repo.UpdatePaymentState(
			tx,
			paymentID,
			state,
			newStatus,
		); err != nil {
			return err
		}
	}
	//Move offset(cursor)
	newSeq := entries[len(entries)-1].Seq

	if err = repo.UpdateOffset(tx, "payment_projector", newSeq); err != nil {
		return err
	}

	return tx.Commit()
}

func deriveStatus(captured int64, refunded int64) string {
	switch {
	case captured == 0:
		return "PROCESSING"
	case captured != 0 && refunded == 0:
		return "CAPTURED"
	case refunded > 0 && refunded < captured:
		return "PARTIALLY_REFUNDED"
	case refunded == captured:
		return "REFUNDED"
	default:
		return "INVALID"
	}
}
