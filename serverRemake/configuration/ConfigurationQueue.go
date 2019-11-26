package configuration

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"serverRemake/util"
)

var(
	//配置文件的路径
	toolboxConfigPath string
	//配置文件
	toolboxConfigObject ToolboxConfig
	/**
	专门用来给出队列提供数据的副本(因为存在引用,所以得深copy传输)
	 */
	toolboxConfigObjectCopy ToolboxConfig

	//文件放置文件夹下已拥有的MD5文件(MD5码的16进制字符串)
	//用map类型的原因是方便遍历,要想知道存不存在该MD5,直接把MD5当做key来里面找一次就行了,复杂度是1
	//如果用slice,那就需要循环遍历,复杂度为n
	//基于此,选择了用map
	filePathMD5 map[string]interface{}
	/**
	专门用来给出队列提供数据的副本(因为存在引用,所以得深copy传输)
	 */
	filePathMD5Copy map[string]interface{}

	/**
	ToolboxConfigObject的出队列
	 */
	departureToolboxConfigObject chan ToolboxConfig = make(chan ToolboxConfig)
	/**
	FilePathMD5的出队列
	 */
	departureFilePathMD5 chan map[string]interface{} = make(chan map[string]interface{})

	/**
	外部开放,FilePathMD5添加一个string键的队列
	 */
	AddFileMD5 chan string = make(chan string)
	/**
	外部开放,ToolboxConfig添加一个配置项的队列
	 */
	AddToolboxConfigObject chan FileConfig = make(chan FileConfig)
)

/**
获取当前FilePathMD5
 */
func PopFilePathMD5()map[string]interface{}{
	return <- departureFilePathMD5
}

/**
获取当前ToolboxConfig
 */
func PopToolboxConfig()ToolboxConfig{
	return <- departureToolboxConfigObject
}

/**
配置文件
 */
type ToolboxConfig struct {
	File []FileConfig `json:"file"`
}

/**
配置项
 */
type FileConfig struct {
	FilePath string `json:"filepath"`
	Md5Array []string `json:"md5Array"`
	Ready string `json:"ready"`
}

/**
环境初始化
 */
func Initialization(FilePath string,ToolboxConfigPath string){
	toolboxConfigPath = ToolboxConfigPath
	b,err := checkConfigurationFile(toolboxConfigPath)
	logErr(err)

	if b ==true {
		f,err := os.Open(toolboxConfigPath)
		logErr(err)
		defer f.Close()
		util.BodyToStruct(f,&toolboxConfigObject)
	}else{
		f,err := os.Create(toolboxConfigPath)
		logErr(err)
		defer f.Close()
		toolboxConfigObject = ToolboxConfig{
			File:make([]FileConfig,0),
		}
	}

	filePathMD5 = make(map[string]interface{},0)
	//读取文件夹下的文件
	filepath.Walk(FilePath, func(path string, info os.FileInfo, err error) error {
		if path == FilePath{
			return nil
		}else{
			name := util.GetFileName(path)
			filePathMD5[name] = nil
			return nil
		}
	})

	messageQueueInitialization()
}

/**
检查配置文件,不合适就panic程序(只有既存在又不是文件夹的时候才返回true)
 */
func checkConfigurationFile(path string)(bool,error){
	f, err := os.Stat(path)
	if err == nil {
		if f.IsDir() {
			return false,nil
		}else{
			//只有既存在又不是文件夹的时候才返回true
			return true, nil
		}
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

/**
消息队列初始化
 */
func messageQueueInitialization(){
	util.DeepCopy(&filePathMD5Copy,filePathMD5)
	util.DeepCopy(&toolboxConfigObjectCopy,toolboxConfigObject)

	//监听ToolboxConfigObject出队列是否有空位,有就加入一个
	go addDepartureToolboxConfigObject()
	//监听FilePathMD5的出队列
	go addDepartureFilePathMD5()
	//监听队列,是否有配置队列添加操作
	go toolboxConfigObjectAdd()
	//监听队列,是否有Filemd5添加操作
	go filePathMD5Add()
}

func addDepartureToolboxConfigObject(){
	for ;true;  {
		departureToolboxConfigObject <- toolboxConfigObjectCopy
	}
}

func addDepartureFilePathMD5(){
	for ;true;  {
		departureFilePathMD5 <- filePathMD5Copy
	}
}

func toolboxConfigObjectAdd(){
	toolboxConfig := make(map[string]string,0)
	for ;true;  {
		tbco := <- AddToolboxConfigObject
		if toolboxConfig[tbco.FilePath] == ""{
			toolboxConfig[tbco.FilePath] = "用于filename去重"
			toolboxConfigObject.File = append(toolboxConfigObject.File,tbco)
			//将配置数据刷入硬盘
			refreshConfiguration()

			util.DeepCopy(&toolboxConfigObjectCopy,toolboxConfigObject)
		}
	}
}

/**
将配置数据刷入硬盘文件
 */
func refreshConfiguration(){
	b,err := json.Marshal(toolboxConfigObject)
	logErr(err)
	fw,err := os.Create(toolboxConfigPath)
	logErr(err)
	defer fw.Close()
	_,err = fw.Write(b)
	logErr(err)
}

func filePathMD5Add(){
	for ;true;  {
		pm5 := <- AddFileMD5
		filePathMD5[pm5] = nil
		util.DeepCopy(&filePathMD5Copy,filePathMD5)
	}
}

//err打印到日志文件
func logErr(err error){
	if err != nil {
		log.Println(err)
	}
}