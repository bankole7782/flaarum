package main

import (
	"net/http"
	"github.com/bankole7782/flaarum/flaarum_shared"
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"path/filepath"
	"github.com/gorilla/mux"
	"strings"
	"os"
)


func validateTableStruct(projName string, tableStruct flaarum_shared.TableStruct) error {
	if err := projAndTableNameValidate(tableStruct.TableName); err != nil {
		return err
	}

	dataPath, _ := GetDataPath()
	tableNames := make([]string, 0)
	fis, err := ioutil.ReadDir(filepath.Join(dataPath, projName))
	if err != nil {
		return errors.Wrap(err, "ioutil error")
	}
	for _, fi := range fis {
		tableNames = append(tableNames, fi.Name())
	}

	for _, fks := range tableStruct.ForeignKeys {
		if flaarum_shared.FindIn(tableNames, fks.PointedTable) == -1 {
			return errors.New(fmt.Sprintf("Input Error: The table name '%s' in the 'foreign_keys:' section does not exists in the project '%s'.", 
				fks.PointedTable, projName))
		}
	}

	return nil
}


func formatTableStruct(tableStruct flaarum_shared.TableStruct) string {
	stmt := "table: " + tableStruct.TableName + "\n"
	stmt += "fields:\n"
	for _, fieldStruct := range tableStruct.Fields {
		stmt += "\t" + fieldStruct.FieldName + " " + fieldStruct.FieldType
		if fieldStruct.Required {
			stmt += " required"
		}
		if fieldStruct.Unique {
			stmt += " unique"
		}
		stmt += "\n"
	}
	stmt += "::\n"
	if len(tableStruct.ForeignKeys) > 0 {
		stmt += "foreign_keys:\n"
		for _, fks := range tableStruct.ForeignKeys {
			stmt += "\t" + fks.FieldName + " " + fks.PointedTable + " " + fks.OnDelete + "\n"
		}
		stmt += "::\n"
	}

	if len(tableStruct.UniqueGroups) > 0 {
		stmt += "unique_groups:\n"
		for _, ug := range tableStruct.UniqueGroups {
			stmt += "\t" + strings.Join(ug, " ") + "\n"
		}
		stmt += "::\n"
	}

	return stmt
}


func createTable(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projName := vars["proj"]

	stmt := r.FormValue("stmt")

	tableStruct, err := flaarum_shared.ParseTableStructureStmt(stmt)
	if err != nil {
		printError(w, err)
		return
	}

	dataPath, _ := GetDataPath()

	generalMutex.Lock()
	defer generalMutex.Unlock()

	err = validateTableStruct(projName, tableStruct)
	if err != nil {
		printError(w, err)
		return
	}

	if doesPathExists(filepath.Join(dataPath, projName, tableStruct.TableName)) {
		printError(w, errors.New(fmt.Sprintf("Table '%s' of Project '%s' has already being created.", tableStruct.TableName, projName)))
		return
	}

	toMake := []string{"data", "indexes", "structures"}
	for _, tm := range toMake {
		err := os.MkdirAll(filepath.Join(dataPath, projName, tableStruct.TableName, tm), 0777)
		if err != nil {
			printError(w, errors.Wrap(err, "os error."))
			return
		}
	}

	formattedStmt := formatTableStruct(tableStruct)
	err = ioutil.WriteFile(filepath.Join(dataPath, projName, tableStruct.TableName, "structures", "1"),
		[]byte(formattedStmt), 0777)
	if err != nil {
		printError(w, errors.Wrap(err, "ioutil error."))
		return
	}

	fmt.Fprintf(w, "ok")
}
