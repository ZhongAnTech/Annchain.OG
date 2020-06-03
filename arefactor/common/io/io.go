package io

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
)

func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func MkDirIfNotExists(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}

	return os.MkdirAll(path, os.ModePerm)
}

func FixPrefixPath(potentialRoot string, suffix string) (jointPath string) {
	if potentialRoot == "" || path.IsAbs(suffix) {
		return suffix
	}
	return path.Join(potentialRoot, suffix)
}

func CopyFile(sourceFile string, destinationFile string) {
	input, err := ioutil.ReadFile(sourceFile)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = ioutil.WriteFile(destinationFile, input, 0644)
	if err != nil {
		fmt.Println("Error creating", destinationFile)
		fmt.Println(err)
		return
	}
}
