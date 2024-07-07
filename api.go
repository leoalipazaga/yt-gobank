package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/golang-jwt/jwt/v5"
)

type apiFunc func(http.ResponseWriter, *http.Request) error

type ApiServer struct {
	listenAddr string
	store      Storage
}

type ApiError struct {
	Error string `json:"error"`
}

const secret = "secret key"

func WriteJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)

	return json.NewEncoder(w).Encode(v)
}

func createJWT(account *Account) (string, error) {
	claims := &jwt.MapClaims{
		"expireAt":      15000,
		"accountNumber": account.Number,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(secret))
}

func withAuth(handlerFunc http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenRaw := r.Header.Get("x-jwt-token")
		token, err := validateJWT(tokenRaw)
		if err != nil {
			WriteJSON(w, http.StatusForbidden, ApiError{Error: err.Error()})
			return
		}

		if !token.Valid {
			WriteJSON(w, http.StatusForbidden, ApiError{Error: "invalid token"})
			return
		}
		claims := token.Claims.(jwt.MapClaims)
		fmt.Println("claims ::>,", claims)
		handlerFunc(w, r)
	}
}

func validateJWT(tokenRaw string) (*jwt.Token, error) {
	return jwt.Parse(tokenRaw, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
}

func makeHTTPHandleFunc(f apiFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			WriteJSON(w, http.StatusBadRequest, ApiError{Error: err.Error()})
		}
	}
}

func NewApiServer(listenAddr string, store Storage) *ApiServer {
	return &ApiServer{
		listenAddr: listenAddr,
		store:      store,
	}
}

func (s *ApiServer) Run() {
	router := http.NewServeMux()
	router.HandleFunc("GET /account", makeHTTPHandleFunc(s.handleGetAccounts))
	router.HandleFunc("POST /account", makeHTTPHandleFunc(s.handleCreateAccount))
	router.HandleFunc("PUT /account/{id}", withAuth(makeHTTPHandleFunc(s.handleUpdateAccountById)))
	router.HandleFunc("DELETE /account/{id}", withAuth(makeHTTPHandleFunc(s.handleDeleteAccountById)))
	router.HandleFunc("GET /account/{id}", withAuth(makeHTTPHandleFunc(s.handleGetAccountByID)))
	router.HandleFunc("POST /transfer", makeHTTPHandleFunc(s.handleTransferAccount))
	router.HandleFunc("POST /login", makeHTTPHandleFunc(s.handleLogin))
	fmt.Println("JSON Api server running on:", s.listenAddr)

	http.ListenAndServe(s.listenAddr, router)
}

func (s *ApiServer) handleGetAccounts(w http.ResponseWriter, r *http.Request) error {
	accounts, err := s.store.GetAccounts()
	if err != nil {
		return err
	}

	return WriteJSON(w, http.StatusOK, accounts)
}

func (s *ApiServer) handleGetAccountByID(w http.ResponseWriter, r *http.Request) error {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return fmt.Errorf("invalid id given %s", s)
	}
	account, err := s.store.GetAccountByID(id)
	if err != nil {
		return err
	}

	return WriteJSON(w, http.StatusOK, account)
}

func (s *ApiServer) handleCreateAccount(w http.ResponseWriter, r *http.Request) error {
	accountRequest := new(CreateAccountRequest)
	if err := json.NewDecoder(r.Body).Decode(accountRequest); err != nil {
		return WriteJSON(w, http.StatusBadRequest, ApiError{Error: "invalid request"})
	}
	account := NewAccount(accountRequest.FirstName, accountRequest.LastName, accountRequest.Password)
	if err := s.store.CreateAccount(account); err != nil {
		fmt.Println("error", err)
		return WriteJSON(w, http.StatusBadRequest, ApiError{Error: "invalid request. take a look your fields"})
	}

	return WriteJSON(w, http.StatusOK, account)
}

func (s *ApiServer) handleUpdateAccountById(w http.ResponseWriter, r *http.Request) error {
	accountRequest := new(Account)
	id, err := strconv.Atoi(r.PathValue("id"))
	json.NewDecoder(r.Body).Decode(accountRequest)
	accountRequest.ID = id
	if err != nil {
		return err
	}
	err = s.store.UpdateAccount(accountRequest)
	if err != nil {
		return err
	}

	return WriteJSON(w, http.StatusOK, map[string]int{"id": id})
}

func (s *ApiServer) handleDeleteAccountById(w http.ResponseWriter, r *http.Request) error {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return fmt.Errorf("invalid id given %s", s)
	}

	if err := s.store.DeleteAccount(id); err != nil {
		return err
	}

	res := make(map[string]int)
	res["id"] = id

	return WriteJSON(w, http.StatusOK, res)
}

func (s *ApiServer) handleTransferAccount(w http.ResponseWriter, r *http.Request) error {
	transferRequest := new(TransferRequest)
	fromAccount, err := s.store.GetAccountByNumber(transferRequest.FromAccount)
	if err != nil {
		return err
	}

	if fromAccount.Balance < int64(transferRequest.Amount) {
		invalidAmountError := ApiError{Error: fmt.Errorf("invalid amount %d", transferRequest.Amount).Error()}
		return WriteJSON(w, http.StatusBadRequest, invalidAmountError)
	}

	if err := json.NewDecoder(r.Body).Decode(transferRequest); err != nil {
		return WriteJSON(w, http.StatusBadRequest, ApiError{Error: err.Error()})
	}

	if err := s.store.TransferAccount(transferRequest); err != nil {
		return WriteJSON(w, http.StatusBadRequest, ApiError{Error: err.Error()})
	}

	return WriteJSON(w, http.StatusOK, transferRequest)
}

func (s *ApiServer) handleLogin(w http.ResponseWriter, r *http.Request) error {
	req := new(LoginRequest)

	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		return err
	}

	account, err := s.store.GetAccountByNumber(int(req.Number))
	if err != nil {
		return err
	}
	if canAccess := account.MatchPasswords(req.Password); !canAccess {
		return WriteJSON(w, http.StatusUnauthorized, ApiError{Error: "Invaid password or account number"})
	}

	jwt, err := createJWT(account)
	if err != nil {
		return err
	}

	res := make(map[string]string)
	res["jwt"] = jwt

	return WriteJSON(w, http.StatusOK, res)
}
