package worker

type RefundDetails struct {
	refundID       string
	amount         int64
	PspName        string
	pspReferenceID string
}
