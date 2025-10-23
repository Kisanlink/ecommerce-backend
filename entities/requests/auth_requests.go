package requests

// SignupRequest represents the signup request payload
type SignupRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Name     string `json:"name" binding:"required"`
}

// LoginRequest represents the login request payload
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// VerifyTokenRequest represents the verify token request payload
type VerifyTokenRequest struct {
	IDToken string `json:"id_token" binding:"required"`
}

// Phone Signup and Login Request structures
type PhoneSignupRequest struct {
	PhoneNumber string `json:"phone_number" binding:"required"`
	Name        string `json:"name" binding:"required"`
}

type PhoneLoginRequest struct {
	PhoneNumber string `json:"phone_number" binding:"required"`
}

type SendPhoneCodeRequest struct {
	PhoneNumber    string `json:"phone_number" binding:"required"`
	RecaptchaToken string `json:"recaptcha_token" binding:"required"`
}

type VerifyPhoneCodeRequest struct {
	PhoneNumber string `json:"phone_number" binding:"required"`
	Code        string `json:"code" binding:"required"`
	SessionInfo string `json:"session_info" binding:"required"`
}
