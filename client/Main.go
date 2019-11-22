package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
)

func main(){
	/**
	在开始程序的时候应该先读取本机上的一些配置文件以及系统环境变量,
	还没有设计好,那么就先做解释命令部分吧
	 */
	/**
	命令的环境准备以及等待命令输入,以及收到命令后处理命令
	 */

	/**
	处理命令
	 */
	Start()
}

var(
	/**
	服务器路径分隔符
	 */
	ServerPathSeparator string
	/**
	本机路径分隔符
	 */
	NativePathSeparator string

	/**
   服务器地址
	*/
	ServerAddress string
	/**
	获取服务器系统信息
	 */
	SytemInfo string

	/**
	下载文件
	 */
	DownloadFile string

	/**
	查询本地file根目录
	 */
	QueryFileRoot string

	/**
	查询文件是否是文件夹,如果是,就返回遍历后的文件路径切片
	 */
	QueryDir string

	/**
	查询文件目录
	 */
	QueryFile string

	/**
	上传文件,支持多文件
	 */
	UploadFiles string

	/**
	删除文件或者文件夹
	 */
	DeleteFile string

	/**
	json格式字符串
	 */
	JsonRequest string = "application/json"
)

//环境准备
func EnvironmentalPreparation(bio *bufio.Reader){
	for ;true;  {
		fmt.Print("请输入地址端口号(address:port):")
		str := readLine(bio)
		if HandlingAddressSeparators(str){
			fmt.Println("地址有效,连接中...")
			break
		}else{
			fmt.Println("连接失败,无效地址")
		}
	}
	initConst()
	NativePathSeparator = string(os.PathSeparator)
	//获得服务器操作系统名字
	resp,err := http.Get(SytemInfo)
	panicErr(err)
	var m Msg
	BodyToStruct(resp.Body,&m)
	//根据操作系统,赋值路径分隔符
	if m.Message == "windows"{
		ServerPathSeparator = "\\"
	}else {
		ServerPathSeparator = "/"
	}
}

func initConst(){
	SytemInfo = ServerAddress + "/info"

	/**
	下载文件
	 */
	DownloadFile = ServerAddress + "/downloadFile"

	/**
	查询本地file根目录
	 */
	QueryFileRoot = ServerAddress + "/queryFileRoot"

	/**
	查询文件是否是文件夹,如果是,就返回遍历后的文件路径切片
	 */
	QueryDir = ServerAddress + "/queryDir"

	/**
	查询文件目录
	 */
	QueryFile = ServerAddress + "/queryFile"

	/**
	上传文件,支持多文件
	 */
	UploadFiles = ServerAddress + "/uploadFiles"

	/**
	删除文件或者文件夹
	 */
	DeleteFile = ServerAddress + "/deleteFile"
}

/**
处理文件的控制台主函数
 */
func Start(){
	bio := bufio.NewReader(os.Stdin)
	EnvironmentalPreparation(bio)

	for ;true; {
		cmdInit()
		firstLayerPrompt()
		str := readLine(bio)
		if str == "help" {
			fmt.Println("功能不健全")
		}else if str == "system" {
			fmt.Print("服务器系统信息:")
			fmt.Println()
			fmt.Println(runtime.GOOS)
		}else if str == "cpu" {
			fmt.Println(runtime.GOARCH)
		}else if str == "end"{
			fmt.Println("====================退出终端====================")
			os.Exit(0)
		}else if str == "file"{
			CmdFile(bio)
		}else{
			fmt.Println("不认识的命令ლ(′◉❥◉｀ლ)")
		}
	}
}
/**
处理各种分隔符过后,用地址符分隔
 */
func HandlingAddressSeparators(a string)bool{
	arr := ProcessingInterval(a)
	if len(arr) >= 1 {
		address := strings.Split(arr[0],":")
		if len(address) == 2{
			ServerAddress = "http://" + address[0] + ":" + address[1]
			return true
		}
	}
	return false
}

/**
初始化
 */
func cmdInit(){
	fmt.Println(`
		指令大全:
			help 帮助文档
			system 当前系统
			cpu 当前机器的型号
			end 结束终端
			file 文件处理相关操作
	`)
}