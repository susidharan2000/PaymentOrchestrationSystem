package eventprojector

type LedgerEntry struct {
	PaymentID string
	Seq       int64
	EntryType string
	Amount    int64
}

type State struct {
	LastAppliedSeq int64
	CapturedAmount int64
	RefundedAmount int64
}
