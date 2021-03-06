package main

import (
	"net/http"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"strconv"
	"encoding/json"
	"path/filepath"
	"io/ioutil"
	"fmt"
	// "os"
	"strings"
	"time"
	"github.com/bankole7782/flaarum/flaarum_shared"
  "github.com/mcnijman/go-emailaddress"
  "net"
  "net/url"
)


func validateAndMutateDataMap(projName, tableName string, dataMap, oldValues map[string]string) (map[string]string, error) {
  tableStruct, err := getCurrentTableStructureParsed(projName, tableName)
  if err != nil {
    return nil, err
  }

  dataPath, _ := flaarum_shared.GetDataPath()

  fieldsDescs := make(map[string]flaarum_shared.FieldStruct)
  for _, fd := range tableStruct.Fields {
    fieldsDescs[fd.FieldName] = fd
  }

  for k, _ := range dataMap {
    if k == "id" || k == "_version" {
      continue
    }
    _, ok := fieldsDescs[k]
    if ! ok {
      return nil, errors.New(fmt.Sprintf("The field '%s' is not part of this table structure", k))
    }
  }

  for _, fd := range tableStruct.Fields {
    k := fd.FieldName
    v, ok := dataMap[k]

    if ok && v != "" {
      if fd.FieldType == "int" {
        _, err := strconv.ParseInt(v, 10, 64)
        if err != nil {
          return nil, errors.New(fmt.Sprintf("The value '%s' to field '%s' is not of type 'int'", v, k))
        }
      } else if fd.FieldType == "float" {
        _, err := strconv.ParseFloat(v, 64)
        if err != nil {
          return nil, errors.New(fmt.Sprintf("The value '%s' to field '%s' is not of type 'float'", v, k))
        }
      } else if fd.FieldType == "bool" {
        if v != "t" && v != "f" {
          return nil, errors.New(fmt.Sprintf("The value '%s' to field '%s' is not in the short bool format.", v, k))
        }
      } else if fd.FieldType == "date" {
        valueInTimeType, err := time.Parse(flaarum_shared.DATE_FORMAT, v)
        if err != nil {
          return nil, errors.New(fmt.Sprintf("The value '%s' to field '%s' is not in date format.", v, k))
        }
        dataMap[k + "_year"] = strconv.Itoa(valueInTimeType.Year())
        dataMap[k + "_month"] = strconv.Itoa(int(valueInTimeType.Month()))
        dataMap[k + "_day"] = strconv.Itoa(valueInTimeType.Day())
      } else if fd.FieldType == "datetime" {
        valueInTimeType, err := time.Parse(flaarum_shared.DATETIME_FORMAT, v)
        if err != nil {
          return nil, errors.New(fmt.Sprintf("The value '%s' to field '%s' is not in datetime format.", v, k))
        }
        dataMap[k + "_year"] = strconv.Itoa(valueInTimeType.Year())
        dataMap[k + "_month"] = strconv.Itoa(int(valueInTimeType.Month()))
        dataMap[k + "_day"] = strconv.Itoa(valueInTimeType.Day())
        dataMap[k + "_hour"] = strconv.Itoa(valueInTimeType.Hour())
        dataMap[k + "_date"] = valueInTimeType.Format(flaarum_shared.DATE_FORMAT)
				dataMap[k + "_tzname"], _ = valueInTimeType.Zone()
      } else if fd.FieldType == "email" {
        _, err := emailaddress.Parse(v)
        if err != nil {
          return nil, errors.New(fmt.Sprintf("The value '%s' to field '%s' is not in email format.", v, k))
        }
      } else if fd.FieldType == "ipaddr" {
        ipType := net.ParseIP(v)
        if ipType != nil {
          return nil, errors.New(fmt.Sprintf("The value '%s' to field '%s' is not an ip address.", v, k))
        }
      } else if fd.FieldType == "url" {
        _, err := url.Parse(v)
        if err != nil {
          return nil, errors.New(fmt.Sprintf("The value '%s' to field '%s' is not a valid url.", v, k))
        }
      }

    }

    if ok == false && fd.Required {
      return nil, errors.New(fmt.Sprintf("The field '%s' is required.", k))
    }

  }

  for _, fd := range tableStruct.Fields {
    newValue, ok1 := dataMap[fd.FieldName]
    if newValue == "" {
      delete(dataMap, fd.FieldName)
    }
    if oldValues != nil {
      oldValue, ok2 := oldValues[fd.FieldName]
      if ok1 && ok2 && oldValue == newValue {
        continue
      }
    }
    if fd.Unique && ok1 {
      indexPath := filepath.Join(dataPath, projName, tableName, "indexes", fd.FieldName, makeSafeIndexName(newValue))
      if doesPathExists(indexPath) {
        return nil, errors.New(fmt.Sprintf("The data '%s' is not unique to field '%s'.", newValue, fd.FieldName))
      }
    }
  }

  for _, fkd := range tableStruct.ForeignKeys {
    v, ok := dataMap[fkd.FieldName]
    if ok {
      dataPath := filepath.Join(dataPath, projName, fkd.PointedTable, "data", v)
      if ! doesPathExists(dataPath) {
        return nil,  errors.New(fmt.Sprintf("The data with id '%s' does not exist in table '%s'", v, fkd.PointedTable))
      }
    }
  }

  for _, ug := range tableStruct.UniqueGroups {
    wherePartFragment := ""

    for i, fieldName := range ug {

      newValue, ok1 := dataMap[fieldName]
      var value string
      if ok1 {
        value = newValue
      }
      if oldValues != nil {
        oldValue, ok2 := oldValues[fieldName]
        if ok2 && newValue == oldValue {
          continue
        }
      }
      var joiner string
      if i >= 1 {
      	joiner = "and"
      }

      wherePartFragment += fmt.Sprintf("%s %s = '%s' \n", joiner, fieldName, value)
    }

    if len(ug) > 1 {
      innerStmt := fmt.Sprintf(`
      	table: %s
      	where:
      		%s
      	`, tableName, wherePartFragment)
      toCheckRows, err := innerSearch(projName, innerStmt)
      if err != nil {
        return nil, err
      }

      if len(*toCheckRows) > 0 {
        return nil, errors.New(
          fmt.Sprintf("The fields '%s' form a unique group and their data taken together is not unique.",
          strings.Join(ug, ", ")))
      }
    }

  }

  return dataMap, nil
}

