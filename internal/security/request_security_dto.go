package security

type RequestProofDto struct {
}

// Dto fo receiving data for captcha from user input
type RequestProofInputDto struct {
	key   string `json:"key,omitempty" form:"key,omitempty"`
	proof string `json:"proof,omitempty" form:"proof,omitempty"`
}

type RequestProofOutSlideDto struct {
	code  string `json:"code,omitempty" form:"code,omitempty"`
	key   string `json:"key,omitempty" form:"key,omitempty"`
	image string `json:"image,omitempty" form:"image,omitempty"`
}
