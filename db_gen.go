package main

import (
	"bytes"
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"log"
	"strconv"
	"strings"
)

//BuildConnString builds the connection string from file
func BuildConnString(dbUser string, password string, dbName string, server string, useSSL bool) string {

	var buffer bytes.Buffer

	buffer.WriteString("user=")
	buffer.WriteString(dbUser)
	buffer.WriteString(" dbname=")
	buffer.WriteString(dbName)
	buffer.WriteString(" host=")
	buffer.WriteString(server)
	buffer.WriteString(" password=")
	buffer.WriteString(password)
	buffer.WriteString(" sslmode=")
	if !useSSL {
		buffer.WriteString("disable")
	} else {
		buffer.WriteString("enable")
	}

	return buffer.String()
}

//CreateOrAlterTables creates and alters tables based on the struct definition file
func CreateOrAlterTables(structObj *structToCreate, db *sql.DB, group string) {
	var tablePathName string = fmt.Sprintf("%s.%s.%s", AddQuotesIfAnyUpperCase(structObj.database), AddQuotesIfAnyUpperCase(structObj.schema), structObj.tableName)
	var oldTableName string
	var row *sql.Row
	var err error
	var indexes []string
	var indexNames []string
	var primCol string

	//find values for needed variables
	for _, col := range structObj.cols {
		if col.primary {
			primCol = col.colName
		}
		if col.index {
			indexNames = append(indexNames, fmt.Sprintf("ix_%s_%s", structObj.tableName, col.colName))
			indexes = append(indexes, fmt.Sprintf("CREATE INDEX ix_%s_%s ON %s USING btree (%s);", structObj.tableName, col.colName, tablePathName, col.colName))
		}
	}

	//Check if a table exists when [alter table] is in the input file
	checkTable := "SELECT EXISTS(SELECT * FROM information_schema.tables WHERE table_name =  $1 and table_schema = $2)"
	if len(structObj.newAltCols) > 0 {
		//copy data from oldAltCols to newAltCols
		var exists bool
		row = db.QueryRow(checkTable, structObj.actionType, structObj.schema)
		err = row.Scan(&exists)
		if err != nil {
			log.Println("\nAn error occurred checking for the table's existence :" + err.Error() + "\n")
			return
		}
		if !exists {
			fmt.Println(fmt.Sprintf("The table %s to be altered doesn't exist in the database. Please make sure the table name matches the name in your Street CRUD file.", structObj.actionType))
			return
		}
	}

	loop := true
	renameTable := structObj.tableName
	//rename old table if needed, store new name if copying of data is needed
	for i := 1; loop; i++ {
		row = db.QueryRow(checkTable, renameTable, structObj.schema)
		err = row.Scan(&loop)
		if err != nil {
			log.Println("\nAn error occurred checking for the table's existence :" + err.Error() + "\n")
			return
		}
		if loop {
			//name exists, store name and check again
			renameTable = structObj.tableName + strconv.Itoa(i)
		}
	}
	//rename old table if it exists
	if renameTable != structObj.tableName {
		alterTable := "ALTER TABLE IF EXISTS %s RENAME TO %s;"
		_, err = db.Exec(fmt.Sprintf(alterTable, tablePathName, fmt.Sprintf("%s", renameTable)))
		if err != nil {
			log.Println("\nThere was an issue changing the existing table's name: " + err.Error() + "\n")
			return
		}
	}

	//Determine the table name for copying from
	if structObj.actionType != "Add" {
		if structObj.tableName == structObj.actionType {
			//use rename
			oldTableName = fmt.Sprintf("%s.%s.%s", AddQuotesIfAnyUpperCase(structObj.database), AddQuotesIfAnyUpperCase(structObj.schema), renameTable)
		} else {
			//structObj.actionType
			oldTableName = fmt.Sprintf("%s.%s.%s", AddQuotesIfAnyUpperCase(structObj.database), AddQuotesIfAnyUpperCase(structObj.schema), structObj.actionType)
		}
	}

	//Check and rename old primary key constraint if needed
	pgClassStmt := "SELECT EXISTS(SELECT relname FROM pg_class WHERE relname = $1)"
	loop = true
	pkConstraint := fmt.Sprintf("pk_%s_%s", structObj.tableName, primCol)
	pkRename := pkConstraint
	for i := 1; loop; i++ {
		row = db.QueryRow(pgClassStmt, pkRename)
		err = row.Scan(&loop)
		if err != nil {
			log.Println("\nAn error occurred checking for the primary key constraint's name:" + err.Error() + "\n")
			return
		}
		if loop {
			//name exists, store name and check again
			pkRename = pkConstraint + strconv.Itoa(i)
		}
	}
	//rename old pk if it exists
	if pkConstraint != pkRename {
		_, err = db.Exec(fmt.Sprintf("ALTER INDEX %s RENAME TO %s;", pkConstraint, pkRename))
		if err != nil {
			log.Println("\nThere was an issue changing the existing pk constraint's name: " + err.Error() + "\n")
			return
		}
	}

	//Check if sequence exists, then rename it if needed
	loop = true
	seqName := fmt.Sprintf("%s_%s_seq", structObj.tableName, primCol)
	seqRename := seqName
	for i := 1; loop; i++ {
		row = db.QueryRow(pgClassStmt, seqRename)
		err = row.Scan(&loop)
		if err != nil {
			log.Println("An error occurred checking for the sequences' name:" + err.Error() + "\n")
			return
		}
		if loop {
			seqRename = seqName + strconv.Itoa(i)
		}
	}
	//rename old seq if it exists
	if seqName != seqRename {
		_, err = db.Exec(fmt.Sprintf("ALTER SEQUENCE %s RENAME TO %s;", seqName, seqRename))
		if err != nil {
			log.Println("\nThere was an issue changing the existing sequences' name: " + err.Error() + "\n")
			return
		}
	}
	seqName = fmt.Sprintf("%s.%s", AddQuotesIfAnyUpperCase(structObj.schema), seqName)

	//Check if indexes exist, then rename if needed
	var indexRename string
	for _, index := range indexNames {
		indexRename = index
		loop = true
		for i := 1; loop; i++ {
			row = db.QueryRow(pgClassStmt, indexRename)
			err = row.Scan(&loop)
			if err != nil {
				log.Println("\nAn error occurred checking for an indexes' name:" + err.Error() + "\n")
				return
			}
			if loop {
				indexRename = index + strconv.Itoa(i)
			}
		}
		if index != indexRename {
			_, err = db.Exec(fmt.Sprintf("ALTER INDEX %s RENAME TO %s;", index, indexRename))
			if err != nil {
				log.Println("\nThere was an issue changing the existing indexes' name: " + err.Error() + "\n")
				return
			}
		}
	}

	//Create new table
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (", tablePathName))
	for i, col := range structObj.cols {
		buffer.WriteString(fmt.Sprintf("%s %s ", col.colName, col.dbType))
		if !col.nulls || col.primary {
			buffer.WriteString("NOT NULL")
		}
		if col.deleted {
			buffer.WriteString(" DEFAULT false")
		}
		if i < len(structObj.cols)-1 {
			buffer.WriteString(", ")
		}
	}
	buffer.WriteString(" ) WITH (OIDS=FALSE);")
	_, err = db.Exec(buffer.String())
	if err != nil {
		log.Println("Issue creating table: " + err.Error() + "\n")
		return
	}

	//Alter permissions
	buffer.Reset()
	buffer.WriteString(fmt.Sprintf("ALTER TABLE %s OWNER to %s; GRANT ALL ON TABLE %s TO %s;", tablePathName, group, tablePathName, group))
	_, err = db.Exec(buffer.String())
	if err != nil {
		log.Println("\nIssue assigning permissions: " + err.Error() + "\n")
		return
	}

	//Copy data from old table to new table if [alter table]
	lastSequence := 1
	copyData := true
	if oldTableName != "" {
		selectFromOld := fmt.Sprintf("SELECT %s FROM %s", strings.Join(structObj.oldAltCols, ", "), oldTableName)
		insertToNew := fmt.Sprintf("INSERT INTO %s (%s) (%s)", tablePathName, strings.Join(structObj.newAltCols, ", "), selectFromOld)
		_, err = db.Exec(insertToNew)
		if err != nil {
			log.Printf("\nIssue copying data from %s to %s: %s\n", oldTableName, tablePathName, err.Error())
			copyData = false
		}
		if structObj.oldColPrim != "" && copyData {
			//make sure old table has rows.
			numRows := 1
			row = db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", oldTableName))
			err = row.Scan(&numRows)
			if err != nil {
				log.Println("\nIssue checking table being altered" + err.Error() + "\n")
				return
			}
			if numRows > 0 {
				//Get last value for primary key
				row = db.QueryRow(fmt.Sprintf("Select MAX(%s) from %s", structObj.oldColPrim, oldTableName))
				err = row.Scan(&lastSequence)
				if err != nil {
					log.Println("\nIssue reading table's primary key :" + err.Error() + "\n")
					return
				}
				lastSequence = lastSequence + 1
			}
		}
	}

	//Add Primary Key
	_, err = db.Exec(fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s PRIMARY KEY (%s);", tablePathName, pkConstraint, primCol))
	if err != nil {
		log.Println("\nCreating the primary key constraint failed: " + err.Error() + "\n")
		return
	}

	//Create and add sequence to primary key
	_, err = db.Exec(fmt.Sprintf("CREATE SEQUENCE %s INCREMENT 1 MINVALUE 1 MAXVALUE 9223372036854775807 START %d CACHE 1; ALTER TABLE %s OWNER to %s; GRANT ALL ON TABLE %s TO %s;", seqName, lastSequence, seqName, group, seqName, group))
	if err != nil {
		log.Println("\nCreating the primary key sequence failed: " + err.Error() + "\n")
		return
	}

	//Bind sequence to primary key column as its defualt value
	_, err = db.Exec(fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET DEFAULT nextval('%s'::regclass);", tablePathName, primCol, seqName))
	if err != nil {
		log.Println("\nBinding the default primary key sequence failed: " + err.Error() + "\n")
		return
	}

	//Loop and add indexes if needed
	for _, stmt := range indexes {
		_, err = db.Exec(stmt)
		if err != nil {
			log.Println("\nCreating an index failed: " + err.Error() + "\n")
			return
		}
	}

}
