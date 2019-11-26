package util

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
)

//对象转json,最后放入一个缓冲区中用以read读取字节
func InterfaceToJson(v interface{})*bytes.Buffer{
	js,err := json.Marshal(v)
	logErr(err)
	b := bytes.NewBuffer(js)
	return b
}

//响应的body通过json转对象并放入v中
func BodyToStruct(respBody io.ReadCloser,v interface{}){
	b,err := ioutil.ReadAll(respBody)
	logErr(err)
	json.Unmarshal(b,v)
}


//err打印到日志文件
func logErr(err error){
	if err != nil {
		log.Println(err)
	}
}
