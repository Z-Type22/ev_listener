package models

type User struct {
	ID           int64
	Email        string
	Wallet       string
	PasswordHash []byte
}
