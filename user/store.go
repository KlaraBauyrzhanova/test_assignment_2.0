package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

// store is struct of database
type store struct {
	DB *sqlx.DB
}

// NewStore creates store struct
func NewStore(db *sqlx.DB) *store {
	return &store{
		DB: db,
	}
}

// GetUserByID selects user by ID
func (s *store) GetUserByID(id int) (User, error) {
	u := User{ID: id}
	user := &User{}

	row, err := s.DB.NamedQuery(`SELECT * FROM users WHERE id=:id`, u)
	if err != nil {
		return User{}, err
	}
	if !row.Next() {
		return User{}, errors.New("failed to get user by id")
	}

	err = row.StructScan(&user)
	if err != nil {
		return User{}, err
	}

	return *user, nil
}

// SaveUserByID creates new user by ID
func (s *store) SaveUserByID(u User, str string) error {
	_, err := s.DB.NamedExec(`INSERT INTO users(id, data) VALUES(:id, :data)`,
		map[string]interface{}{
			"id":   u.ID,
			"data": str,
		})
	if err != nil {
		return err
	}
	return nil
}

// UpdateUserByID updates user by ID
func (s *store) UpdateUserByID(id int, field, value string) (user *User, err error) {
	trn, err := s.DB.BeginTxx(context.Background(), nil)
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			trn.Rollback()
			return
		}
		err = trn.Commit()
	}()

	u := User{ID: id}
	row, err := trn.NamedQuery(`SELECT * FROM users WHERE id=:id FOR UPDATE`, u)
	if err != nil {
		return
	}
	defer row.Close()

	if !row.Next() {
		return
	}
	var us User
	err = row.StructScan(&us)
	if err != nil {
		return
	}

	if row.Next() {
		return nil, errors.New("id not unique")
	}

	user = &us

	b := []byte(user.Data)
	var d Data
	err = json.Unmarshal(b, &d)
	fmt.Println(d.FirstName)
	if err != nil {
		return
	}
	// v := value[1 : len(value)-1]
	switch field {
	case "first_name":
		d.FirstName = value
	case "last_name":
		d.LastName = value
	case " interest":
		if d.Interests == "" {
			d.Interests = value
		} else {
			d.Interests = d.Interests + "," + value
		}
	case "-interest":
		if strings.Contains(d.Interests, value) {
			if d.Interests == value {
				d.Interests = ""
			} else {
				arr := strings.Split(d.Interests, ",")
				s := ""
				for i := 0; i < len(arr); i++ {
					if arr[i] == value {
						continue
					}
					if i != len(arr) {
						s += arr[i] + ","
					}
				}
				if len(s) > 0 && s[len(s)-1:] == "," {
					d.Interests = s[:len(s)-1]
				}
			}
		}
	default:
		return nil, errors.New("no such field")
	}

	str, err := json.Marshal(d)
	if err != nil {
		return
	}

	dd := map[string]interface{}{
		"id":   id,
		"data": string(str),
	}
	_, err = trn.NamedExec(`UPDATE users SET data=:data WHERE id=:id`, dd)
	if err != nil {
		return
	}
	user = &User{
		ID:   id,
		Data: string(str),
	}

	return
}
