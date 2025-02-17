package security

type RequestProofDto struct {
}

// Dto fo receiving data for captcha from user input
type RequestProofInputDto struct {
	key   string `json:"key,omitempty" form:"key,omitempty"`
	proof string `json:"proof,omitempty" form:"proof,omitempty"`
}

type RequestProofOutSlideDto struct {
	code   int    `json:"code,omitempty" form:"code,omitempty"`
	key    string `json:"key,omitempty" form:"key,omitempty"`
	image  string `json:"image,omitempty" form:"image,omitempty"`
	tile   string `json:"tile,omitempty" form:"tile,omitempty"`
	tile_w int    `json:"tile_w,omitempty" form:"tile_w,omitempty"`
	tile_h int    `json:"tile_h,omitempty" form:"tile_h,omitempty"`
	tile_x int    `json:"tile_x,omitempty" form:"tile_x,omitempty"`
	tile_y int    `json:"tile_y,omitempty" form:"tile_y,omitempty"`
}

type RequestProofOutRotateDto struct {
	code  int    `json:"code,omitempty" form:"code,omitempty"`
	key   string `json:"key,omitempty" form:"key,omitempty"`
	image string `json:"image,omitempty" form:"image,omitempty"`
	thumb string `json:"thumb,omitempty" form:"thumb,omitempty"`
}

type RequestProofOutClickDto struct {
	code  int    `json:"code,omitempty" form:"code,omitempty"`
	key   string `json:"key,omitempty" form:"key,omitempty"`
	image string `json:"image,omitempty" form:"image,omitempty"`
	thumb string `json:"thumb,omitempty" form:"thumb,omitempty"`
}
