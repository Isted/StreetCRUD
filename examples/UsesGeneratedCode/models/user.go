package models

import (
	"database/sql"
	"encoding/json"
	_ "github.com/lib/pq"
	"github.com/markbates/going/nulls"
	"log"
)

//Global Data Layer
var userSQL UserDataLayer

const (
	EXISTS  = iota
	DELETED = iota
	ALL     = iota
)

type User struct {
	LoginID     int              `json:"loginid"`
	Name        string           `json:"name, omitempty"`
	Email       nulls.NullString `json:"email”`
	Password    string           `json:"password" out:"false"`
	DeletedUser bool             `json:”deleted”`
	DelOn       nulls.NullTime   `json:”deletedon”`
}

//Initialize and fill a User object from the DB
func NewUser(loginID int, delFilter int) (*User, error) {
	user := new(User)
	deleted1 := false
	deleted2 := false
	switch delFilter {
	case DELETED:
		deleted1 = true
		deleted2 = true
	case ALL:
		deleted2 = true
	}
	row := userSQL.GetByID.QueryRow(loginID, deleted1, deleted2)
	err := row.Scan(&user.LoginID, &user.Name, &user.Email, &user.Password, &user.DeletedUser, &user.DelOn)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	return user, nil
}

//Transform JSON into a User object
func UserFromJSON(userJSON []byte) (*User, error) {
	user := new(User)
	err := json.Unmarshal(userJSON, user)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	return user, nil
}

//Convert a User object to JSON
func (user *User) ToJSON() ([]byte, error) {
	userJSON, err := json.Marshal(user)
	return userJSON, err
}

//Convert multiple User objects to JSON
func UsersToJSON(users []*User) ([]byte, error) {
	usersJSON, err := json.Marshal(users)
	return usersJSON, err
}

//Fill User object with data from DB
func (user *User) GetByID(loginID int, delFilter int) error {
	deleted1 := false
	deleted2 := false
	switch delFilter {
	case DELETED:
		deleted1 = true
		deleted2 = true
	case ALL:
		deleted2 = true
	}
	row := userSQL.GetByID.QueryRow(loginID, deleted1, deleted2)
	err := row.Scan(&user.LoginID, &user.Name, &user.Email, &user.Password, &user.DeletedUser, &user.DelOn)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	return nil
}

//Insert User object to DB
func (user *User) Insert() error {
	var id int
	row := userSQL.Insert.QueryRow(user.Name, user.Email, user.Password, user.DeletedUser, user.DelOn)
	err := row.Scan(&id)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	user.LoginID = id
	return nil
}

//Update User object in DB
func (user *User) Update() error {
	_, err := userSQL.Update.Exec(user.Name, user.Email, user.Password, user.DeletedUser, user.DelOn, user.LoginID)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	return nil
}

//Mark a row as deleted and at time.Time
func (user *User) MarkDeleted(del bool, when nulls.NullTime) error {
	_, err := userSQL.MarkDel.Exec(del, when, user.LoginID)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	user.DeletedUser = del
	user.DelOn = when
	return nil
}

//Delete will remove the matching row from the DB
func (user *User) Delete() error {
	_, err := userSQL.Delete.Exec(user.LoginID)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	return nil
}

//Get Users by name
func GetUsersByName(name string, delFilter int) ([]*User, error) {
	deleted1 := false
	deleted2 := false
	switch delFilter {
	case DELETED:
		deleted1 = true
		deleted2 = true
	case ALL:
		deleted2 = true
	}
	rows, err := userSQL.GetByName.Query(name, deleted1, deleted2)
	if err != nil {
		rows.Close()
		log.Println(err.Error())
		return nil, err
	}
	users := []*User{}
	for rows.Next() {
		user := new(User)
		if err = rows.Scan(&user.LoginID, &user.Name, &user.Email, &user.Password, &user.DeletedUser, &user.DelOn); err != nil {
			log.Println(err.Error())
			rows.Close()
			return users, err
		}
		users = append(users, user)
	}

	rows.Close()
	return users, nil
}

//Get Users by email
func GetUsersByEmail(email nulls.NullString, delFilter int) ([]*User, error) {
	deleted1 := false
	deleted2 := false
	switch delFilter {
	case DELETED:
		deleted1 = true
		deleted2 = true
	case ALL:
		deleted2 = true
	}
	rows, err := userSQL.GetByEmail.Query(email, deleted1, deleted2)
	if err != nil {
		rows.Close()
		log.Println(err.Error())
		return nil, err
	}
	users := []*User{}
	for rows.Next() {
		user := new(User)
		if err = rows.Scan(&user.LoginID, &user.Name, &user.Email, &user.Password, &user.DeletedUser, &user.DelOn); err != nil {
			log.Println(err.Error())
			rows.Close()
			return users, err
		}
		users = append(users, user)
	}

	rows.Close()
	return users, nil
}

//Update name only
func (user *User) PatchName(name string) error {
	_, err := userSQL.PatchName.Exec(name, user.LoginID)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	user.Name = name
	return nil
}

//DataLayer is used to store prepared SQL statements
type UserDataLayer struct {
	DB         *sql.DB
	GetByID    *sql.Stmt
	Update     *sql.Stmt
	Insert     *sql.Stmt
	Delete     *sql.Stmt
	MarkDel    *sql.Stmt
	GetByName  *sql.Stmt
	GetByEmail *sql.Stmt
	PatchName  *sql.Stmt
	Init       bool
}

//InitUserDataLayer prepares SQL statements and assigns the passed in DB pointer
func InitUserDataLayer(db *sql.DB) error {
	var err error
	if !userSQL.Init {
		userSQL.GetByID, err = db.Prepare("SELECT login_id, name, email, password, deleted_user, del_on FROM vikiblog.public.user WHERE login_id = $1 and (deleted_user = $2 or deleted_user = $3)")
		userSQL.Update, err = db.Prepare("UPDATE vikiblog.public.user SET name = $1, email = $2, password = $3, deleted_user = $4, del_on = $5 WHERE login_id = $6")
		userSQL.Insert, err = db.Prepare("INSERT INTO vikiblog.public.user (name, email, password, deleted_user, del_on) VALUES ($1, $2, $3, $4, $5) RETURNING login_id")
		userSQL.MarkDel, err = db.Prepare("UPDATE vikiblog.public.user SET deleted_user = $1, del_on = $2 WHERE login_id = $3")
		userSQL.Delete, err = db.Prepare("DELETE from vikiblog.public.user WHERE login_id = $1")
		userSQL.GetByName, err = db.Prepare("SELECT login_id, name, email, password, deleted_user, del_on FROM vikiblog.public.user WHERE name = $1 and (deleted_user = $2 or deleted_user = $3) ORDER BY login_id")
		userSQL.GetByEmail, err = db.Prepare("SELECT login_id, name, email, password, deleted_user, del_on FROM vikiblog.public.user WHERE email = $1 and (deleted_user = $2 or deleted_user = $3) ORDER BY login_id")
		userSQL.PatchName, err = db.Prepare("UPDATE vikiblog.public.user SET name = $1 WHERE login_id = $2")
		userSQL.Init = true
		userSQL.DB = db
	}
	return err
}

//CloseUserStmts should be called when prepared SQL statements aren't needed anymore
func CloseUserStmts() {
	if userSQL.Init {
		userSQL.GetByID.Close()
		userSQL.Update.Close()
		userSQL.Insert.Close()
		userSQL.Delete.Close()
		userSQL.MarkDel.Close()
		userSQL.GetByName.Close()
		userSQL.GetByEmail.Close()
		userSQL.PatchName.Close()
		userSQL.Init = false
	}
}
