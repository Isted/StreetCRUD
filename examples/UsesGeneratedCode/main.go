//The following is example code that shows
//the basics of how to interact with the
//generated struct file user.go. Process the crudGen.txt
//file to create the table that corresponds with user.go
package main

import (
	"./models"
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/markbates/going/nulls"
	"strconv"
	"strings"
	"time"
)

//IMPORTANT NOTE:
//if user.go was generated w/o prepared statements, the
//models.InitUserDataLayer() and models.CloseUserStmts()
//shouldn't be called. Instead just pass a DB connection
//pointer to your generated file's global DB variable.
//e.g. models.UserDB = userDB before calling any struct
//methods such as Insert(), GetByID(), etc.
//Many of the functions and methods return errors, but I don't use
//them much below since this is just from demonstration purposes

func main() {

	//Open local DB connection
	userDB, _ := sql.Open("postgres", "postgres://userName:Password@localhost/dbName?sslmode=disable")

	//Prepare statments and assign DB
	models.InitUserDataLayer(userDB)

	//Fill User struct and then insert it into the DB
	var user *models.User = &models.User{}
	user.Name = "Viki"
	user.Email = nulls.NewString("Viki@demo.com") //nulls.String{"Viki@demo.com", true}
	user.Password = "pass"
	user.DeletedUser = false
	err := user.Insert()
	if err != nil {
		fmt.Printf("Insert Error: %s", err.Error())
	}
	fmt.Printf("%s Inserted with an ID of %d\n", user.Name, user.LoginID)

	//Lookup newly added User by LoginID
	user.GetByID(user.LoginID, models.ALL)
	fmt.Printf("%s just retrieved from the DB\n", user.Name)

	//Print JSON representation of User object
	message, _ := user.ToJSON()
	fmt.Printf("%s as JSON: %s\n", user.Name, string(message))

	//Get a new User object from the DB using an existing LoginID
	user2, _ := models.NewUser(user.LoginID, models.EXISTS)
	user2.Name = "Victoria"

	//Update Name change in DB
	user2.Update()
	fmt.Printf("Updated name %s to %s in the DB\n", user.Name, user2.Name)

	//Insert two more rows for the demo
	user3 := models.User{Name: "Sam", Email: nulls.NewNullString("Sam@Sam.com"), Password: "secret"}
	user3.Insert()
	user3.Email = nulls.NewString("ChangeLocal")
	user3.Password = "newSam"
	user3.Insert()

	//Get function gets rows by an index (Name in this case)
	var loginIDs []string
	users, _ := models.GetUsersByName("Sam", models.ALL)
	fmt.Println("The following LoginID's have the Name Sam: ")
	for _, usr := range users {
		loginIDs = append(loginIDs, strconv.Itoa(usr.LoginID))
	}
	fmt.Println(strings.Join(loginIDs, ", "))

	//Convert the returned users slice to JSON
	baUsers, _ := models.UsersToJSON(users)
	fmt.Println("Convert and print users named Sam to JSON: ")
	fmt.Println(string(baUsers))

	//Convert JSON to a User struct
	jsonUser := []byte(`{"loginID":15,"name":"Rachel","email":"r@r.com","password":"pw","deleted":false, "deletedOn":null}`)
	user4, e0 := models.UserFromJSON(jsonUser)
	if e0 != nil {
		fmt.Println(e0.Error())
	}
	fmt.Printf("%s was converted from JSON to a User struct.", user4.Name)

	//Insert
	user4.Insert()
	fmt.Printf("\n%s was inserted in the DB. Her LoginID is %d.", user4.Name, user4.LoginID)

	//Mark Rachel as a deleted user
	user4.MarkDeleted(true, nulls.Time{time.Now(), true})
	fmt.Printf("\n%s marked as deleted at %v.\n", user4.Name, user4.DelOn.Time)

	//Delete
	user3.Delete()
	fmt.Printf("%s (LoginID: %d) was permanently deleted.\n", user3.Name, user3.LoginID)

	//Patch Email
	fmt.Printf("%s's name has been changed to ", user4.Name)
	user4.PatchName("Shallan")
	fmt.Printf("%s.\n", user4.Name)
	fmt.Println("Only the name column was updated(patched).\n")

	//Close the User prepared SQL statements
	models.CloseUserStmts()

	//Close local DB connection
	userDB.Close()
}
