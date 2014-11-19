package main //StreetCRUD by Daniel Isted (c) 2014

import (
	"bytes"
	"database/sql"
	"fmt"
	"unicode"
	"unicode/utf8"

	_ "github.com/lib/pq"
	//"./filePro"
	"bufio"
	//"github.com/isted/StreetCRUD"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type structToCreate struct {
	cols       []*column
	oldAltCols []string
	newAltCols []string
	actionType string
	structName string
	tableName  string
	filePath   string
	fileName   string
	hasKey     bool
	deletedCol string
	nullsPkg   bool
}

type column struct {
	colName    string
	varName    string
	structLine string
	goType     string
	dbType     string
	primary    bool
	index      bool
	patch      bool
	size       string // "" if not varchar w/ size
	deleted    bool
	deletedOn  bool
	nulls      bool
}

func (str *structToCreate) SetDefaultTblName(structName string, useUnderscore bool) {
	if useUnderscore {
		str.tableName = "tbl_" + structName
	} else {
		str.tableName = "tbl" + strings.ToLower(structName)
	}
}

func (struc *structToCreate) CheckStructForDeletes() bool {
	var isDel, isDelOn bool
	for _, col := range struc.cols {
		if col.deleted {
			isDel = true
		} else if col.deletedOn {
			isDelOn = true
		}
	}
	if (!isDel && isDelOn) || (isDel && !isDelOn) {
		return false
	}
	return true
}

func (col *column) MapGoTypeToDBTypes() (bool, string) {
	switch strings.ToLower(col.goType) {
	case "int", "int8", "int16", "int32", "uint", "uint8", "uint16", "uint32", "uintptr", "byte":
		col.dbType = "integer"
	case "int64", "uint64":
		col.dbType = "bigint"
	case "float32":
		col.dbType = "real"
	case "float64":
		col.dbType = "double precision"
	case "bool":
		col.dbType = "boolean"
	case "time.time":
		col.dbType = "timestamp without time zone"
	case "string":
		if col.size == "" {
			col.dbType = "character varying"
		} else {
			col.dbType = "character varying(" + col.size + ")"
		}
	case "rune":
		col.dbType = "character varying"
	case "[]byte":
		col.dbType = "bytea"

	default:
		return false, "A non-supported data type (" + col.goType + ") was provided. The [ignore] option can be added to the end of a struct variable allowing it to be ignored for code generation."
	}
	return true, ""
}

func (col *column) MapNullTypes() error {
	switch strings.ToLower(col.goType) {
	case "int":
		col.goType = "nulls.NullInt"
	case "int32":
		col.goType = "nulls.NullInt32"
	case "int64":
		col.goType = "nulls.NullInt64"
	case "uint32":
		col.dbType = "nulls.NullUInt32"
	case "float32":
		col.goType = "nulls.NullFloat32"
	case "float64":
		col.goType = "nulls.NullFloat64"
	case "bool":
		col.goType = "nulls.NullBool"
	case "time.time":
		col.goType = "nulls.NullTime"
	case "string":
		col.goType = "nulls.NullString"
	case "[]byte":
		col.goType = "nulls.NullByteSlice"
	default:
		return fmt.Errorf("A non-supported data type (" + col.goType + ") was provided as a nullable column. Types must be int64, uint32, int32, int, float64, float32,  string, bool, time.Time, or []byte.")
	}
	return nil
}

func CheckColAndTblNames(name string) error {
	runes := []rune(name)
	if len(runes) < 1 {
		return fmt.Errorf("The name was left empty.")
	}
	if !unicode.IsLetter(runes[0]) {
		return fmt.Errorf("The first character of the name must start w/ a letter.")
	}
	for i := 1; i < len(runes); i++ {
		if !unicode.IsLetter(runes[i]) && !unicode.IsDigit(runes[i]) && runes[i] != '_' {
			return fmt.Errorf("At least one character in the name was either not a letter, number, or underscore.")
		}
	}
	return nil
}

