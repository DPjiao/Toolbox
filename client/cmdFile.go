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
	"path/filepath"
	"runtime"
	"strings"
)

/**
文件处理主方法
 */
func CmdFile(bio *bufio.Reader){
	//查询远程文件夹,需要用到这个字符串,表示路径
	var fq FileQuery
	fq.Path = NativePathSeparator

	quit := false
	for ;quit == false;  {
		//从远程文件夹,得到文件目录信息的数组
		fqr := QueryFileSending(fq)
		//打印环境信息
		fq.Path = PrintInformation(fqr,fq.Path)
		//等待输入一行
		line := readLine(bio)
		if line == "end()" {
			quit = true
		}else if PathBack(line){
			//回退指令执行
			fq.Path = HandlingRollbackInstructions(fq.Path)
		}else{
			if RemoveSpaces(line){
				//使用" "分隔后得到的数组
				strArr := ProcessingInterval(line)
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
				}else if strArr[0] == "up" {
					//up命令
					if len(strArr) == 2{
						ExecuteUploadFile(fq.Path,strArr[1])
					}else {
						fmt.Println("命令错误,up命令后面只能有一个参数")
					}
				}else if strArr[0] == "delete"{
					if len(strArr) == 2 {
						m := deleteFile(strArr[1])
						fmt.Println(m.Message)
					}else{
						fmt.Println("命令错误,delete有一个参数")
					}
				}else{
					fmt.Println("不认识的命令ლ(′◉❥◉｀ლ)")
				}
			}
		}
	}
}

func deleteFile(path string)Msg{
	var buf FileQuery
	buf.Path = path
	resp,err := http.Post(DeleteFile,JsonRequest,InterfaceToJson(buf))
	panicErr(err)
	var m Msg
	BodyToStruct(resp.Body,&m)
	return m
}

/**
输入指令之前,需要的环境信息打印
 */
func PrintInformation(fqr FileQueryResp,cp string)string{
	if fqr.Success {
		FileDirectoryEnvironmentalPrinting(fqr.DirList,cp)
		return cp
	}else{
		fmt.Println("不是目录")
		pstr := HandlingRollbackInstructions(cp)
		EnvironmentalPrinting(pstr)
		return pstr
	}
}

/**
路径退级操作(如果为根不做任何操作)
输入:/aa/cc
返回:/aa

输入:/
返回:/
 */
func HandlingRollbackInstructions(c string)string{
	if c == NativePathSeparator {
		return c
	}else{
		//去掉最后一个路径分隔符之后的所有字符
		last := strings.LastIndex(c,NativePathSeparator)
		if last == 0{
			c = c[:last+1]
		}else{
			c = c[:last]
		}
		return c
	}
}

//获取服务器信息返回值,'/info'
type Msg struct {
	Message string `json:"msg"`
}

type SystemInfo struct {
	//操作系统
	System string `json:"system"`
	//cpu架构信息
	CpuArch string `json:"cpuArch"`
	//
}

//查询文件目录请求收到的响应
type FileQueryResp struct {
	Success bool `json:"success"`
	DirList []string `json:"dirList"`
}

/**
处理一个间隔字符串(str里面无论包括多少\n\t\r ,等字符,都会当做间隔,然后分隔开装入一个切片中返回)
例子:hello\r\n World  \n !
得到结果: [0]hello
[1]world
[2]!
 */
func ProcessingInterval(str string)[]string{
	str = strings.ReplaceAll(str,"\t"," ")
	str = strings.ReplaceAll(str,"\n"," ")
	str = strings.ReplaceAll(str,"\r"," ")
	strArr := strings.Split(str," ")
	rs := make([]string,0)
	for _,v := range strArr{
		if v != "" {
			rs = append(rs,v)
		}
	}
	return rs
}

func RemoveSpaces(line string)bool{
	str := strings.ReplaceAll(line," ","")
	if str != "" {
		return true
	}
	return false
}

//路径回退指令判断
func PathBack(line string)bool{
	//去掉line中所有空格
	line = strings.ReplaceAll(line," ","")
	if len(line) == 2 && line == ".."{
		return true
	}
	return false
}

/**
打印文件目录和环境信息
 */
func FileDirectoryEnvironmentalPrinting(DirList []string,Path string){
	//每一个v都是一个文件的绝对路径字符串
	for _,v := range DirList{
		fmt.Print(v + "\t")
	}
	EnvironmentalPrinting(Path)
}

/**
打印环境信息
 */
func EnvironmentalPrinting(Path string){
	fmt.Println()
	fmt.Println(`
		命令:
			end() 退出指令
			cd xx 进入指定文件夹
			..		退回上级文件夹
			cp aa bb	下载aa到本地的bb目录下(bb为绝对地址)
			up aa	上传aa到远程的当前目录下(aa为绝对地址)
			delete aa	删除aa
		`)
	//打印字符串,例如:>/>
	//中间这个符号就是fq.Path的值
	secondLevelPromptString(Path)
}

/**
执行上传文件任务
第一个参数,环境当前的远程文件夹路径
第二个参数,要上传的本机文件路径(可以是文件夹)
 */
