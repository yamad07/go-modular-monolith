package adapter

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
