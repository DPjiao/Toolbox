package util

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
)

//响应的body通过json转对象并放入v中
func BodyToStruct(respBody io.ReadCloser,v interface{}){
	b,err := ioutil.ReadAll(respBody)
	logErr(err)
	if len(b) != 0{
		err = json.Unmarshal(b,v)
		logErr(err)
	}
}

//err打印到日志文件
func logErr(err error){
	if err != nil {
		log.Println(err)
	}
}