package main

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	/**
	作为服务端,和客户端对接后给客户端提供文件下载服务,也支持上传服务
	 */
	/**
	环境初始化
	 */
	modifyTheEnvironment()
	/**
	监听上传和下载文件的协程
	 */
	HandlerFiles()
}

//获取文件存放位置的文件夹名字,后缀为"file"
func folderName()string{

	fp,err := os.Getwd()
	panicErr(err)
	fp += string(os.PathSeparator) + "file"

	return fp
}

//获取文件夹根部,后缀为路径分隔符,要新建文件或者文件夹只需要往后面添加标识符就行了.
func folderRoot()string{
	return folderName() + string(os.PathSeparator)
}

/**
环境初始化
 */
func modifyTheEnvironment()string{

	filePath := folderName()

	//根据filepath文件夹是否存在做新建操作
	createFile(filePath)

	//给文件夹添加后缀
	filePath += string(os.PathSeparator)

	return filePath
}

//根据filepath和是否存在做操作
func createFile(filePath string){
	//检查指定文件夹存在否,并赋值,存在为true,不存在为false
	p := exists(filePath)
	if p == false {
		err := os.Mkdir(filePath,os.ModePerm)
		panicErr(err)
		fmt.Println("file文件夹创建完成")
	}else {
		fmt.Println("file文件夹已存在")
	}
}

// 判断所给路径文件/文件夹是否存在
func exists(path string) bool {
	_, err := os.Stat(path)    //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}else {
			return false
		}
	}else {
		return true
	}
}

//文件下载相关请求的结构体
type FileString struct {
	Fs string `json:"fs"`
}

/**
文件下载服务
 */
func downloadFile(c *gin.Context){
	var fs FileString
	c.BindJSON(&fs)

	frs := folderRoot()+fs.Fs
	fr,err := os.Open(frs)
	defer fr.Close()
	panicErr(err)
	w := c.Writer
	iou,err := ioutil.ReadAll(fr)
	w.Write(iou)
	w.Flush()
}

/**
文件服务
 */
func HandlerFiles(){

	hl := gin.Default()

	//上传文件,支持多文件
	hl.POST("/uploadFiles",uploadHandler)
	//下载文件
	hl.POST("/downloadFile",downloadFile)
	//查询文件是否是文件夹,如果是,就返回遍历后的文件路径切片
	hl.POST("/queryDir",queryDir)
	//查询文件目录
	hl.POST("/queryFile",queryFile)
	//查询本地file根目录
	hl.GET("/queryFileRoot",queryFileRoot)
	//获取服务器系统信息
	hl.GET("/info",info)

	http.ListenAndServe(":8080",hl)
}

//获取服务器系统信息
func info(c *gin.Context){
	c.JSON(http.StatusOK,gin.H{
		"msg":runtime.GOOS,
	})
}

//查询本地file根目录
func queryFileRoot(c *gin.Context){
	fn,err := os.Getwd()
	panicErr(err)
	fn += string(os.PathSeparator) + "file"
	c.JSON(http.StatusOK,gin.H{
		"message":fn,
	})
}

//查询文件是否是文件夹,如果是,就返回遍历后的文件路径切片
func queryDir(c *gin.Context){
	var fs FileString
	c.BindJSON(&fs)

	//获取根目录加上前端传来的文件,结果例子:c:/file + /xxx
	filename := folderName() + fs.Fs
	f,err := os.Stat(filename)
	panicErr(err)
	b := f.IsDir()
	//如果filename是一个文件夹
	if b == true{
		pathArr := make([]string,0)
		//遍历指定目录
		filepath.Walk(filename, func(path string, info os.FileInfo, err error) error {
			if filename != path{
				pathArr = append(pathArr,path)
			}
			return nil
		})
		c.JSON(http.StatusOK,gin.H{
			"success":b,
			"message":pathArr,
		})
	}else {
		c.JSON(http.StatusOK,gin.H{
			"success":b,
			"message":nil,
		})
	}
}

//查询文件目录请求
type FileQuery struct {
	//指定的路径
	Path string `json:"path"`
}

//查询文件目录
func queryFile(c *gin.Context){
	var fq FileQuery
	c.BindJSON(&fq)

	files,err := ioutil.ReadDir(folderName()+fq.Path)
	panicErr(err)
	resp := make([]string,len(files))
	for _,v := range files{
		resp = append(resp,v.Name())
	}

	c.JSON(http.StatusOK,gin.H{
		"dirList":resp,
	})
}

//处理文件上传,可以多文件
func uploadHandler(c *gin.Context){
	fd,err := c.MultipartForm()
	panicErr(err)

	//文件切片
	files := fd.File["files"]
	//和服务器上已存在文件重名的切片声明
	dn := make([]string,0)
	//检查是否存在文件重名
	for i,_ := range files {
		file := files[i]
		fn := file.Filename
		if exists(folderRoot() + fn) {
			//存在同名文件了
			dn = append(dn,fn)
		}else{
			//没有同名文件
			fr,err := file.Open()
			defer fr.Close()
			panicErr(err)
			//写入文件
			writeToFile(folderRoot() + fn,fr)
		}
	}

	if len(dn) == 0 {
		//没有重名文件
		c.JSON(http.StatusOK,gin.H{
			"success":true,
			"message":dn,
		})
	}else {
		//存在重名文件
		c.JSON(http.StatusOK,gin.H{
			"success":false,
			"message":dn,
		})
	}
}

//写入到文件
func writeToFile(fw string,fr multipart.File){
	//创建文件夹
	last := strings.LastIndex(fw,string(os.PathSeparator))
	dir := fw[:last]
	err := os.MkdirAll(dir,os.ModePerm)
	panicErr(err)

	f,err := os.Create(fw)
	defer f.Close()
	panicErr(err)
	_,err = io.Copy(f,fr)
	panicErr(err)
}

/**
错误panic处理
 */
func panicErr(err error){
	if err != nil {
		panic(err)
	}
}

/**
根据传入文本信息抛出panic错误
 */
func panicText(text string){
	if text != ""{
		panic(errors.New(text))
	}
}