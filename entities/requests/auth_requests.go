package requests

// SignupRequest represents the signup request payload
type SignupRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Name     string `json:"name" binding:"required"`
}

// VerifyTokenRequest represents the verify token request payload
type VerifyTokenRequest struct {
	IDToken string `json:"id_token" binding:"required"`
}
