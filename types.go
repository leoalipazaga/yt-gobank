package main

import (
	cryptoRand "crypto/rand"
	"fmt"
	"math/rand"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Number   int64  `json:"number"`
	Password string `json:"password"`
}

type TransferRequest struct {
	FromAccount int `json:"fromAccount"`
	ToAccount   int `json:"toAccount"`
	Amount      int `json:"amount"`
}

type CreateAccountRequest struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Password  string `json:"password"`
}

type Account struct {
	ID        int       `json:"id"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	Password  []byte    `json:"-"`
	Number    int64     `json:"number"`
	Balance   int64     `json:"balance"`
	CreatedAt time.Time `json:"createdAt"`
}

func createSalt(size int) []byte {
	salt := make([]byte, size)
	_, err := cryptoRand.Read(salt[:])
	if err != nil {
		panic(err)
	}
	return salt
}

func hashPassword(password []byte) []byte {
	hashedPassword, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	if err != nil {
		fmt.Printf("there was an error on hashing password %v\n", err)
	}

	return hashedPassword
}

func NewAccount(firstName, lastName, password string) *Account {
	encryptedPassword := hashPassword([]byte(password))
	return &Account{
		FirstName: firstName,
		LastName:  lastName,
		Password:  encryptedPassword,
		Number:    int64(rand.Intn(1000000)),
		Balance:   0,
		CreatedAt: time.Now().UTC(),
	}
}

func (s *Account) MatchPasswords(password string) bool {
	if err := bcrypt.CompareHashAndPassword(s.Password, []byte(password)); err != nil {
		return false
	}

	return true
}