func ExecuteUploadFile(currentPath string,absolutePath string){
	np := PathStringReplacement(absolutePath,NativePathSeparator)
	currentPath = PathStringReplacement(currentPath,ServerPathSeparator)
	fi,err := os.Stat(np)
	panicErr(err)
	if fi.IsDir() {
		fileMap := make(map[string]string,0)
		//是文件夹的处理
		filepath.Walk(np, func(path string, info os.FileInfo, err error) error {
			fileMap = ProcessingFolder(path, absolutePath, currentPath)
			return nil
		})
		if len(fileMap) != 0 {
			ExecuteMultipleFileUpload(fileMap)
		}
	} else {
		last := strings.LastIndex(np, NativePathSeparator)
		snp := PathStringReplacement(np[last:], ServerPathSeparator)
		fileMap := make(map[string]string,0)
		fileMap[currentPath+snp] = np
		ExecuteMultipleFileUpload(fileMap)
	}
}

/**
处理文件信息,将非文件夹的路径整理起来,放到一个map里去.
map的key是服务器路径格式的环境当前的远程文件夹路径
value是本机的绝对路径
 */
func ProcessingFolder(path string, absolutePath string, currentPath string) map[string]string{
	fileMap := make(map[string]string,0)
	strArr := strings.Split(path, absolutePath)
	if strArr[1] != "" {
		rp := strArr[1]
		last := strings.LastIndex(rp, NativePathSeparator)
		b := strings.Contains(rp[last:], ".")
		if b {
			sp := PathStringReplacement(strArr[1], ServerPathSeparator)
			fileMap[currentPath+sp] = path
		}
	}
	return fileMap
}

/**
执行多文件上传任务
*/
func ExecuteMultipleFileUpload(fileMap map[string]string){
	resp,err := MultipleFileUpload(UploadFiles,fileMap)
	panicErr(err)
	var ds DirectorySlice
	BodyToStruct(resp.Body,&ds)
	if  ds.Success == false{
		fmt.Println("以下文件重名了!")
		for _,v := range ds.Message{
			fmt.Println(v)
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
	resp, err := http.Post(DownloadFile, JsonRequest, InterfaceToJson(fs))
	panicErr(err)
	defer resp.Body.Close()

	//根据路径处理文件下载
	ProcessingFileInformation(DestinationPath,resp.Body)
}

/**
创建文件夹,如果路径末尾有后缀,就创建文件夹并写入文件,如果没有后缀,就只创建文件夹
例子1 c:/echat/abc/test.txt 创建文件夹c:/echat/abc,并在当前文件夹下写入test.txt文件
例子2 c:/echat/abc 创建文件夹c:/echat/abc
 */
func ProcessingFileInformation(path string,body io.Reader){
	last := strings.LastIndex(path,NativePathSeparator)
	if strings.Contains(path[last:],".") {
		//路径最后是有后缀的
		newDirString := path[:last]
		err := os.MkdirAll(newDirString,os.ModePerm)
		panicErr(err)
		//创建文件,并返回文件描述符供于写入
		f,err := os.Create(path)
		panicErr(err)
		defer f.Close()

		_,err = io.Copy(f,body)
		panicErr(err)
	}else{
		//没有后缀
		err := os.MkdirAll(path,os.ModePerm)
		panicErr(err)
	}
}

//获取服务器file根目录
func getServerRootDirectory()ServerRootName{
	resp, err := http.Get(QueryFileRoot)
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
	resp, err := http.Post(QueryDir, JsonRequest, b)
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

//查询文件目录请求
type FileQuery struct {
	//指定的路径
	Path string `json:"path"`
}

//查询文件发送post请求
func QueryFileSending(fq FileQuery)FileQueryResp{
	fq.Path = PathStringReplacement(fq.Path,ServerPathSeparator)
	b := InterfaceToJson(fq)
	resp,err := http.Post(QueryFile,JsonRequest,b)
	panicErr(err)

	var fqr FileQueryResp
	BodyToStruct(resp.Body,&fqr)
	return fqr
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

/**
Multiple多文件上传
参数是一个map,key是上传文件到服务器时,提交给服务器的文件路径
value是该文件在本地的绝对路径
 */
func MultipleFileUpload(urlAddress string,fileMap map[string]string)(*http.Response,error){
	writer,buf := multipartBuffer()
	for k,v := range fileMap{
		w,err := writer.CreateFormFile("files",k)
		panicErr(err)

		f,err := os.Open(v)
		panicErr(err)
		defer f.Close()

		_,err = io.Copy(w,f)
		panicErr(err)
	}

	writer.Close()
	return http.Post(urlAddress,writer.FormDataContentType(),buf)
}

/**
Multiple多文件上传(MD5秒传)
参数是一个map,key是上传文件到服务器时,提交给服务器的文件路径
value是该文件在本地的绝对路径
 */
func MD5MultipleFileUpload(urlAddress string,fileMap map[string]string)(*http.Response,error){

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

//打印等待命令输入的第一层提示符
func firstLayerPrompt(){
	fmt.Print(">")
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

/**
根据传入文本信息抛出panic错误
 */
func panicText(text string){
	if text != ""{
		panic(errors.New(text))
	}
}