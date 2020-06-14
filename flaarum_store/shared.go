package main

import (
	"os"
	"path/filepath"
	"github.com/pkg/errors"
	"strings"
	"net/http"
	"fmt"
)


func GetDataPath() (string, error) {
	hd, err := os.UserHomeDir()
	if err != nil {
		return "", errors.Wrap(err, "os error")
	}
	return filepath.Join(hd, ".flaarum_data"), nil
}


func projAndTableNameValidate(name string) error {
	if strings.Contains(name, ".") || strings.Contains(name, " ") || strings.Contains(name, "\t") ||
	strings.Contains(name, "\n") || strings.Contains(name, ":") || strings.Contains(name, "/") {
		return errors.New("object name must not contain space, '.', ':', '/', ")
	}

	return nil
}


func printError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusBadRequest)
	fmt.Fprintf(w, fmt.Sprintf("%+v", err))
}


func doesPathExists(p string) bool {
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return false
	}
	return true
}
