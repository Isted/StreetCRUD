package main //StreetCRUD by Daniel Isted (c) 2014

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

//The start of the main program
func main() {
	const reqBaseVarsC = 9
	var server string
	var dbUser string
	var dbGroup string
	var password string
	var dbName string
	var schemaName string
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
	fmt.Printf("Please see github.com/isted/StreetCRUD for instructions.\n")
	fmt.Printf("Press return at any time to quit.\n")
	//uiLoop:
	for {
		fmt.Printf("\nEnter file path for StreetCRUD struct file: ")
		_, err := fmt.Scanf("%s", &filePath)
		if err != nil || filePath == "" {
			fmt.Print("StreetCRUD Closed\n\n")
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

							case "[group]":
								if utf8.RuneCountInString(sLine) > letterIndex+1 {
									dbGroup = strings.TrimSpace(string(sLine[letterIndex+1:]))
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
							case "[schema]":
								if utf8.RuneCountInString(sLine) > letterIndex+1 {
									schemaName = strings.TrimSpace(string(sLine[letterIndex+1:]))
								}
								if schemaName == "" {
									schemaName = "public"
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
								structFromFile.prepared = true
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
								structFromFile.prepared = true
								continue LineParsed
							} //switch
						}

					} else if inAddStructState {
						//checks to make sure all of the base variables were collected
						if reqVarCount < reqBaseVarsC {
							fmt.Print(processFail + "At least one of the following was not specified: [Server], [User], [Password], [Database], [Schema], [SSL], [Underscore], and/or [Package].\n")
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
								case "[prepared]":
									if utf8.RuneCountInString(sLine) <= letterIndex+1 {
										//No data, use prepared statments
										structFromFile.prepared = true
									} else {
										usePrepared := strings.TrimSpace(string(sLine[letterIndex+1:]))
										if strings.ToLower(usePrepared) == "false" || strings.ToLower(usePrepared) == "f" {
											structFromFile.prepared = false
										}
									}
									continue LineParsed
								} //switch
							}
						} else if cLetter == 't' || cLetter == 'T' {
							//Read in stuct name from file
							lineStructDef := strings.Split(sLine, " ")
							if len(lineStructDef) > 1 {
								structFromFile.structName = UpperCaseFirstChar(lineStructDef[1])
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
									structFromFile.tableName = tblName
								} else {
									structFromFile.tableName = strings.ToLower(structFromFile.structName)
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
								structFromFile.database = dbName
								structFromFile.schema = schemaName
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
										userOptions = strings.TrimSpace(strings.ToLower(scOptsColumn[i]))
										wasTypeAssigned = false
										switch {
										case userOptions == "primary]":
											if !structFromFile.hasKey {
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
												//Find and store the primary key column from the old table
												for i, newCol := range structFromFile.newAltCols {
													if newCol == col.colName {
														structFromFile.oldColPrim = structFromFile.oldAltCols[i]
													}
												}
											} else {
												fmt.Println(processFail + "The [primary] keyword can only be used on one column per struct definition.")
												return
											}
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
										//Handle meta data contained w/in ` `
										strucOptsColumn := strings.Split(sLine, "`")
										if len(strucOptsColumn) > 1 {
											col.structLine = lineColumn[0] + " " + col.goType + " `" + strucOptsColumn[1] + "`"
										} else {
											col.structLine = lineColumn[0] + " " + col.goType
										}
									}

									//add the built struct to the slice of structs to use later for code gen
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

			//Assign user to group if group wasn't defined in the file
			if dbGroup == "" {
				dbGroup = dbUser
			}

			//Cycle through structsToAdd
			fileOpen := make(map[string]*os.File)
			pathChanged := make(map[string]string)
			connString := BuildConnString(dbUser, password, dbName, server, useSSL)
			dbConnected := false
			var db *sql.DB
			for _, structObj := range structsToAdd {
				if pathChanged[structObj.fileName] == "" {
					//New path, check to make sure it doesn't already exist
					pathChanged[structObj.fileName] = GetSafePathForSave(structObj.fileName)
				}
				if fileOpen[pathChanged[structObj.fileName]] == nil {
					//file is new so don't append
					fileOpen[pathChanged[structObj.fileName]], err = os.Create(pathChanged[structObj.fileName])
					if err != nil {
						fmt.Println("There was a problem generating a new go file. " + err.Error())
						return
					}

					//BuildStringForFileWrite(structObj, true, packageName)
					fileOpen[pathChanged[structObj.fileName]].WriteString(BuildStringForFileWrite(structObj, true, packageName, connString))
				} else {
					//file exists so append
					fileOpen[pathChanged[structObj.fileName]].WriteString(BuildStringForFileWrite(structObj, false, packageName, connString))

				}
				fileOpen[pathChanged[structObj.fileName]].Sync()

				//Check to see if user wants to generate or alter tables
				var yesOrNo string
				fmt.Printf("File %s generated.", structObj.fileName)
				fmt.Printf("\nDo you want to create/alter the table %s (y or n): ", structObj.tableName)
				_, err := fmt.Scanf("%s", &yesOrNo)
				if err != nil {
					fmt.Println("An error occurred, exiting Street CRUD.\n")
					return
				}
				if strings.ToLower(yesOrNo) == "y" || strings.ToLower(yesOrNo) == "yes" {
					if dbConnected == false {
						dbConnected = true
						var err error
						db, err = sql.Open("postgres", connString)
						if err != nil {
							fmt.Printf("There was a problem opening the database: %s", err.Error())
							return
						}
						if err := db.Ping(); err != nil {
							fmt.Printf("DB connection issue: %s", err.Error())
							return
						}
					}
					CreateOrAlterTables(structObj, db, dbGroup)
				}

			} //end range structsToAdd
			if dbConnected {
				db.Close()
			}
			//Close files manually since the defer.Close() doesn't get called until the program exits
			for _, value := range fileOpen {
				value.Close()
			}

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