func ConvertToUnderscore(camel string) (string, error) {
	var prevRune rune
	var underscore []rune
	for index, runeChar := range camel {
		if runeChar == '_' {
			return strings.ToLower(camel), nil
		}
		if index == 0 {
			if !unicode.IsLetter(runeChar) {
				return "", fmt.Errorf("Table and column names can't start with a character other than a letter.")
			}
			underscore = append(underscore, unicode.ToLower(runeChar))
			prevRune = runeChar
		} else {
			if runeChar == '_' || unicode.IsLetter(runeChar) || unicode.IsDigit(runeChar) {
				//Look for Upper case letters, append _ and make character lower case
				if unicode.IsUpper(runeChar) {
					if !unicode.IsUpper(prevRune) {
						underscore = append(underscore, '_')
					}
					underscore = append(underscore, unicode.ToLower(runeChar))
				} else {
					underscore = append(underscore, runeChar)
				}
			} else {
				return "", fmt.Errorf("Table and column names can't contain non-alphanumeric characters.")
			}
		}
		prevRune = runeChar
	}
	return string(underscore), nil
}

func TrimInnerSpacesToOne(spacey string) string {

	if strings.TrimSpace(spacey) == "" {
		return ""
	}
	var runeSlice []rune
	var isAtStart bool = true
	var isWord bool = false
	for _, runeChar := range spacey {
		if runeChar != ' ' && runeChar != '\t' && isAtStart {
			runeSlice = append(runeSlice, runeChar)
			isAtStart = false
			isWord = true
		} else if isWord {
			if runeChar != ' ' && runeChar != '\t' {
				runeSlice = append(runeSlice, runeChar)
			} else {
				runeSlice = append(runeSlice, ' ')
				isWord = false
			}
		} else if !isWord {
			if runeChar != ' ' && runeChar != '\t' {
				runeSlice = append(runeSlice, runeChar)
				isWord = true
			}
		}
	}
	if runeSlice[len(runeSlice)-1] == ' ' {
		return fmt.Sprint(string(runeSlice[:len(runeSlice)-1]))
	} else {
		return fmt.Sprint(string(runeSlice))
	}

}

func ChangeCaseForRange(changeMe string, startIndex int, endIndex int) string {
	if changeMe == "" || utf8.RuneCountInString(changeMe) < endIndex+1 || startIndex > endIndex || startIndex < 0 {
		return changeMe
	}
	newWord := []rune(changeMe)
	for ; startIndex <= endIndex; startIndex++ {
		newWord[startIndex] = unicode.ToLower(newWord[startIndex])
	}
	return string(newWord)
}

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

