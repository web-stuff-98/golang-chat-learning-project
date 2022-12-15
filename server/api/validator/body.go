package validator

type Credentials struct {
	Password string `json:"password" validate:"required"`
	Username string `json:"username" validate:"required"`
}

type Room struct {
	Name string `json:"name"`
}
