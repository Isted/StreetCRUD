package main

import (
	"fmt"
	"strings"
	"unicode"
)

type structToCreate struct {
	cols       []*column
	oldAltCols []string
	newAltCols []string
	oldColPrim string
	actionType string
	structName string
	tableName  string
	database   string
	schema     string
	filePath   string
	fileName   string
	hasKey     bool
	nullsPkg   bool
	prepared   bool
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
		col.goType = "nulls.Int"
	case "int32":
		col.goType = "nulls.Int32"
	case "int64":
		col.goType = "nulls.Int64"
	case "uint32":
		col.goType = "nulls.UInt32"
	case "float32":
		col.goType = "nulls.Float32"
	case "float64":
		col.goType = "nulls.Float64"
	case "bool":
		col.goType = "nulls.Bool"
	case "time.time":
		col.goType = "nulls.Time"
	case "string":
		col.goType = "nulls.String"
	case "[]byte":
		col.goType = "nulls.ByteSlice"
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
