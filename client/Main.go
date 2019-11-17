package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
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
	Handler()
}

/**
这些变量都应该被当做常量使用,唯一可以修改他们的方法就是EnvironmentalPreparation
 */
var(
	/**
	服务器路径分隔符
	 */
	ServerPathSeparator string
	/**
	本机路径分隔符
	 */
	NativePathSeparator string
)

//环境准备
func EnvironmentalPreparation(){
	NativePathSeparator = string(os.PathSeparator)
	//获得服务器操作系统名字
	resp,err := http.Get("http://localhost:8080/info")
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

//处理命令
func Handler(){
	bio := bufio.NewReader(os.Stdin)
	//初始化全局常量
	EnvironmentalPreparation()
	for ;true; {
		cmdInit()
		firstLayerPrompt()
		str := readLine(bio)
		if str == "help" {
			fmt.Println("功能不健全")
		}else if str == "system" {
			fmt.Println(runtime.GOOS)
		}else if str == "cpu" {
			fmt.Println(runtime.GOARCH)
		}else if str == "end"{
			fmt.Println("====================退出终端====================")
			os.Exit(0)
		}else if str == "sendFile" {
			sendFile(bio)
		}else if str == "queryFile"{
			queryFile(bio)
		}else{
			fmt.Println("不认识的命令ლ(′◉❥◉｀ლ)")
		}
	}
}

//获取服务器信息返回值,'/info'
type Msg struct {
	Message string `json:"msg"`
}

//查询文件目录请求
type FileQuery struct {
	//指定的路径
	Path string `json:"path"`
}

//查询文件目录请求收到的响应
type FileQueryResp struct {
	DirList []string `json:"dirList"`
}

//处理查询文件目录指令
func queryFile(bio *bufio.Reader){
	//查询远程文件夹,需要用到这个字符串,表示路径
	var fq FileQuery
	fq.Path = NativePathSeparator

	quit := false
	for ;quit == false;  {
		//从远程文件夹,得到文件目录信息的数组
		fqr := QueryFileSending(fq)
		//每一个v都是一个文件的绝对路径字符串
		for _,v := range fqr.DirList{
			fmt.Print(v + "  \t  ")
		}
		fmt.Println()
		fmt.Println(`
		命令:
			end() 退出指令
			cd xx 进入指定文件夹
			..		退回上级文件夹
			cp aa bb	下载aa到本地的bb目录下(bb为绝对地址)
		`)
		//打印字符串,例如:>/>
		//中间这个符号就是fq.Path的值
		secondLevelPromptString(fq.Path)
		//等待输入一行
		line := readLine(bio)
		if line == "end()" {
			quit = true
		}else if line[:2] == ".."{
			if fq.Path != NativePathSeparator{
				//去掉最后一个路径分隔符之后的所有字符
				last := strings.LastIndex(fq.Path,NativePathSeparator)
				fq.Path = fq.Path[:last+1]
			}
		}else{
			//使用" "分隔后得到的数组
			strArr := spaceSeparated(line)
			if strArr[0] == "cd" {
				fq.Path = dealWithCD(fq.Path,strArr)
			}else if strArr[0] == "cp"{
				//CP命令,同时是三个字符串用空格分隔的
				if len(strArr) == 3{
					//传入远程服务器的当前路径,第二个参数,第三个参数
					ExecuteDownloadFile(
						strArr[1],
						fq.Path,
						strArr[2],
						)

				} else {
					fmt.Println("命令错误,cp命令后面应该跟两个参数")
				}
			}else{
				fmt.Println("不认识的命令ლ(′◉❥◉｀ლ)")
			}
		}
	}
}

/**
执行下载文件操作
第一个参数是指远程服务器上的目标路径(cp命令的第二个字符串)
第二个参数是指远程服务器的当前路径(控制台实例:>xxx>cp xx xxx)(cp前面那个xx就是这个参数了)
第三个参数是从服务器下载文件后指定存放到本机位置的根目录(cp命令的第三个字符串)
 */
func ExecuteDownloadFile(targetPath string, currentPath string,storageRoot string) {
	//将可能不规则的路径分隔符输入替换为标准的本地分隔符
	storageRoot = PathStringReplacement(storageRoot,NativePathSeparator)
	//获得currentPath + targetPath路径下的目录信息
	ds := QueryDirectoryInformation(targetPath, currentPath)
	//目录切片不为空
	if ds.Success {
		//获取服务器file根目录
		srn := getServerRootDirectory()

		dirSlice := ds.Message
		//dirSlice就是服务器文件夹下的文件绝对路径数组
		for _, v := range dirSlice {
			ProcessingFileDownload(v, srn, storageRoot)
		}
	} else {
		DownloadASingleFile(targetPath,currentPath,storageRoot)
	}
}

//下载单个文件
func DownloadASingleFile(targetPath string,currentPath string,storageRoot string){
	//将targetPath里面的所有'/'或者'\\'符号替换成本机分隔符
	ltp := PathStringReplacement(targetPath,NativePathSeparator)
	ltp = LocalPathProcessingMerge(currentPath,ltp)
	merge := LocalPathProcessingMerge(storageRoot,ltp)
	path := ServerFileRelativePath(targetPath,currentPath)
	DownloadToLocal(path,merge)
}

/**
本地路径处理合并
判断第一个字符串的最后一个字符是不是,不是就加上,
第二个字符串的首部第一个字符是不是,是就减去
最后再把他们合并成一个字符串,
第一个字符串放前面,第二个字符串放后面
 */
func LocalPathProcessingMerge(lastAdd string,headRemove string)string{
	return PathProcessingMerge(lastAdd,headRemove,NativePathSeparator)
}

/**
根据服务器指定文件的绝对路径,处理文件下载
 */
func ProcessingFileDownload(v string, srn ServerRootName, storageRoot string) {
	strArr := strings.Split(v, srn.Message)
	if len(strArr) == 2 {
		//获得服务器的绝对路径,去掉了根目录的剩下字符串,以路径分隔符开头
		path := strArr[1]

		nativePath := strings.Replace(path,ServerPathSeparator,NativePathSeparator,-1)
		DestinationPath := PathProcessingMerge(storageRoot,nativePath,NativePathSeparator)

		DownloadToLocal(path, DestinationPath)
	}
}

/**
下载到指定目录
path为文件在服务器上的绝对路径去掉根目录
DestinationPath为下载到本地机器的绝对路径
 */
func DownloadToLocal(path string,DestinationPath string){
	var fs FileString
	fs.Fs = path
	resp, err := http.Post("http://localhost:8080/downloadFile", "application/json", InterfaceToJson(fs))
	panicErr(err)

	//创建文件夹
	last := strings.LastIndex(DestinationPath,NativePathSeparator)
	newDirString := DestinationPath[:last]
	err = os.MkdirAll(newDirString,os.ModePerm)
	panicErr(err)

	//创建文件,并返回文件描述符供于写入
	f,err := os.Create(DestinationPath)
	panicErr(err)
	defer f.Close()

	_,err = io.Copy(f,resp.Body)
	panicErr(err)
}

//获取服务器file根目录
func getServerRootDirectory()ServerRootName{
	resp, err := http.Get("http://localhost:8080/queryFileRoot")
	panicErr(err)
	var srn ServerRootName
	BodyToStruct(resp.Body, &srn)
	return srn
}

//得到可以用来下载的文件地址(表示了一个远程服务器上的文件相对地址字符串)
func ServerFileRelativePath(targetPath string,currentPath string)string{
	//将targetPath里面的所有'/'或者'\\'符号替换成服务器分隔符
	targetPath = PathStringReplacement(targetPath,ServerPathSeparator)
	//将currentPath里面的所有'/'或者'\\'符号替换成服务器分隔符
	currentPath = PathStringReplacement(currentPath,ServerPathSeparator)
	//得到可以用来下载的文件地址(表示了一个远程服务器上的文件相对地址字符串)
	path := PathProcessingMerge(currentPath, targetPath,ServerPathSeparator)
	return path
}

//去服务器上查询目录信息
func QueryDirectoryInformation(targetPath string, currentPath string) DirectorySlice {
	path := ServerFileRelativePath(targetPath,currentPath)
	//包装一下这个地址字符串
	var fs FileString
	fs.Fs = path
	b := InterfaceToJson(fs)
	//作为一个结构体发送给服务器
	resp, err := http.Post("http://localhost:8080/queryDir", "application/json", b)
	panicErr(err)
	var ds DirectorySlice
	//解析返回的json字符串,并包装为结构体作为返回值返回
	BodyToStruct(resp.Body, &ds)
	return ds
}

//服务器的根目录
type ServerRootName struct {
	Message string `json:"message"`
}

//用来装遍历后的文件目录切片的
type DirectorySlice struct {
	//为true就是目录,message不为空
	Success bool `json:"success"`
	//success为false就不是目录,message为nil
	Message []string `json:"message"`
}

//文件下载请求的结构体
type FileString struct {
	Fs string `json:"fs"`
}

//处理cp命令
func dealWithCP(){

}

//处理cd命令
func dealWithCD(fqPath string,strArr []string)string{
	if len(strArr) == 2 {
		strArr[1] = PathStringReplacement(strArr[1],NativePathSeparator)
		fqPath,strArr[1] = PathProcessing(fqPath,strArr[1],NativePathSeparator)

		fqPath += strArr[1]
	}else{
		fmt.Println("命令错误,cd命令后面应该只跟一个参数")
	}
	return fqPath
}

/**
将字符串中的所有'/'或者'\\'符号都替换为ps
如果没有,就什么操作都不做
 */
func PathStringReplacement(path string,ps string)string{
	if strings.Contains(path,"\\") || strings.Contains(path,"/"){
		if strings.Contains(path,"\\") {
			path = strings.Replace(path,"\\",ps,-1)
		}else{
			path = strings.Replace(path,"/",ps,-1)
		}
	}
	return path
}

/**
判断第一个字符串的最后一位是不是ps,不是就加上,
第二个字符串的第一位是不是ps,是就减去
 */
func PathProcessing(lastAdd string,headRemove string,ps string)(string,string){
	//判断字符串最后一位
	if lastAdd[len(lastAdd)-1] != ps[0] {
		lastAdd += ps
	}
	if headRemove[0] == ps[0] {
		//去掉headRemove的第一个字符
		headRemove = headRemove[1:]
	}
	return lastAdd,headRemove
}

/**
判断第一个字符串的最后一个字符是不是ps,不是就加上,
第二个字符串的首部第一个字符是不是ps,是就减去
最后再把他们合并成一个字符串,
第一个字符串放前面,第二个字符串放后面
 */
func PathProcessingMerge(lastAdd string,headRemove string,ps string)string{
	lastAdd,headRemove = PathProcessing(lastAdd,headRemove,ps)
	return lastAdd + headRemove
}

//对象转json,最后放入一个缓冲区中用以read读取字节
func InterfaceToJson(v interface{})*bytes.Buffer{
	js,err := json.Marshal(v)
	panicErr(err)
	b := bytes.NewBuffer(js)
	return b
}

//响应的body通过json转对象并放入v中
func BodyToStruct(respBody io.ReadCloser,v interface{}){
	b,err := ioutil.ReadAll(respBody)
	panicErr(err)
	json.Unmarshal(b,v)
}

//查询文件发送post请求
func QueryFileSending(fq FileQuery)FileQueryResp{
	b := InterfaceToJson(fq)
	resp,err := http.Post("http://localhost:8080/queryFile","application/json",b)
	panicErr(err)

	var fqr FileQueryResp
	BodyToStruct(resp.Body,&fqr)
	return fqr
}

//上传文件模块打印数据
func ProcessingFilePreparation(print string){
	fmt.Println(print)
	secondLevelPrompt()
}

/*
multipart包装过后的缓冲区,
第一个返回值是包装后用于写入的对象,
第二个返回值是作为容器的bytes.Buffer对象(可以理解为是第一个返回值的缓冲区)
 */
func multipartBuffer()(*multipart.Writer,*bytes.Buffer){
	buf := new(bytes.Buffer)
	writer := multipart.NewWriter(buf)
	return writer,buf
}

//处理发送文件指令
func sendFile(bio *bufio.Reader){
	writer,buf := multipartBuffer()
	quit := false
	for ;quit == false;  {
		ProcessingFilePreparation(`
			run() 			开始上传
			end() 			退出指令
			`+NativePathSeparator+`xxx`+NativePathSeparator+`xxxx.xx 	相对路径的文件格式例子1
			xxxx.xx																相对路径的文件格式例子2
			ap `+NativePathSeparator+`xxx`+NativePathSeparator+`xxxx.xx	绝对路径的文件格式,前面加指令ap
		`)

		//标准输入
		//读取标准输入
		str := readLine(bio)
		if str == "run()"{
			writer.Close()
			resp,err := http.Post("http://localhost:8080/uploadFiles",writer.FormDataContentType(),buf)
			panicErr(err)
			handlingFileUploadResponses(resp)
			quit = true
		}else if str == "end()" {
			writer.Close()
			quit = true
		}else{
			//既不是run命令也不是end命令的情况
			strArr := spaceSeparated(str)
			if strArr[0] == "ap"{
				//首单词是ap的情况
				writeBuffer(writer,strArr[1])
			}else{
				//首单词不是ap的情况
				writeBuffer(writer,strArr[0])
			}
		}
	}
}

//文件上传可能会遇到命名重复的情况,,服务器会将重名的文件名字做成切片返回
type FileUpload struct {
	//为true的时候,表示没有文件重名这种情况,为false表示有,
	Success bool 			`json:"success"`
	//success为false的时候,,重名文件的名字切片,就放在这里面了
	Message []string 		`json:"message"`
}

//处理文件上传响应
func handlingFileUploadResponses(response *http.Response){
	code := response.StatusCode
	body := response.Body
	byteArr,err := ioutil.ReadAll(body)
	panicErr(err)
	if code == 500 {
		fmt.Println("服务器出错了")
	}else {
		var uploadResponse FileUpload
		json.Unmarshal(byteArr,&uploadResponse)

		if uploadResponse.Success == false {
			fmt.Println("上传任务已完成,以下文件重名了,请处理以下文件:")
			arr := uploadResponse.Message
			for _,v := range arr{
				fmt.Println(v)
			}
		}else{
			fmt.Println("上传任务已完成")
		}
	}
}

//将filePath路径指定的文件通过formdata方式添加到writer
func writeBuffer(writer *multipart.Writer,filePath string){
	fileName := getFilesFromPath(filePath)
	fw,err := writer.CreateFormFile("files",fileName)
	panicErr(err)
	//将文件写出去
	go writeOut(fw,filePath)
}

//读取filePath路径的文件,然后写入到writer里,最后写入边界
func writeOut(writer io.Writer,filePath string){
	file,err := os.Open(filePath)
	defer file.Close()
	panicErr(err)

	_,err = io.Copy(writer,file)
	panicErr(err)
}

/**
从一个路径中,取出最后的后缀文件名,
例如从:/file/ios.img 取出 ios.img
路径分隔符会从/或者\\两种情况去分隔
如果没有路径分隔符,则返回参数本身
 */
func getFilesFromPath(path string)string{
	strArr := strings.Split(path,"/")
	if len(strArr) == 1 {
		strArr = strings.Split(path,"\\")
		if len(strArr) == 1{
			return path
		}
	}

	return strArr[len(strArr)-1]
}

//从bufio.Reader读入一行('\n'已去掉)
func readLine(bufioStdin *bufio.Reader)string{
	str,err := bufioStdin.ReadString('\n')
	panicErr(err)
	return tailByte(str)
}

/**
去掉读取的输入流字符串末尾无用字符(指的是linux末尾添加的'\n',windows系统末尾添加的'\r\n')
 */
func tailByte(str string)string{
	if runtime.GOOS == "linux" {
		return str[:len(str)-1]
	}else if runtime.GOOS == "windows"{
		return str[:len(str)-2]
	}else{
		panicText("暂不支持的操作系统")
		return str[:len(str)-1]
	}
}

//将字符串用空格分隔
func spaceSeparated(str string)[]string{
	return strings.Split(str," ")
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
			sendFile 发送文件
			queryFile 查询服务器共享文件
	`)
}

//打印等待命令输入的第一层提示符
func firstLayerPrompt(){
	fmt.Print(">")
}

//打印等待命令输入的第二层提示符
func secondLevelPrompt(){
	fmt.Print(">>")
}

/**
打印等待命令输入的第二层提示符(可传入string)
打印格式如下:>string>
 */
func secondLevelPromptString(str string){
	fmt.Print(">"+str+">")
}

/**
错误panic处理
 */
func panicErr(err error){
	if err != nil {
		panic(err)
	}
}

//向标准输出打印字符串信息
func fmtString(str string){
	fmt.Println(str)
}

/**
根据传入文本信息抛出panic错误
 */
func panicText(text string){
	if text != ""{
		panic(errors.New(text))
	}
}