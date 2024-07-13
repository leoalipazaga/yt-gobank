package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

type Storage interface {
	CreateAccount(*Account) error
	DeleteAccount(int) error
	UpdateAccount(*Account) error
	GetAccountByID(int) (*Account, error)
	GetAccountByNumber(int) (*Account, error)
	GetAccounts() ([]*Account, error)
	TransferAccount(*TransferRequest) error
}

type PostgresStore struct {
	db *sql.DB
}

func (s *PostgresStore) Init() error {
	err := s.createAccountTable()
	return err
}

func (s *PostgresStore) createAccountTable() error {
	query := `CREATE TABLE IF NOT EXISTS account(
    id serial primary key,
    first_name varchar(50),
    last_name varchar(50),
    password bytea,
    number serial unique,
    balance serial,
    created_at timestamp
  )`
	_, err := s.db.Exec(query)
	return err
}

func NewPostgresStore() (*PostgresStore, error) {
	conn := fmt.Sprintf("host=db user=%s dbname=%s password=%s sslmode=disable",
		os.Getenv("DB_USER"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PASSWORD"),
	)
	db, err := sql.Open("postgres", conn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &PostgresStore{
		db: db,
	}, nil
}

func (s *PostgresStore) CreateAccount(account *Account) error {
	_, err := s.db.Exec(`
    insert into account(first_name, last_name, password, number, balance, created_at)
    values($1, $2, $3, $4, $5, $6)`,
		account.FirstName, account.LastName, account.Password, account.Number, account.Balance, account.CreatedAt,
	)

	return err
}

func (s *PostgresStore) DeleteAccount(id int) error {
	query := `delete from account where id = $1`
	rows, err := s.db.Exec(query, id)
	if err != nil {
		return err
	}
	counts, err := rows.RowsAffected()
	if err != nil {
		return err
	}
	if counts == 0 {
		return fmt.Errorf("id %d not found", id)
	}

	return nil
}

func (s *PostgresStore) UpdateAccount(account *Account) error {
	query := `update account set first_name = $1, last_name = $2, number = $3, balance = $4 where id = $5`
	rows, err := s.db.Exec(query, account.FirstName, account.LastName, account.Number, account.Balance, account.ID)
	if err != nil {
		return err
	}
	count, err := rows.RowsAffected()
	if err != nil {
		return err
	}
	fmt.Printf("rows deleted %d", count)

	return err
}

func (s *PostgresStore) GetAccountByID(id int) (*Account, error) {
	query := `select id, first_name, last_name, number, balance, password, created_at from account where id = $1`
	rows, err := s.db.Query(query, id)
	if err != nil {
		return nil, err
	}
	accounts, err := scanAccounts(rows)
	if err != nil {
		return nil, err
	}

	if len(accounts) == 0 {
		return nil, fmt.Errorf("account with id %d not found", id)
	}

	return accounts[0], nil
}

func (s *PostgresStore) GetAccountByNumber(number int) (*Account, error) {
	query := `select id, first_name, last_name, number, balance, password, created_at from account where number = $1`
	rows, err := s.db.Query(query, number)
	if err != nil {
		return nil, err
	}

	account, err := scanAccounts(rows)
	if err != nil {
		return nil, err
	}
	if len(account) == 0 {
		return nil, fmt.Errorf("account with number %d not found", number)
	}

	return account[0], nil
}

func (s *PostgresStore) GetAccounts() ([]*Account, error) {
	query := `select id, first_name, last_name, number, balance, password, created_at from account`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	return scanAccounts(rows)
}

func (s *PostgresStore) TransferAccount(transferRequest *TransferRequest) error {
	fromAccountQuery := `update account set balance = (account.balance - $1) where number = $2`
	toAccountQuery := `update account set balance = (account.balance + $1) where number = $2`
	if _, err := s.db.Exec(fromAccountQuery, transferRequest.Amount, transferRequest.FromAccount); err != nil {
		return err
	}
	if _, err := s.db.Exec(toAccountQuery, transferRequest.Amount, transferRequest.ToAccount); err != nil {
		return err
	}

	return nil
}

func scanAccounts(rows *sql.Rows) ([]*Account, error) {
	accounts := []*Account{}
	for rows.Next() {
		row := Account{}
		err := rows.Scan(&row.ID, &row.FirstName, &row.LastName, &row.Number, &row.Balance, &row.Password, &row.CreatedAt)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, &row)
	}

	return accounts, nil
}