func CheckForTables(db *sql.DB) {
	//Have to group structs that will go into the same file

	var strSearchTerm string = "x9"
	var rows *sql.Rows
	var err error
	if strSearchTerm == "x9" {
		rows, err = db.Query("Select practice_id, prac_time from tbl_practice")
	} else {
		rows, err = db.Query("Select loginid, name, email, password from login Where email like $1", "a@a.com")
	}

	if err != nil {
		fmt.Println("Db Error: " + err.Error())
		return
	}

	//defer rows.Close()

	defer func() {
		log.Println("CheckForTables(): Rows Closed")
		fmt.Println("Rows Closed")
		rows.Close()
	}()

	var practiceID int
	var pracTime time.Time
	for rows.Next() {
		err = rows.Scan(&practiceID, &pracTime)
		if err != nil {
			fmt.Println("Db Error: " + err.Error())
			return
		}
		fmt.Printf("\nID: %d\n", practiceID)
		fmt.Println(pracTime)
	}

	//createTable := "Create Table IF NOT EXISTS tbl_practice_code ( practice_id integer NOT NULL, practice_s character varying, CONSTRAINT pk_practice_code PRIMARY KEY (practice_id) ) WITH (OIDS=FALSE); ALTER TABLE tbl_practice_code OWNER TO postgres; GRANT ALL ON TABLE tbl_practice_code TO vikiblogall;"
	createTable := "Alter Table IF EXISTS tbl_practice_code Rename To prac_code1;"
	_, errCreateTbl := db.Exec(createTable)
	if errCreateTbl != nil {
		fmt.Println("Db Error: " + errCreateTbl.Error())
		return
	}
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
			newName = strings.Replace(filePath, ".go", "Gen"+strconv.Itoa(tryInt)+".go", -1)
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

//The start of the main program
func main() {
	const reqBaseVarsC = 7
	var server string
	var dbUser string
	var password string
	var dbName string
	var useSSL bool
	var useUnderscore bool
	var packageName string
	var reqVarCount uint8
	var structsToAdd []*structToCreate
	var structFromFile *structToCreate

	var filePath string
	var isFileFound bool
	var processFail string = "The file could not be processed. "

	//db, err := sql.Open("postgres", "")
	//if err == nil {
	//	CheckForTables(db)
	//}
	fmt.Println("")
	fmt.Println("////////////////////////////////////////////////////")
	fmt.Println("  __  ___  __   ___  ___ ___     __   __        __  ")
	fmt.Println(" /__`  |  |__) |__  |__   |     /  ` |__) |  | |  \\ ")
	fmt.Println(" .__/  |  |  \\ |___ |___  |     \\__, |  \\ \\__/ |__/ ")
	fmt.Println("")
	//fmt.Println("")
	//fmt.Println("")
	fmt.Println("                       __      ")
	fmt.Println("                      |__) \\ / ")
	fmt.Println("                      |__)  |  ")
	fmt.Println("")
	fmt.Println("  __               ___            __  ___  ___  __  ")
	fmt.Println(" |  \\  /\\  |\\ | | |__  |       | /__`  |  |__  |  \\ ")
	fmt.Println(" |__/ /~~\\ | \\| | |___ |___    | .__/  |  |___ |__/ ")
	fmt.Println("")
	fmt.Println("////////////////////////////////////////////////////")
	fmt.Println("")
	fmt.Printf("Show first run instructions here:\n")
	fmt.Printf("Press return to quit.\n")
	//fmt.Println("")
	//uiLoop:
	for {
		fmt.Printf("\nEnter file path for StreetCRUD struct file: ")
		_, err := fmt.Scanf("%s", &filePath)
		if err != nil || filePath == "" {
			fmt.Print("Exiting StreetCRUD\n\n")
			return
		}
		isFileFound = true
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			fmt.Printf("The file %s does not exist.", filePath)
			isFileFound = false
		}
		//build go file and generate SQL
		if isFileFound {
			//read in file
			lineSlices, e := readFileMakeSlice(filePath)
			if lineSlices == nil || e != nil {
				fmt.Printf("The file is empty or missing key elements.\n")
				continue //uiLoop
			}
			//gather directory path for writing generated go files later
			absPath, _ := filepath.Abs(filePath)
			absPath, _ = filepath.Split(absPath)
			var inCollectState bool = false
			var inAddStructState bool = false
			var inCollectStructState bool = false
			var inCollectNamesState bool = false
			var inAlterStructState bool = false
		LineParsed:
			for _, sLine := range lineSlices {
				var bracks []rune
				//Loop through characters since some whitespace is needed for keywords, structure, etc.
				for letterIndex, cLetter := range sLine {
					if (cLetter == '[' || inCollectState) && (!inAddStructState) && (!inAlterStructState) {
						inCollectState = true
						bracks = append(bracks, cLetter)

						if cLetter == ']' {
							inCollectState = false

							switch strings.ToLower(string(bracks)) {

							case "[server]":

								if utf8.RuneCountInString(sLine) <= letterIndex+1 {
									fmt.Print(processFail + "No Server was specified.\n")
									return
								}
								server = strings.TrimSpace(string(sLine[letterIndex+1:]))
								//Check to see if there is only an empty string left after the whitespace was trimmed
								if server == "" {
									fmt.Print(processFail + "[Server] consists of whitespace.\n")
									return
								}
								reqVarCount = reqVarCount + 1
								continue LineParsed

							case "[user]":
								if utf8.RuneCountInString(sLine) <= letterIndex+1 {
									fmt.Print(processFail + "No User was specified.\n")
									return
								}
								dbUser = strings.TrimSpace(string(sLine[letterIndex+1:]))
								//Check to see if there is only an empty string left after the whitespace was trimmed
								if dbUser == "" {
									fmt.Print(processFail + "[User] consists of whitespace.\n")
									return
								}
								reqVarCount = reqVarCount + 1
								continue LineParsed
							case "[password]":
								if utf8.RuneCountInString(sLine) <= letterIndex+1 {
									fmt.Print(processFail + "No [Password] was specified.\n")
									return
								}
								password = strings.TrimSpace(string(sLine[letterIndex+1:]))
								if password == "" {
									fmt.Print(processFail + "[Password] consists of whitespace.\n")
									return
								}
								reqVarCount = reqVarCount + 1
								continue LineParsed
							case "[database]":
								if utf8.RuneCountInString(sLine) <= letterIndex+1 {
									fmt.Print(processFail + "No Database was specified.\n")
									return
								}
								dbName = strings.TrimSpace(string(sLine[letterIndex+1:]))
								if dbName == "" {
									fmt.Print(processFail + "[Database] consists of whitespace.\n")
									return
								}
								reqVarCount = reqVarCount + 1
								continue LineParsed
							case "[ssl]":
								if utf8.RuneCountInString(sLine) <= letterIndex+1 {
									fmt.Print(processFail + "No SSL option was specified.\n")
									return
								}
								if strings.TrimSpace(string(sLine[letterIndex+1:])) == "true" {
									useSSL = true
								} else {
									useSSL = false
								}
								reqVarCount = reqVarCount + 1
								continue LineParsed
							case "[underscore]":
								if utf8.RuneCountInString(sLine) <= letterIndex+1 {
									fmt.Print(processFail + "No Underscore option was specified.\n")
									return
								}
								if strings.TrimSpace(string(sLine[letterIndex+1:])) == "true" {
									useUnderscore = true
								} else {
									useUnderscore = false
								}
								reqVarCount = reqVarCount + 1
								continue LineParsed
							case "[package]":
								if utf8.RuneCountInString(sLine) <= letterIndex+1 {
									fmt.Print(processFail + "No Package was specified.\n")
									return
								}
								packageName = strings.TrimSpace(string(sLine[letterIndex+1:]))
								if packageName == "" {
									fmt.Print(processFail + "[Package] consists of whitespace.\n")
									return
								}
								reqVarCount = reqVarCount + 1
								continue LineParsed
							case "[add struct]":
								//enter addStruct state
								inAddStructState = true
								inCollectState = false
								structFromFile = new(structToCreate)
								structFromFile.actionType = "Add"
								continue LineParsed
							case "[alter table]":
								inAlterStructState = true
								inCollectState = false
								//collect name of struct to alter
								if utf8.RuneCountInString(sLine) <= letterIndex+1 {
									fmt.Print(processFail + "The table name from which to copy data must exist after the [alter table] statement.\n")
									return
								}
								tblToAlter := strings.TrimSpace(string(sLine[letterIndex+1:]))
								if tblToAlter == "" {
									fmt.Print(processFail + "The table name from which to copy data must exist after the [alter table] statement.\n")
									return
								}
								structFromFile = new(structToCreate)
								structFromFile.actionType = tblToAlter
								continue LineParsed
							} //switch

						}

					} else if inAddStructState {
						//checks to make sure all of the base variables were collected
						if reqVarCount < reqBaseVarsC {
							fmt.Print(processFail + "At least one of the following was not specified: [Server], [User], [Password], [Database], [SSL], [Underscore], and/or [Package].\n")
							return
						}

						if cLetter == '[' || inCollectNamesState {
							inCollectNamesState = true
							bracks = append(bracks, cLetter)
							if cLetter == ']' {
								inCollectNamesState = false
								switch strings.ToLower(string(bracks)) {
								case "[table]":
									if utf8.RuneCountInString(sLine) <= letterIndex+1 {
										//No data, use default table naming once struct name is known
										structFromFile.tableName = ""
									} else {
										userTbl := strings.TrimSpace(string(sLine[letterIndex+1:]))
										if userTbl != "" {
											if errNaming := CheckColAndTblNames(userTbl); errNaming != nil {
												fmt.Println(processFail + "[Table] issue: " + errNaming.Error())
												return
											}
										}
										fmt.Println("Test:" + userTbl)
										structFromFile.tableName = userTbl
									}
									continue LineParsed
								case "[file name]":
									if utf8.RuneCountInString(sLine) <= letterIndex+1 {
										//No data, use default file naming
										structFromFile.fileName = ""
									} else {
										var fileName = strings.TrimSpace(string(sLine[letterIndex+1:]))
										if fileName != "" {
											if strings.Contains(strings.ToLower(fileName), ".go") {
												fileName = ChangeCaseForRange(fileName, utf8.RuneCountInString(fileName)-2, utf8.RuneCountInString(fileName)-1)
												structFromFile.fileName = absPath + fileName
												structFromFile.filePath = absPath + fileName
											} else {
												structFromFile.fileName = absPath + fileName + ".go"
											}
										} else {
											structFromFile.fileName = ""
										}
									}
									continue LineParsed
								} //switch
							}
						} else if cLetter == 't' || cLetter == 'T' {
							//Read in stuct name from file
							lineStructDef := strings.Split(sLine, " ")
							if len(lineStructDef) > 1 {
								structFromFile.structName = lineStructDef[1]
							} else {
								fmt.Print(processFail + "No struct name was given.\n")
								return
							}
							//Finish naming Table
							var err error
							var tblName string
							if structFromFile.tableName == "" {
								if errNaming := CheckColAndTblNames(structFromFile.structName); errNaming != nil {
									fmt.Println(processFail + err.Error())
									return
								}
								if useUnderscore {
									tblName, err = ConvertToUnderscore(structFromFile.structName)
									structFromFile.SetDefaultTblName(tblName, useUnderscore)
								} else {
									structFromFile.SetDefaultTblName(structFromFile.structName, useUnderscore)
								}

							} else {
								if useUnderscore {
									structFromFile.tableName, err = ConvertToUnderscore(structFromFile.tableName)
								} else {
									structFromFile.tableName = strings.ToLower(structFromFile.tableName)
								}
							}
							if err != nil {
								fmt.Println(processFail + err.Error())
								return
							}

							//Finish naming File if needed
							if structFromFile.fileName == "" {
								structFromFile.fileName = absPath + strings.ToLower(structFromFile.structName) + ".go"
							}
							inCollectStructState = true
							continue LineParsed

						} else if inCollectStructState {
							//end of struct logic
							if cLetter == '}' {
								if !structFromFile.CheckStructForDeletes() {
									fmt.Println(processFail + "If a column has a [deleted] option, then another column must be marked as [deletedOn] and vice versa.")
									return
								}
								//columns are finished being read, end Add states
								inAddStructState = false
								inCollectStructState = false
								if !structFromFile.hasKey {
									fmt.Println(processFail + "At least one column of type integer must be marked with the keyword [Primary].")
									return
								}
								structsToAdd = append(structsToAdd, structFromFile)
								continue LineParsed
							}
							//Collect column, type, json, bracks
							if strings.TrimSpace(sLine) != "" {
								if lineColumn := strings.Split(TrimInnerSpacesToOne(sLine), " "); len(lineColumn) > 1 {
									var err error
									err = CheckColAndTblNames(lineColumn[0])
									if err != nil {
										fmt.Println(processFail + err.Error())
										return
									}
									col := new(column)
									col.varName = lineColumn[0]
									if useUnderscore {
										col.colName, err = ConvertToUnderscore(lineColumn[0])
										if err != nil {
											fmt.Println(processFail + err.Error())
											return
										}
									} else {
										col.colName = strings.ToLower(lineColumn[0])
									}

									col.goType = lineColumn[1]
									//Handle meta data contained w/in ` `
									strucOptsColumn := strings.Split(sLine, "`")
									if len(strucOptsColumn) > 1 {
										col.structLine = lineColumn[0] + " " + col.goType + " `" + strucOptsColumn[1] + "`"
									} else {
										col.structLine = lineColumn[0] + " " + col.goType
									}

									//Handle column options
									scOptsColumn := strings.Split(sLine, "[")
									var userOptions string
									var wasTypeAssigned bool
									for i := 1; i < len(scOptsColumn); i++ {
										userOptions = strings.ToLower(scOptsColumn[i])
										wasTypeAssigned = false
										switch {
										case userOptions == "primary]":
											switch strings.ToLower(col.goType) {
											case "int", "int8", "int16", "int32", "uint", "uint8", "uint16", "uint32", "uintptr":
												col.dbType = "integer"
											case "int64", "uint64":
												col.dbType = "bigint"
											default:
												fmt.Println(processFail + "Not a known primary key type. StreetCRUD only supports auto incrementing integers at this point.")
												return
											}
											col.primary = true
											wasTypeAssigned = true
											structFromFile.hasKey = true

										case strings.Contains(userOptions, "size:"):
											if col.goType != "string" {
												fmt.Println(processFail + "[size] can only be used with type string.")
												return
											}
											col.size = userOptions[5:strings.IndexRune(userOptions, ']')]
										case userOptions == "index]":
											col.index = true
										case userOptions == "patch]":
											col.patch = true
										case userOptions == "deleted]":
											if strings.ToLower(col.goType) != "bool" {
												fmt.Println(processFail + "A column marked as [deleted] must have the type bool.")
												return
											}
											col.dbType = "boolean"
											col.deleted = true
											wasTypeAssigned = true
										case userOptions == "deletedon]":
											fmt.Println(col.goType)
											if strings.ToLower(col.goType) == "time.time" {
												col.dbType = "timestamp without time zone"
											} else {
												fmt.Println(processFail + "A column marked as [deletedOn] must have the type time.Time.")
												return
											}
											col.deletedOn = true
											wasTypeAssigned = true
										case userOptions == "ignore]":
											//ignore this line of the input struct
											col = nil
											continue LineParsed
										case userOptions == "nulls]":
											col.nulls = true
											structFromFile.nullsPkg = true
										}

									} //for i < len(scOptsColumn)

									if !wasTypeAssigned {
										//map goType to dbType if a dbType wasn't assigned above
										if check, msg := col.MapGoTypeToDBTypes(); !check {
											fmt.Println(processFail + msg)
											return
										}
									}

									if col.nulls {
										if err := col.MapNullTypes(); err != nil {
											fmt.Println(processFail + err.Error())
											return
										}
										fmt.Printf("Nulls No Err: %s", col.goType)
										//Handle meta data contained w/in ` `
										strucOptsColumn := strings.Split(sLine, "`")
										if len(strucOptsColumn) > 1 {
											col.structLine = lineColumn[0] + " " + col.goType + " `" + strucOptsColumn[1] + "`"
										} else {
											col.structLine = lineColumn[0] + " " + col.goType
										}
									}

									//add the built struct to the slice of structs to use for code gen
									structFromFile.cols = append(structFromFile.cols, col)

								} else {
									fmt.Println(processFail + "Struct variable data was missing.")
									return
								}
							}
							continue LineParsed

						}

					} else if inAlterStructState {
						lineMap := strings.Split(TrimInnerSpacesToOne(sLine), "[")
						if len(lineMap) > 1 {
							//collect column mapping data
							if strings.ToLower(lineMap[1]) == "add struct]" {
								inAlterStructState = false
								inAddStructState = true
							} else {
								errorCheck := strings.ToLower(TrimInnerSpacesToOne(sLine))
								if strings.Contains(strings.ToLower(errorCheck), "[to]") {
									if strings.Index(errorCheck, "[") == 0 || utf8.RuneCountInString(errorCheck) <= strings.LastIndex(errorCheck, "]")+1 {
										fmt.Println(processFail + "The old column name and/or the new struct name were not included in one of the [alter table] [to] sections.")
										return
									}
								} else {
									if errorCheck != "[copy cols]" {
										fmt.Println(processFail + "[to] was missing from OldColumnName [to] NewStructVar.")
										return
									}
									continue LineParsed
								}
								//The line appears to be formatted properly
								structFromFile.oldAltCols = append(structFromFile.oldAltCols, strings.TrimSpace(lineMap[0]))
								if useUnderscore {
									under, err := ConvertToUnderscore(strings.TrimSpace(lineMap[1][strings.Index(lineMap[1], "]")+1:]))
									if err != nil {
										fmt.Println(processFail + err.Error())
										return
									}
									structFromFile.newAltCols = append(structFromFile.newAltCols, under)
								} else {
									structFromFile.newAltCols = append(structFromFile.newAltCols, strings.ToLower(strings.TrimSpace(lineMap[1][strings.Index(lineMap[1], "]")+1:])))
								}
							}
						} else {
							fmt.Println(processFail + "Problem mapping columns to structs in [alter table] section.")
							return
						}
						continue LineParsed
					}

				} //for range sLine
			} //for lineSlices

			fmt.Println(dbUser)
			fmt.Println(password)
			fmt.Println(dbName)
			fmt.Println(useSSL)
			fmt.Println(useUnderscore)
			fmt.Println(packageName)
			fmt.Println(reqVarCount)
			fmt.Println(len(structsToAdd))
			fmt.Println()

			//Cycle through structsToAdd
			fileOpen := make(map[string]*os.File)
			pathChanged := make(map[string]string)
			connString := BuildConnString(dbUser, password, dbName, server, useSSL)
			//"/Users/disted/go/src/github.com/isted/StreetCRUD/testing.txt"
			for i, structObj := range structsToAdd {
				if pathChanged[structObj.fileName] == "" {
					//New path, check to make sure it doesn't already exist
					pathChanged[structObj.fileName] = GetSafePathForSave(structObj.fileName)
					fmt.Println(pathChanged[structObj.fileName])
				}
				if fileOpen[pathChanged[structObj.fileName]] == nil {
					//file is new so don't append
					fileOpen[pathChanged[structObj.fileName]], err = os.Create(pathChanged[structObj.fileName])
					if err != nil {
						fmt.Println("There was a problem generating a new go file. " + err.Error())
						return
					}
					//TODO: Doesn't seem to be working, so maybe write a for loop that explicitly closes the files
					//defer fileOpen[structObj.fileName].Close()

					//BuildStringForFileWrite(structObj, true, packageName)
					fileOpen[pathChanged[structObj.fileName]].WriteString(BuildStringForFileWrite(structObj, true, packageName, connString))
				} else {
					//file exists so append
					fileOpen[pathChanged[structObj.fileName]].WriteString(BuildStringForFileWrite(structObj, false, packageName, connString))

				}
				fileOpen[pathChanged[structObj.fileName]].Sync()
				fmt.Printf("Struct: %d\n", i)
				fmt.Println(structObj.structName)
				fmt.Println(structObj.tableName)
				fmt.Println(structObj.fileName)
				fmt.Println()
				for _, c := range structObj.cols {
					fmt.Println("Cols: " + c.colName)
				}
			}

			//Close files manually since the defer.Close() doesn't get called until the program exits
			for _, value := range fileOpen {
				//fmt.Println("Key:", key, "Value:", value)
				value.Close()
				fmt.Println("Closed")
			}

			//defer func() {
			//	log.Println("CheckForTables(): Rows Closed")
			//	defer value.Close()
			//}()

		}

		//reinitialize variables after a file is processed
		dbUser = ""
		password = ""
		dbName = ""
		useSSL = false
		useUnderscore = false
		packageName = ""
		reqVarCount = 0
		structsToAdd = nil
		structFromFile = nil
		filePath = ""
		isFileFound = false
	}
}