func insertRow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projName := vars["proj"]
	tableName := vars["tbl"]

	r.FormValue("email")

	toInsert := make(map[string]string)
	for k, _ := range r.PostForm {
		if k == "key-str" || k == "id" || k == "_version" {
			continue
		}
		if r.FormValue(k) == "" {
			continue
		}
		toInsert[k] = r.FormValue(k)
	}

	currentVersionNum, err := getCurrentVersionNum(projName, tableName)
	if err != nil {
		printError(w, err)
		return
	}

	toInsert["_version"] = fmt.Sprintf("%d", currentVersionNum)

	dataPath, _ := GetDataPath()
	tablePath := filepath.Join(dataPath, projName, tableName)
	if ! doesPathExists(tablePath) {
		printError(w, errors.New(fmt.Sprintf("Table '%s' of Project '%s' does not exists.", tableName, projName)))
		return
	}

	// check if data conforms with table structure
  toInsert, err = validateAndMutateDataMap(projName, tableName, toInsert, nil)
  if err != nil {
  	printError(w, err)
    return
  }

	createTableMutexIfNecessary(projName, tableName)
	fullTableName := projName + ":" + tableName
	tablesMutexes[fullTableName].Lock()
	defer tablesMutexes[fullTableName].Unlock()

  tableStruct, err := getCurrentTableStructureParsed(projName, tableName)
  if err != nil {
    printError(w, err)
    return
  }

  if tableStruct.TableType == "proper" {

    var nextId int64
    if ! doesPathExists(filepath.Join(tablePath, "lastId")) {
      nextId = 1
    } else {
      raw, err := ioutil.ReadFile(filepath.Join(tablePath, "lastId"))
      if err != nil {
        printError(w, errors.Wrap(err, "ioutil error"))
        return
      }

      rawNum, err := strconv.ParseInt(string(raw), 10, 64)
      if err != nil {
        printError(w, errors.Wrap(err, "strconv error"))
        return
      }
      nextId = rawNum + 1
    }

    nextIdStr := strconv.FormatInt(nextId, 10)

    err = saveRowData(projName, tableName, nextIdStr, toInsert)
    if err != nil {
      printError(w, err)
      return
    }

    // create indexes
    for k, v := range toInsert {
      if isFieldOfTypeText(projName, tableName, k) {
        // create a .text file which is a message to the tindexer program.
        newTextFileName := nextIdStr + flaarum_shared.TEXT_INTR_DELIM + k + ".text"
        err = ioutil.WriteFile(filepath.Join(tablePath, "txtinstrs", newTextFileName), []byte(v), 0777)
        if err != nil {
          printError(w, errors.Wrap(err, "ioutil error"))
          return
        }
			} else if isNotIndexedField(projName, tableName, k) {
					// do nothing.
      } else {
        err := makeIndex(projName, tableName, k, v, nextIdStr)
        if err != nil {
          printError(w, err)
          return
        }
      }

    }


    // store last id
    err = ioutil.WriteFile(filepath.Join(tablePath, "lastId"), []byte(nextIdStr), 0777)
    if err != nil {
      printError(w, errors.Wrap(err, "ioutil error"))
      return
    }

    fmt.Fprintf(w, nextIdStr)

  } else if tableStruct.TableType == "logs" {
    nextId := flaarum_shared.UntestedRandomString(15)
    timeNow := time.Now()
    toInsert["created"] = timeNow.Format(flaarum_shared.DATETIME_FORMAT)
    toInsert["created_year"] = strconv.Itoa(timeNow.Year())
    toInsert["created_month"] = strconv.Itoa(int(timeNow.Month()))
    toInsert["created_day"] = strconv.Itoa(timeNow.Day())
    toInsert["created_hour"] = strconv.Itoa(timeNow.Hour())
		toInsert["created_date"] = timeNow.Format(flaarum_shared.DATE_FORMAT)
    toInsert["created_tzname"], _ = timeNow.Zone()

    err = saveRowData(projName, tableName, nextId, toInsert)
    if err != nil {
      printError(w, err)
      return
    }

    // create index only for 'implicit datetime type'
    for k, v := range toInsert {
      if k == "created" || strings.HasPrefix(k, "created_") || k == "_version" {
        err := makeIndex(projName, tableName, k, v, nextId)
        if err != nil {
          printError(w, err)
          return
        }
      }
    }

    fmt.Fprintf(w, nextId)
  }

}


func saveRowData(projName, tableName, rowId string, toWrite map[string]string) error {
  tablePath := getTablePath(projName, tableName)
  jsonBytes, err := json.Marshal(&toWrite)
  if err != nil {
    return errors.Wrap(err, "json error")
  }
  err = ioutil.WriteFile(filepath.Join(tablePath, "data", rowId), jsonBytes, 0777)
  if err != nil {
    return errors.Wrap(err, "write file failed.")
  }

  return nil
}


func makeIndex(projName, tableName, fieldName, newData, rowId string) error {
  return flaarum_shared.MakeIndex(projName, tableName, fieldName, newData, rowId)
}
