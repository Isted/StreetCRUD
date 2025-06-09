package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func readFileMakeSlice(filePath string) ([]string, error) {

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var fileLines []string
	fileScanner := bufio.NewScanner(file)
	for fileScanner.Scan() {

		fileLines = append(fileLines, fileScanner.Text())
	}

	return fileLines, nil
}

func GetSafePathForSave(filePath string) string {
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		//A file doesn't exist at the given path
		return filePath
	} else {
		//Need to rename file (check if rename name is taken too)
		var newName string
		tryInt := 1
		for tryInt > 0 {
			//filepath.Ext
			newName = strings.Replace(filePath, ".go", "_gen_"+strconv.Itoa(tryInt)+".go", -1)
			if _, err := os.Stat(newName); os.IsNotExist(err) {
				//File doesn't exist here, good to go
				tryInt = 0
			} else {
				//File exists, try another name
				tryInt++
			}
		}
		return newName
	}
}

func BuildStringForFileWrite(structFromFile *structToCreate, isNew bool, packageName string) string {

	var buffer bytes.Buffer
	var primColName string
	var primVarName string
	var primVarType string
	var delColName string
	var delOnColName string
	var delColType string
	var delOnColType string
	var delVarName string
	var delOnVarName string
	var tablePathName string = fmt.Sprintf("%s.%s.%s", AddQuotesIfAnyUpperCase(structFromFile.database), AddQuotesIfAnyUpperCase(structFromFile.schema), structFromFile.tableName)
	structObject := LowerCaseFirstChar(structFromFile.structName)

	//Write package and imports
	if isNew {
		//discover if the time package needs to be included
		time := "\n"
		for _, col := range structFromFile.cols {
			if (col.deletedOn && !col.nulls) || col.goType == "time.Time" {
				time = "\n\"time\"\n"
			}
		}
		buffer.WriteString("package ")
		buffer.WriteString(packageName)
		buffer.WriteString("\n\n")
		buffer.WriteString("import (\n")
		buffer.WriteString("\"database/sql\"\n//DB Driver\n_ \"github.com/lib/pq\"\n\"encoding/json\"\n\"log\"")
		buffer.WriteString(time)
		if structFromFile.nullsPkg {
			buffer.WriteString("\"github.com/markbates/going/nulls\"")
		}
		buffer.WriteString("\n)\n")
	}

	//Write global variable if generated code will be using prepared stmts
	var dataLayerVar string = LowerCaseFirstChar(structFromFile.structName) + "SQL"
	if structFromFile.prepared {
		buffer.WriteString("\n//Global Data Layer\n")
		buffer.WriteString(fmt.Sprintf("var %s %sDataLayer\n", dataLayerVar, structFromFile.structName))
	} else {
		buffer.WriteString("\n//Global DB Pointer\n")
		buffer.WriteString(fmt.Sprintf("var %sDB *sql.DB\n", structFromFile.structName))
	}

	//Get name of primary column and deleted column
	for _, col := range structFromFile.cols {
		if col.primary {
			primColName = col.colName
			primVarName = col.varName
			primVarType = col.goType
		} else if col.deleted {
			//ignore [nulls] if a column is marked as [deleted]
			if col.nulls {
				col.dbType = "boolean"
				col.goType = "bool"
				col.structLine = strings.Replace(col.structLine, "nulls.Bool", "bool", 1)
				col.nulls = false
			}
			delColName = col.colName
			delColType = col.goType
			delVarName = col.varName
		} else if col.deletedOn {
			delOnColName = col.colName
			delOnColType = "time.Time"
			if col.nulls {
				delOnColType = "nulls.Time"
			}
			delOnVarName = col.varName
		}
	}

	//Create query statements
	var indexMethods [][]string
	var patchMethods [][]string
	var updateSet []string
	var insertSet []string
	var insertVals []string
	var selectVals []string
	var objectVars []string
	var updateVars []string
	var insertVars []string
	var sqlVarFinal string
	i := 0
	for _, col := range structFromFile.cols {
		//build slices for insert and update statements
		if !col.primary {
			i += 1
			updateSet = append(updateSet, col.colName+" = $"+strconv.Itoa(i))
			insertSet = append(insertSet, col.colName)
			insertVals = append(insertVals, "$"+strconv.Itoa(i))
			insertVars = append(insertVars, structObject+"."+col.varName)
			updateVars = append(updateVars, structObject+"."+col.varName)
		}
		selectVals = append(selectVals, col.colName)
		objectVars = append(objectVars, "&"+structObject+"."+col.varName)
	}
	sqlVarFinal = "$" + strconv.Itoa(len(structFromFile.cols))
	updateVars = append(updateVars, structObject+"."+primVarName)

	for _, col := range structFromFile.cols {
		if col.index {
			indexMethods = append(indexMethods, []string{fmt.Sprintf("Get%ssBy%s", structFromFile.structName, UpperCaseFirstChar(col.varName)), fmt.Sprintf("SELECT %s FROM %s WHERE %s = $1 ORDER BY %s", strings.Join(selectVals, ", "), tablePathName, col.colName, primColName), LowerCaseFirstChar(col.varName), col.goType, fmt.Sprintf("GetBy%s", UpperCaseFirstChar(col.varName))})
			if delColName != "" {
				indexMethods[len(indexMethods)-1][1] = fmt.Sprintf("SELECT %s FROM %s WHERE %s = $1 and (%s = $2 or %s = $3) ORDER BY %s", strings.Join(selectVals, ", "), tablePathName, col.colName, delColName, delColName, primColName)
			}
		}
		if col.patch {
			patchMethods = append(patchMethods, []string{"Patch" + UpperCaseFirstChar(col.varName), fmt.Sprintf("UPDATE %s SET %s = $1 WHERE %s = $2", tablePathName, col.colName, primColName), LowerCaseFirstChar(col.varName), col.goType, fmt.Sprintf("Patch%s", UpperCaseFirstChar(col.varName)), col.varName})
		}
	}

	selectStmt := fmt.Sprintf("SELECT %s FROM %s WHERE %s = $1", strings.Join(selectVals, ", "), tablePathName, primColName)
	if delColName != "" {
		selectStmt = fmt.Sprintf("%s and (%s = $2 or %s = $3)", selectStmt, delColName, delColName)
	}
	updateStmt := fmt.Sprintf("UPDATE %s SET %s WHERE %s = %s", tablePathName, strings.Join(updateSet, ", "), primColName, sqlVarFinal)
	insertStmt := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) RETURNING %s", tablePathName, strings.Join(insertSet, ", "), strings.Join(insertVals, ", "), primColName)
	markDelStmt := fmt.Sprintf("UPDATE %s SET %s = $1, %s = $2 WHERE %s = $3", tablePathName, delColName, delOnColName, primColName)
	delStmt := fmt.Sprintf("DELETE from %s WHERE %s = $1", tablePathName, primColName)
	constStmt := fmt.Sprintf("\n//Constants used to alter Get queries (for rows marked as deleted)\nconst (\nEXISTS%s = iota\nDELETED%s = iota\nALL%s = iota\n)\n", strings.ToUpper(structFromFile.structName), strings.ToUpper(structFromFile.structName), strings.ToUpper(structFromFile.structName))
	//End Create query statements

	//Write constants used to alter Get queries
	if delColName != "" {
		buffer.WriteString(constStmt)
	}

	//Write struct
	buffer.WriteString("\ntype ")
	buffer.WriteString(structFromFile.structName)
	buffer.WriteString(" struct {\n")
	for _, col := range structFromFile.cols {
		buffer.WriteString(col.structLine)
		buffer.WriteString("\n")
	}
	buffer.WriteString("}\n\n")

	//Write New()
	delFilter := ""
	if delColName != "" {
		delFilter = ", delFilter int"
	}
	buffer.WriteString(fmt.Sprintf("//Initialize and fill a %s object from the DB\nfunc New%s(%s %s%s) (*%s, error) {\n", structFromFile.structName, structFromFile.structName, LowerCaseFirstChar(primVarName), primVarType, delFilter, structFromFile.structName))
	buffer.WriteString(fmt.Sprintf("%s := new(%s)\n", structObject, structFromFile.structName))
	delFilter = ""
	if delColName != "" {
		delFilter = ", deleted1, deleted2"
		buffer.WriteString("deleted1 := false\ndeleted2 := false\nswitch delFilter {\ncase DELETED" + strings.ToUpper(structFromFile.structName) + ":\ndeleted1 = true\ndeleted2 = true\ncase ALL" + strings.ToUpper(structFromFile.structName) + ":\ndeleted2 = true\n}\n")
	}
	if structFromFile.prepared {
		buffer.WriteString(fmt.Sprintf("row := %s.GetByID.QueryRow(%s%s)\n", dataLayerVar, LowerCaseFirstChar(primVarName), delFilter))
	} else {
		buffer.WriteString(fmt.Sprintf("row := %sDB.QueryRow(\"%s\", %s%s)\n", structFromFile.structName, selectStmt, LowerCaseFirstChar(primVarName), delFilter))
	}
	buffer.WriteString(fmt.Sprintf("err := row.Scan(%s)\n", strings.Join(objectVars, ", ")))
	buffer.WriteString(fmt.Sprintf("if err != nil {\nlog.Println(err.Error())\nreturn nil, err\n}\nreturn %s, nil\n}\n\n", structObject))

	//Write UserFromJSON()
	buffer.WriteString(fmt.Sprintf("//Transform JSON into a %s object\nfunc %sFromJSON(%sJSON []byte) (*%s, error) {\n", structFromFile.structName, structFromFile.structName, structObject, structFromFile.structName))
	buffer.WriteString(fmt.Sprintf("%s := new(%s)\nerr := json.Unmarshal(%sJSON, %s)\n", structObject, structFromFile.structName, structObject, structObject))
	buffer.WriteString(fmt.Sprintf("if err != nil{\nlog.Println(err.Error())\nreturn nil, err\n}\nreturn %s, nil\n}\n\n", structObject))

	//Write ToJSON()
	buffer.WriteString(fmt.Sprintf("//Convert a %s object to JSON\nfunc(%s *%s) ToJSON() ([]byte, error) {\n", structFromFile.structName, structObject, structFromFile.structName))
	buffer.WriteString(fmt.Sprintf("%sJSON, err := json.Marshal(%s)\nreturn %sJSON, err\n}\n\n", structObject, structObject, structObject))

	//Write ObjectsToJSON()
	buffer.WriteString(fmt.Sprintf("//Convert multiple %s objects to JSON\nfunc %ssToJSON(%ss []*%s) ([]byte, error) {\n", structFromFile.structName, UpperCaseFirstChar(structObject), structObject, structFromFile.structName))
	buffer.WriteString(fmt.Sprintf("%ssJSON, err := json.Marshal(%ss)\nreturn %ssJSON, err\n}\n\n", structObject, structObject, structObject))

	//Write GetBy()
	delFilter = ""
	if delColName != "" {
		delFilter = ", delFilter int"
	}
	buffer.WriteString(fmt.Sprintf("//Fill %s object with data from DB\nfunc (%s *%s) GetByID(%s %s%s) error {\n", structFromFile.structName, structObject, structFromFile.structName, LowerCaseFirstChar(primVarName), primVarType, delFilter))
	delFilter = ""
	if delColName != "" {
		delFilter = ", deleted1, deleted2"
		buffer.WriteString("deleted1 := false\ndeleted2 := false\nswitch delFilter {\ncase DELETED" + strings.ToUpper(structFromFile.structName) + ":\ndeleted1 = true\ndeleted2 = true\ncase ALL" + strings.ToUpper(structFromFile.structName) + ":\ndeleted2 = true\n}\n")
	}
	if structFromFile.prepared {
		buffer.WriteString(fmt.Sprintf("row := %s.GetByID.QueryRow(%s%s)\n", dataLayerVar, LowerCaseFirstChar(primVarName), delFilter))
	} else {
		buffer.WriteString(fmt.Sprintf("row := %sDB.QueryRow(\"%s\", %s%s)\n", structFromFile.structName, selectStmt, LowerCaseFirstChar(primVarName), delFilter))
	}
	buffer.WriteString(fmt.Sprintf("err := row.Scan(%s)\n", strings.Join(objectVars, ", ")))
	buffer.WriteString("if err != nil {\nlog.Println(err.Error())\nreturn err\n}\nreturn nil\n}\n\n")

	//Write Insert()
	buffer.WriteString(fmt.Sprintf("//Insert %s object to DB\nfunc (%s *%s) Insert() error {\n", structFromFile.structName, structObject, structFromFile.structName))
	if structFromFile.prepared {
		buffer.WriteString(fmt.Sprintf("var id int\n row := %s.Insert.QueryRow(%s)\n", dataLayerVar, strings.Join(insertVars, ", ")))
	} else {
		buffer.WriteString(fmt.Sprintf("var id int\n row := %sDB.QueryRow(\"%s\", %s)\n", structFromFile.structName, insertStmt, strings.Join(insertVars, ", ")))
	}
	buffer.WriteString(fmt.Sprintf("err := row.Scan(&id)\nif err != nil {\nlog.Println(err.Error())\nreturn err\n}\n%s.%s = id\nreturn nil\n}\n\n", structObject, primVarName))

	//Write Update()
	buffer.WriteString(fmt.Sprintf("//Update %s object in DB\nfunc (%s *%s) Update() error {\n", structFromFile.structName, structObject, structFromFile.structName))
	if structFromFile.prepared {
		buffer.WriteString(fmt.Sprintf("_, err := %s.Update.Exec(%s)\n", dataLayerVar, strings.Join(updateVars, ", ")))
	} else {
		buffer.WriteString(fmt.Sprintf("_, err := %sDB.Exec(\"%s\", %s)\n", structFromFile.structName, updateStmt, strings.Join(updateVars, ", ")))
	}
	buffer.WriteString("if err != nil {\nlog.Println(err.Error())\nreturn err\n}\nreturn nil\n}\n\n")

	//Write MarkDeleted() if needed
	if delColName != "" {
		buffer.WriteString(fmt.Sprintf("//Mark a row as deleted at a specific time\nfunc (%s *%s) MarkDeleted(del ", structObject, structFromFile.structName))
		buffer.WriteString(fmt.Sprintf("%s, when %s) error {\n", delColType, delOnColType))
		if structFromFile.prepared {
			buffer.WriteString(fmt.Sprintf("_, err := %s.MarkDel.Exec(del, when, %s.%s)\n", dataLayerVar, structObject, primVarName))
		} else {
			buffer.WriteString(fmt.Sprintf("_, err := %sDB.Exec(\"%s\", del, when, %s.%s)\n", structFromFile.structName, markDelStmt, structObject, primVarName))
		}
		buffer.WriteString("if err != nil {\nlog.Println(err.Error())\nreturn err\n}\n")
		buffer.WriteString(fmt.Sprintf("%s.%s = del\n%s.%s = when\n", structObject, delVarName, structObject, delOnVarName))
		buffer.WriteString("return nil\n}\n\n")
	}

	//Write Delete()
	buffer.WriteString(fmt.Sprintf("//Delete will remove the matching row from the DB"))
	buffer.WriteString(fmt.Sprintf("\nfunc (%s *%s) Delete() error {\n", structObject, structFromFile.structName))
	if structFromFile.prepared {
		buffer.WriteString(fmt.Sprintf("_, err := %s.Delete.Exec(%s.%s)\n", dataLayerVar, structObject, primVarName))
	} else {
		buffer.WriteString(fmt.Sprintf("_, err := %sDB.Exec(\"%s\", %s.%s)\n", structFromFile.structName, delStmt, structObject, primVarName))
	}
	buffer.WriteString("if err != nil {\nlog.Println(err.Error())\nreturn err\n}\nreturn nil\n}\n\n")

	//Write GetObjectsByColumn
	for _, method := range indexMethods {
		buffer.WriteString(fmt.Sprintf("//Get %ss by %s\n", structFromFile.structName, method[2]))
		delFilter = ""
		if delColName != "" {
			delFilter = ", delFilter int"
		}
		buffer.WriteString(fmt.Sprintf("func %s(%s %s%s) ([]*%s, error) {\n", method[0], method[2], method[3], delFilter, structFromFile.structName))
		delFilter = ""
		if delColName != "" {
			delFilter = ", deleted1, deleted2"
			buffer.WriteString("deleted1 := false\ndeleted2 := false\nswitch delFilter {\ncase DELETED" + strings.ToUpper(structFromFile.structName) + ":\ndeleted1 = true\ndeleted2 = true\ncase ALL" + strings.ToUpper(structFromFile.structName) + ":\ndeleted2 = true\n}\n")
		}
		if structFromFile.prepared {
			buffer.WriteString(fmt.Sprintf("rows, err := %s.%s.Query(%s%s)\n", dataLayerVar, method[4], method[2], delFilter))
		} else {
			buffer.WriteString(fmt.Sprintf("rows, err := %sDB.Query(\"%s\", %s%s)\n", structFromFile.structName, method[1], method[2], delFilter))
		}
		buffer.WriteString("if err != nil {\nrows.Close()\nlog.Println(err.Error())\nreturn nil, err\n}\n")
		buffer.WriteString(fmt.Sprintf("%ss := []*%s{}\nfor rows.Next() {\n%s := new(%s)\nif err = rows.Scan(%s); err != nil {\n", structObject, structFromFile.structName, structObject, structFromFile.structName, strings.Join(objectVars, ", ")))
		buffer.WriteString("log.Println(err.Error())\nrows.Close()\nreturn")
		buffer.WriteString(fmt.Sprintf(" %ss, err\n}\n%ss = append(%ss, %s)\n}\n\nrows.Close()\nreturn %ss, nil\n}\n\n", structObject, structObject, structObject, structObject, structObject))
	}

	//Write PatchVar
	for _, method := range patchMethods {
		buffer.WriteString(fmt.Sprintf("//Update %s only\n", method[2]))
		buffer.WriteString(fmt.Sprintf("func (%s *%s) %s(%s %s) error {\n", structObject, structFromFile.structName, method[0], method[2], method[3]))
		if structFromFile.prepared {
			buffer.WriteString(fmt.Sprintf("_, err := %s.%s.Exec(%s, %s.%s)\n", dataLayerVar, method[4], method[2], structObject, primVarName))
		} else {
			buffer.WriteString(fmt.Sprintf("_, err := %sDB.Exec(\"%s\", %s, %s.%s)\n", structFromFile.structName, method[1], method[2], structObject, primVarName))
		}
		buffer.WriteString("if err != nil {\nlog.Println(err.Error())\nreturn err\n}\n")
		buffer.WriteString(fmt.Sprintf("%s.%s = %s\n", structObject, method[5], method[2]))
		buffer.WriteString("return nil\n}\n\n")
	}

	//Create DataLayer section if prepared statements are being used
	if structFromFile.prepared {
		buffer.WriteString(fmt.Sprintf("//DataLayer is used to store prepared SQL statements\ntype %sDataLayer struct {\n", structFromFile.structName))
		buffer.WriteString("DB *sql.DB\nGetByID *sql.Stmt\nUpdate *sql.Stmt\nInsert *sql.Stmt\nDelete *sql.Stmt\n")
		if delColName != "" {
			buffer.WriteString("MarkDel *sql.Stmt\n")
		}
		for _, methodSlc := range indexMethods {
			buffer.WriteString(fmt.Sprintf("%s *sql.Stmt\n", methodSlc[4]))
		}
		for _, methodSlc := range patchMethods {
			buffer.WriteString(fmt.Sprintf("%s *sql.Stmt\n", methodSlc[4]))
		}
		buffer.WriteString("Init bool\n}\n")

		//Write InitDataLayer f() and prepared SQL statements
		buffer.WriteString(fmt.Sprintf("\n//Init%sDataLayer prepares SQL statements and assigns the passed in DB pointer\nfunc Init%sDataLayer(db *sql.DB) error {\nvar err error\nif !%s.Init {\n", structFromFile.structName, structFromFile.structName, dataLayerVar))
		buffer.WriteString(fmt.Sprintf("%s.GetByID, err = db.Prepare(\"%s\")\n", dataLayerVar, selectStmt))
		buffer.WriteString(fmt.Sprintf("%s.Update, err = db.Prepare(\"%s\")\n", dataLayerVar, updateStmt))
		buffer.WriteString(fmt.Sprintf("%s.Insert, err = db.Prepare(\"%s\")\n", dataLayerVar, insertStmt))
		if delColName != "" {
			buffer.WriteString(fmt.Sprintf("%s.MarkDel, err = db.Prepare(\"%s\")\n", dataLayerVar, markDelStmt))
		}
		buffer.WriteString(fmt.Sprintf("%s.Delete, err = db.Prepare(\"%s\")\n", dataLayerVar, delStmt))
		//Write patch and index methods if they exist
		for _, method := range indexMethods {
			buffer.WriteString(fmt.Sprintf("%s.%s, err = db.Prepare(\"%s\")\n", dataLayerVar, method[4], method[1]))
		}
		for _, method := range patchMethods {
			buffer.WriteString(fmt.Sprintf("%s.%s, err = db.Prepare(\"%s\")\n", dataLayerVar, method[4], method[1]))
		}
		buffer.WriteString(fmt.Sprintf("%s.Init = true\n%s.DB = db\n}\nreturn err\n}\n", dataLayerVar, dataLayerVar))
		//Write CloseStmts f()
		buffer.WriteString(fmt.Sprintf("\n//Close%sStmts should be called when prepared SQL statements aren't needed anymore\nfunc Close%sStmts() {\n", structFromFile.structName, structFromFile.structName))
		buffer.WriteString(fmt.Sprintf("if %s.Init {\n%s.GetByID.Close()\n%s.Update.Close()\n%s.Insert.Close()\n%s.Delete.Close()\n", dataLayerVar, dataLayerVar, dataLayerVar, dataLayerVar, dataLayerVar))
		if delColName != "" {
			buffer.WriteString(fmt.Sprintf("%s.MarkDel.Close()\n", dataLayerVar))
		}
		for _, method := range indexMethods {
			buffer.WriteString(fmt.Sprintf("%s.%s.Close()\n", dataLayerVar, method[4]))
		}
		for _, method := range patchMethods {
			buffer.WriteString(fmt.Sprintf("%s.%s.Close()\n", dataLayerVar, method[4]))
		}
		buffer.WriteString(fmt.Sprintf("%s.Init = false\n}\n}\n", dataLayerVar))
	}

	return buffer.String()
}
