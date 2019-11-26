package view

import (
	"log"
)


//err打印到日志文件
func logErr(err error){
	if err != nil {
		log.Println(err)
	}
}


