package psp

// Registry
type Registry struct {
	PaymentServiceProviders map[string]PSP
}

func NewRegistry() *Registry {
	return &Registry{
		PaymentServiceProviders: make(map[string]PSP),
	}
}
func (r *Registry) Register(name string, psp PSP) {
	r.PaymentServiceProviders[name] = psp
}

func (r *Registry) Get(name string) (PSP, bool) {
	psp, ok := r.PaymentServiceProviders[name]
	return psp, ok
}
