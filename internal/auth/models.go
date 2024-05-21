package auth

type UserDetails struct {
	Id       string `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Password []byte `json:"password"`
}

type RequestDetails struct {
	Id       string `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}
