package util

import (
	"os"
	"path/filepath"
)

func GetFileName(path string)string{
	_,name := filepath.Split(path)
	return name
}

/**
检查path是否存在,存在返回false
 */
func CheckConfigurationFileDir(path string)(bool,error){
	_, err := os.Stat(path)
	if err == nil {
		return false, nil
	}
	if os.IsNotExist(err) {
		return true, nil
	}
	return true, err
}