func BuildStringForFileWrite(structFromFile *structToCreate, isNew bool, packageName string, conn string) string {

	var buffer bytes.Buffer
	var colName string
	var varName string
	var delColName string

	if isNew {
		buffer.WriteString("package ")
		buffer.WriteString(packageName)
		buffer.WriteString("\n\n")
		buffer.WriteString("import (\n")
		buffer.WriteString("\"database/sql\"\n_ \"github.com/lib/pq\"\n\"encoding/json\"\n\"log\"")
		buffer.WriteString("\n\"time\"")
		if structFromFile.nullsPkg {
			buffer.WriteString("\n\"github.com/markbates/going/nulls\"")
		}
		buffer.WriteString("\n)\n")
	}

	for _, col := range structFromFile.cols {
		if col.primary {
			colName = col.colName
			varName = col.varName
		} else if col.deleted {
			delColName = col.colName
		}
	}

	buffer.WriteString("\ntype ")
	buffer.WriteString(structFromFile.structName)
	buffer.WriteString(" struct {\n")
	for _, col := range structFromFile.cols {
		buffer.WriteString(col.structLine)
		buffer.WriteString("\n")
	}
	buffer.WriteString("}\n\n")

	//for _, col := range structFromFile.cols {
	buffer.WriteString("rows, err = db.Query(Select * from ")
	buffer.WriteString(structFromFile.tableName)
	buffer.WriteString(" Where ")
	buffer.WriteString(colName)
	buffer.WriteString(" = ")
	buffer.WriteString(varName)
	if delColName != "" {
		buffer.WriteString(" && ")
		buffer.WriteString(delColName)
		buffer.WriteString(" = false")
	}
	buffer.WriteString("defer rows.Close()")

	buffer.WriteString("\n)")

	//rows, err = db.Query("Select loginid, name, email, password from login Where email like $1", strSearchTerm)
	//defer rows.Close()

	//var intLoginID int
	//var strName, strEmail, strPassword string
	//for rows.Next() {
	//	err := rows.Scan(&intLoginID, &strName, &strEmail, &strPassword)
	//	PanicIf(err)
	//	fmt.Fprintf(resW, "Login ID: %d\nName: %s\nEmail: %s\nPassword: %s\n", intLoginID, strName, strEmail, strPassword)
	//}

	buffer.WriteString("func (str *structToCreate) GetBlog(id Int64) {")
	buffer.WriteString("func (")
	buffer.WriteString(structFromFile.structName)
	buffer.WriteString(" *")
	buffer.WriteString(structFromFile.structName)
	buffer.WriteString(") Get")
	buffer.WriteString(structFromFile.structName)
	buffer.WriteString("(id Int64)")

	return buffer.String()
}

//Builds the connection string from file
func BuildConnString(dbUser string, password string, dbName string, server string, useSSL bool) string {

	var buffer bytes.Buffer
	buffer.WriteString("postgres://")
	buffer.WriteString(dbUser)
	buffer.WriteString(":")
	buffer.WriteString(password)
	buffer.WriteString("@")
	buffer.WriteString(server)
	buffer.WriteString("/")
	buffer.WriteString(dbName)
	buffer.WriteString("?sslmode=")
	if useSSL {
		buffer.WriteString("disable")
	} else {
		buffer.WriteString("enable")
	}

	return buffer.String()
}
