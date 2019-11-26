package main

/**
以下是服务器的初始化操作
新建一个本地配置文件(ToolboxConfig.json)
文件结构:
{
	"file":[
		{
			"filepath":"/xxx/xxx/xx.xx",
			"md5Array":[
				"xxxxxxx",
				"xxxxxxx",
				"xxxxxxx"
			],
			"ready":"上传已完成"
		}
	]
}
新建一个系统日志文件,log.Print专用
新建一个file文件夹,用来装以MD5做文件名的文件块
新建一个工作文件夹,里面保存还没有
 */


import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"
	"serverRemake/configuration"
	"serverRemake/util"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var(
	//当前时间,目的是用于新建一个当前文件下的日志文件
	LogWriter = time.Now().Format("2006-01-02-15-04-05") //时间格式
	//配置文件的路径
	ToolboxConfigPath = "ToolboxConfig.json"
	//文件放置的路径
	FilePath = "file"

)

func main() {
	//设置log
	LogWriter+=".log"
	f,err := os.Create(LogWriter)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	log.SetOutput(f)

	/**
	读入配置文件相关配套
	 */
	NecessaryPreparation()

	HandlerServer()
}

//服务
func HandlerServer(){
	router := gin.Default()

	/**
	获取服务器信息
	 */
	 router.POST("/systemInfo",info)

	/**
	文件上传
	 */
	router.POST("/upload",uploadFile)

	/**
	上传MD5进行验证
	 */
	router.POST("/MD5",md5Verification)

	/**
	上传MD5块
	 */
	router.POST("/MD5Block",MD5Block)

	/**
	下载文件块
	 */
	router.POST("/DownloadBlock",DownloadFileBlock)

	/**
	查询指定filepath的块序列
	 */
	router.POST("/FileBlock",FileBlock)

	/**
	查询所有filepath
	 */
	router.POST("/QueryFile",QueryFile)

	/**
	查询配置项filepath对应的文件块
	 */
	router.POST("/QueryFileBlock",QueryFileBlock)

	/**
	查询所有filepath,返回一个数组,元素为满足以下条件的filepath:
	参数为filepath的子串
	 */
	router.POST("/QuerySubFile",QuerySubFile)

	http.ListenAndServe(":8989",router)
}

func QuerySubFile(c *gin.Context){
	var psf string
	c.BindJSON(&psf)

	path := make([]string,0)
	files := configuration.PopToolboxConfig().File
	for _,v := range files{
		if strings.Contains(v.FilePath,psf) {
			path = append(path,v.FilePath)
		}
	}

	c.JSON(200,path)
}

type FilePathBlock struct {
	FilePath string `json:"filepath"`
}

func QueryFileBlock(c *gin.Context){
	var fpb FilePathBlock
	c.BindJSON(fpb)

	resp := make([]string,0)
	files := configuration.PopToolboxConfig().File
	for _,v := range files{
		if v.FilePath == fpb.FilePath{
			resp = v.Md5Array
		}
	}

	c.JSON(200,resp)
}

func QueryFile(c *gin.Context){
	arr := configuration.PopToolboxConfig().File
	pathArr := make([]string,0)
	for _,v := range arr{
		pathArr = append(pathArr,v.FilePath)
	}
	c.JSON(200,pathArr)
}

//获取服务器系统信息
func info(c *gin.Context){
	c.JSON(http.StatusOK,gin.H{
		"msg":runtime.GOOS,
	})
}

func FileBlock(c *gin.Context){
	var fn string
	c.BindJSON(&fn)

	toolBoxConfig := configuration.PopToolboxConfig()
	fileArr := toolBoxConfig.File
	resp := make([]string,0)
	for _,v := range fileArr{
		if fn == v.FilePath {
			resp = v.Md5Array
		}
	}
	c.JSON(200,resp)
}

func DownloadFileBlock(c *gin.Context){
	var md5Code string
	c.BindJSON(&md5Code)

	f,err := os.Open(FilePath + string(os.PathSeparator) + md5Code)
	logErr(err)

	_,err = io.Copy(c.Writer,f)
	logErr(err)

	c.Writer.Flush()
}

func MD5Block(c *gin.Context){
	file,err := c.FormFile("file")
	logErr(err)

	fw,err := os.Create(FilePath + string(os.PathSeparator) + file.Filename)
	logErr(err)

	fr,err := file.Open()
	defer fr.Close()
	_,err = io.Copy(fw,fr)
	logErr(err)
	//向已拥有的md5列表里添加
	AppendFilePathMD5(file.Filename)

	c.JSON(200,true)
}

func md5Verification(c *gin.Context){
	var md5Arr []string
	c.BindJSON(&md5Arr)

	NoArr := make([]string,0)
	for _,v := range md5Arr{
		if CheckMD5(v) == false{
			NoArr = append(NoArr,v)
		}
	}

	//返回一个服务器上不存在的md5文件数组
	c.JSON(200,NoArr)
}

/**
检查MD5c是否在文件夹下存在,若存在,返回true,否则返回false
 */
func CheckMD5(md5c string)bool{
	fp := configuration.PopFilePathMD5()
	_,ok := fp[md5c]
	if ok == false{
		return false
	}
	return true
}

type CheckFile struct {
	Success bool `json:"success"`
	Msg []string `json:"msg"`
}

func uploadFile(c *gin.Context){
	formdata,err := c.MultipartForm()
	logErr(err)

	name := formdata.Value[`name`][0]
	NoArr := make([]string,0)
	md5ca := formdata.Value[`md5`]
	for _,v := range md5ca{
		if CheckMD5(v) == false {
			NoArr = append(NoArr,v)
		}
	}

	if len(NoArr) == 0 {
		UpdateConfigurationFile(name,md5ca,"上传已完成")

		resp := CheckFile{
			Success:true,
			Msg:nil,
		}
		c.JSON(200,resp)
	}else{
		resp := CheckFile{
			Success:false,
			Msg:NoArr,
		}

		c.JSON(200,resp)
	}
}

/**
向配置列表中添加一个文件项
 */
func UpdateConfigurationFile(path string,md5Array []string,ready string){
	fc := configuration.FileConfig{
		FilePath:path,
		Md5Array:md5Array,
		Ready:ready,
	}
	configuration.AddToolboxConfigObject <- fc
}

/**
向已拥有的MD5块变量里append一个md5Code
 */
func AppendFilePathMD5(md5Code string){
	configuration.AddFileMD5 <- md5Code
}

/**
读入配置文件相关配套
 */
func NecessaryPreparation(){

	bl,err := util.CheckConfigurationFileDir(FilePath)
	logErr(err)
	if bl{
		//不存在
		//创建文件夹
		err = os.Mkdir(FilePath,os.ModePerm)
		logErr(err)
	}

	configuration.Initialization(FilePath,ToolboxConfigPath)
}

//响应的body通过json转对象并放入v中
func BodyToStruct(respBody io.ReadCloser,v interface{}){
	b,err := ioutil.ReadAll(respBody)
	logErr(err)
	err = json.Unmarshal(b,v)
	logErr(err)
}

//err打印到日志文件
func logErr(err error){
	if err != nil {
		log.Println(err)
	}
}

