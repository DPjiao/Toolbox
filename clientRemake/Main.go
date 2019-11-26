package main

import (
	"bufio"
	"bytes"
	"clientRemake/util"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

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

	BlockSize = 1024 << 16
)

func main() {
	/**
	处理命令
	 */
	Start()
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
			fmt.Print("系统信息:")
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
文件处理主方法
 */
func CmdFile(bio *bufio.Reader){
	//查询远程文件夹,需要用到这个字符串,表示路径
	var fq FileQuery
	fq.Path = NativePathSeparator

	quit := false
	for ;quit == false;  {
		//从远程文件夹,得到文件的数组
		fl := QueryFileSending()
		//打印环境信息
		fq.Path = PrintInformation(fl,fq.Path)
		//等待输入一行
		line := readLine(bio)
		if line == "end()" {
			quit = true
		}else{
			if RemoveSpaces(line){
				//使用" "分隔后得到的数组
				strArr := ProcessingInterval(line)
				if strArr[0] == "cp"{
					//CP命令,同时是三个字符串用空格分隔的
					if len(strArr) == 3{
						//传入远程服务器的当前路径,第二个参数,第三个参数
						ExecuteDownloadFile(
							strArr[1],
							strArr[2],
						)
					} else {
						fmt.Println("命令错误,cp命令后面应该跟两个参数")
					}
				}else if strArr[0] == "up" {
					//up命令
					if len(strArr) == 2 {
						ExecuteUploadFile(strArr[1])
					} else {
						fmt.Println("命令错误,up命令后面只能有一个参数")
					}
				}else{
					fmt.Println("不认识的命令ლ(′◉❥◉｀ლ)")
				}
			}
		}
	}
}

/**
执行上传文件任务
第一个参数,要上传的本机文件路径(可以是文件夹)
 */
func ExecuteUploadFile(localPath string){
	np := PathStringReplacement(localPath,NativePathSeparator)
	fi,err := os.Stat(np)
	logErr(err)
	if fi.IsDir() {
		//是文件夹的处理
		filepath.Walk(np, func(path string, info os.FileInfo, err error) error {
			if path != np {
				f,err := os.Stat(path)
				logErr(err)
				if f.IsDir() == false{
					rp := RelativePath(np,path)
					FileUploadRemake(path,rp)
				}
			}
			return nil
		})
	} else {
		FileUploadRemake(np,np)
	}
}

func RelativePath(root string,path string)string{
	arr := strings.Split(path,root)
	return arr[1]
}

/**
文件上传到服务器
localPath为本机路径
第二个参数被做为filename使用
 */
func FileUpload(localPath string,filename string){

	fr,err := os.Open(localPath)
	logErr(err)
	defer fr.Close()
	blockArr := GetBlockAndMd5Array(fr,BlockSize)

	md5Arr := make([]string,0)
	for _,v := range blockArr{
		md5Arr = append(md5Arr,v.MD5Code)
	}
	resp,err := UploadFileRequest(filename,md5Arr)
	logErr(err)
	defer resp.Body.Close()

	var cf CheckFile
	util.BodyToStruct(resp.Body,&cf)
	if cf.Success ==false {
		arr := cf.Msg
		b := UploadMD5File(blockArr,arr)
		if b == false{
			fmt.Println("有文件上传失败了,文件路径为"+localPath+",服务器命名为"+filename)
		}else{
			UploadFileRequest(filename,md5Arr)
		}
	}
}

/**
文件上传到服务器
localPath为本机路径
第二个参数被做为filename使用
 */
func FileUploadRemake(localPath string,filename string){
	fr,err := os.Open(localPath)
	logErr(err)
	defer fr.Close()
	blockArr := GetBlockAndMd5Array(fr,BlockSize)

	md5Arr := make([]string,0)
	for _,v := range blockArr{
		md5Arr = append(md5Arr,v.MD5Code)
	}
	//上传服务器没有的md5块
	HandlerMD5NoUpload(md5Arr,blockArr)

	//上传文件信息
	UploadFileRequest(filename,md5Arr)
}

/**
查询md5数组是否服务器上都已存在,不存在的就上传
 */
func HandlerMD5NoUpload(md5Arr []string,block []Block){
	resp,err := http.Post(ServerAddress+"/MD5","application/json",util.InterfaceToJson(md5Arr))
	logErr(err)
	defer resp.Body.Close()

	var res []string
	util.BodyToStruct(resp.Body,&res)
	if len(res) != 0 {
		for _,v := range res{
			for _,b := range block{
				if v == b.MD5Code {
					buf := new(bytes.Buffer)
					writer := multipart.NewWriter(buf)
					w,err := writer.CreateFormFile("file",b.MD5Code)
					logErr(err)
					w.Write(b.BlockBytes)
					writer.Close()
					_,err = http.Post(ServerAddress + "/MD5Block",writer.FormDataContentType(),buf)
					logErr(err)
				}
			}
		}
	}
}

/**
第一个参数是文件块数组,第二个是需要上传的md5码数组,
从第一个参数里取出块,按照第二个数组上传到服务器
 */
func UploadMD5File(block []Block,md5Arr []string)bool{
	bo := true
	for _,v := range md5Arr{
		for _,b := range block{
			if v == b.MD5Code {
				buf := new(bytes.Buffer)
				writer := multipart.NewWriter(buf)
				w,err := writer.CreateFormFile("file",v)
				logErr(err)
				w.Write(b.BlockBytes)
				writer.Close()
				resp,err := http.Post(ServerAddress + "/MD5Block",writer.FormDataContentType(),buf)
				logErr(err)
				var b bool
				util.BodyToStruct(resp.Body,&b)
				if b == false{
					bo = false
					logErr(errors.New("MD5块上传失败了!"))
				}
			}
		}
	}
	return bo
}

type CheckFile struct {
	Success bool `json:"success"`
	Msg []string `json:"msg"`
}

/**
filePath是上传给服务器的路径
第二个参数是file块的md5组成的切片
 */
func UploadFileRequest(filePath string,md5Array []string)(resp *http.Response, err error){
	buf := new(bytes.Buffer)
	writer := multipart.NewWriter(buf)
	wf,err := writer.CreateFormField("name")
	logErr(err)
	wf.Write([]byte(filePath))
	for _,v := range md5Array{
		wf,err := writer.CreateFormField("md5")
		logErr(err)
		wf.Write([]byte(v))
	}
	writer.Close()
	return http.Post(ServerAddress + "/upload",writer.FormDataContentType(),buf)
}

type Block struct {
	MD5Code string
	BlockBytes []byte
}
/**
参数是一个read对象和块大小
返回切片,每一项是一个块和这个块的md5值
 */
func GetBlockAndMd5Array(fr io.Reader,blockSize int)[]Block{
	block := make([]Block,0)
	buffer := make([]byte,blockSize)
	for ;true;  {
		length,err := fr.Read(buffer)
		if err == io.EOF {
			break
		}else {
			logErr(err)
		}
		buf := buffer[:length]

		bl0 := Block{}
		bl0.MD5Code = fmt.Sprintf("%x",md5.Sum(buf))
		bl0.BlockBytes = []byte(string(buf))

		block = append(block,bl0)
	}
	return block
}


/**
执行下载文件操作
第一个参数是指远程服务器上的filepath字符串(可能为不完整版)
第二个参数是从服务器下载文件后指定存放到本机位置的绝对地址(一个文件夹的地址)
 */
func ExecuteDownloadFile(serverFilePath string, localDir string) {
	//将可能不规则的路径分隔符输入替换为标准的本地分隔符
	localDir = PathStringReplacement(localDir, NativePathSeparator)

	//获得filepath的Slice
	p := QueryDirectoryInformation(serverFilePath)
	for _, v := range p {
		ProcessingFileDownload(localDir, v)
	}
}

/**
根据服务器指定filepath,处理文件下载
 */
func ProcessingFileDownload(localPath string,v string) {
	DownloadToLocal(v, localPath)
}

/**
下载到指定目录
path为服务器上的filepath匹配字符串
localPath为下载到本地机器的绝对路径
 */
func DownloadToLocal(path string,localPath string){
	resp, err := http.Post(ServerAddress + "/FileBlock","application/json", util.InterfaceToJson(path))
	logErr(err)
	defer resp.Body.Close()

	var strArr []string
	util.BodyToStruct(resp.Body,&strArr)
	if strArr == nil{
		fmt.Println("服务器出错了!")
	}else{
		if len(strArr) == 0{
			fmt.Println("不存在的文件,名字:"+path)
		}else{
			//根据路径处理文件下载
			p := HandlerPath(path,localPath)
			ProcessingFileInformation(p,strArr)
		}
	}
}

/**
处理本地路径和filename,以保证能用它们正常创建文件,从而也就保证了来自服务器的块能正常写入到硬盘上
path是filename,localPath是输入的本地绝对地址
处理它们并返回一个可以直接os.Create执行的字符串
 */
func HandlerPath(path string,localPath string)string{
	path = PathStringReplacement(path,NativePathSeparator)
	localPath = PathStringReplacement(localPath,NativePathSeparator)
	b := ([]byte(NativePathSeparator))[0]
	if localPath[len(localPath)-1] == b{
		localPath = localPath[:len(localPath)-1]
	}
	last := strings.LastIndex(path,NativePathSeparator)
	path = path[last:]
	p := localPath + path

	for ;util.Exists(p) == true;  {
		last = strings.LastIndex(path,NativePathSeparator)
		path = path[last:]
		p = localPath + path
	}

	la := strings.LastIndex(p,NativePathSeparator)
	err := os.MkdirAll(p[:la],os.ModePerm)
	logErr(err)

	return p
}

/**
以path为路径创建一个文件,并根据strArr发起请求,将块按照顺序写入path文件
 */
func ProcessingFileInformation(path string,strArr []string){
	//创建文件,并返回文件描述符供于写入
	f,err := os.Create(path)
	logErr(err)
	defer f.Close()
	for _,v := range strArr{
		DownloadFileBlock(v,f)
	}
}

/**
根据MD5块获得服务器文件数据,写入到指定writer
 */
func DownloadFileBlock(blockMD5 string,fw io.Writer){
	fr,err := http.Post(ServerAddress + "/DownloadBlock","application/json",util.InterfaceToJson(blockMD5))
	logErr(err)
	defer fr.Body.Close()
	b,err := ioutil.ReadAll(fr.Body)
	logErr(err)
	_,err = fw.Write(b)
	logErr(err)
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

type PathSubFile struct {
	Path string `json:"path"`
}

//去服务器上查询目录信息
func QueryDirectoryInformation(serverFilePath string)[]string {
	//发送给服务器
	resp, err := http.Post(ServerAddress + "/QuerySubFile","application/json",util.InterfaceToJson(serverFilePath))
	logErr(err)
	var p []string
	//解析返回的json字符串,并包装为结构体作为返回值返回
	util.BodyToStruct(resp.Body, &p)
	return p
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

func RemoveSpaces(line string)bool{
	str := strings.ReplaceAll(line," ","")
	if str != "" {
		return true
	}
	return false
}

/**
输入指令之前,需要的环境信息打印
 */
func PrintInformation(fl []string,cp string)string{
	if fl == nil {
		fmt.Println("服务器出错啦!")
		return cp
	}else{
		FileDirectoryEnvironmentalPrinting(fl,cp)
		return cp
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

/**
打印文件目录和环境信息
 */
func FileDirectoryEnvironmentalPrinting(DirList []string,Path string){
	if len(DirList) == 0 {
		fmt.Println("没有目录信息")
	}else{
		//每一个v都是一个文件的绝对路径字符串
		for _,v := range DirList{
			fmt.Print(v + "\t")
		}
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
			cp aa bb	下载aa到本地的bb目录下(bb是文件夹地址,为绝对地址)
			up aa	上传aa(aa为本机绝对地址,既可以是文件也可以是文件夹)
		`)

	//打印字符串,例如:>/>
	//中间这个符号就是fq.Path的值
	secondLevelPromptString(Path)
}

/**
打印等待命令输入的第二层提示符(可传入string)
打印格式如下:>string>
 */
func secondLevelPromptString(str string){
	fmt.Print(">"+str+">")
}

//查询文件发送post请求
func QueryFileSending()[]string {
	b := new(bytes.Buffer)
	resp,err := http.Post(ServerAddress + "/QueryFile","application/json",b)
	logErr(err)

	var fl []string
	util.BodyToStruct(resp.Body,&fl)
	return fl
}

//查询文件目录请求收到的响应
type FileQueryResp struct {
	Success bool `json:"success"`
	DirList []string `json:"dirList"`
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

//查询文件目录请求
type FileQuery struct {
	//指定的路径
	Path string `json:"path"`
}

//打印等待命令输入的第一层提示符
func firstLayerPrompt(){
	fmt.Print(">")
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
	NativePathSeparator = string(os.PathSeparator)
	//获得服务器操作系统名字
	resp,err := http.Get(ServerAddress + "/systemInfo")
	logErr(err)
	var m Msg
	util.BodyToStruct(resp.Body,&m)
	//根据操作系统,赋值路径分隔符
	if m.Message == "windows"{
		ServerPathSeparator = "\\"
	}else {
		ServerPathSeparator = "/"
	}
}
//获取服务器信息返回值,'/info'
type Msg struct {
	Message string `json:"msg"`
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

//从bufio.Reader读入一行('\n'已去掉)
func readLine(bufioStdin *bufio.Reader)string{
	str,err := bufioStdin.ReadString('\n')
	logErr(err)
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
		return str[:len(str)-1]
	}
}

//err打印到日志文件
func logErr(err error){
	if err != nil {
		log.Println(err)
	}
}