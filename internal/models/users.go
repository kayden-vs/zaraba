package models

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type UserModelInterface interface {
	Insert(name, email, password string) (int, error)
	Authenticate(email, password string) (int, error)
	Exists(id int) (bool, error)
	GetUserInfo(id int) (*User, error)
	ComparePassword(id int, password string) error
	UpdatePassword(id int, password string) error
}

type User struct {
	ID             int
	Name           string
	Email          string
	HashedPassword []byte
	Created        time.Time
}

type UserModel struct {
	DB *sql.DB
}

func (m *UserModel) Insert(name, email, password string) (int, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return 0, err
	}

	var id int

	stmt := `INSERT INTO users (name, email, hashed_password, created)
    VALUES($1, $2, $3, NOW()) RETURNING id`

	err = m.DB.QueryRow(stmt, name, email, hashedPassword).Scan(&id)
	if err != nil {
		// Checking if the error is for duplicate email
		var pgError *pq.Error
		if errors.As(err, &pgError) {
			if pgError.Code == "23505" && strings.Contains(pgError.Constraint, "users_uc_email") {
				return 0, ErrDuplicateEmail
			}
		}
		return 0, err
	}

	return id, nil
}

func (m *UserModel) Authenticate(email, password string) (int, error) {
	var id int
	var hashedPassword []byte

	stmt := "SELECT id, hashed_password FROM users WHERE email = $1"

	// Check if the email exists in db
	err := m.DB.QueryRow(stmt, email).Scan(&id, &hashedPassword)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrInvalidCredentials
		} else {
			return 0, err
		}
	}

	// Compare passwords
	err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return 0, ErrInvalidCredentials
		} else {
			return 0, err
		}
	}

	return id, nil
}

func (m *UserModel) Exists(id int) (bool, error) {
	var exists bool

	stmt := "SELECT EXISTS(SELECT true FROM users WHERE id = $1)"
	err := m.DB.QueryRow(stmt, id).Scan(&exists)

	return exists, err
}

func (m *UserModel) GetUserInfo(id int) (*User, error) {
	user := &User{}

	stmt := "SELECT id,name,email,created FROM users WHERE id = $1"

	err := m.DB.QueryRow(stmt, id).Scan(&user.ID, &user.Name, &user.Email, &user.Created)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoRecord
		}
		return nil, err
	}

	return user, nil
}

func (m *UserModel) ComparePassword(id int, password string) error {
	stmt := `SELECT hashed_password FROM users WHERE id = $1`

	var hashedPassword []byte

	err := m.DB.QueryRow(stmt, id).Scan(&hashedPassword)
	if err != nil {
		return err
	}
	err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return ErrInvalidCredentials
		} else {
			return err
		}
	}
	return nil
}

func (m *UserModel) UpdatePassword(id int, password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return err
	}

	stmt := `UPDATE users SET hashed_password = $1 WHERE id = $2`

	_, err = m.DB.Exec(stmt, hashedPassword, id)
	if err != nil {
		return err
	}

	return nil
}
