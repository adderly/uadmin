package security

type securityProofService struct {
}

// create a proof to be solved
func (sp *securityProofService) Create() (error, bool) {
	return nil, false
}

func (sp securityProofService) Valid() {

}
