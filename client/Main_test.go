package main

import (
	"bytes"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestHandler(t *testing.T){
	buf := new(bytes.Buffer)
	mw := multipart.NewWriter(buf)
	assert.NoError(t, mw.WriteField("foo", "bar"))
	w, err := mw.CreateFormFile("file", "test")
	if assert.NoError(t, err) {

		file,err := os.Open("e:\\账号.txt")
		defer file.Close()
		io.Copy(w,file)
		//_, err = w.Write([]byte("test"))
		assert.NoError(t, err)
	}
	w, err = mw.CreateFormFile("file","haha")
	panicErr(err)

	file,err := os.Open("E:\\U盘备份\\弃用\\数据库定义.txt")
	defer file.Close()
	io.Copy(w,file)
	//w.Write([]byte("heiheihei"))
	mw.Close()
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request, _ = http.NewRequest("POST", "/", buf)
	c.Request.Header.Set("Content-Type", mw.FormDataContentType())
	f, err := c.MultipartForm()
	if assert.NoError(t, err) {
		assert.NotNil(t, f)
	}

	assert.NoError(t, c.SaveUploadedFile(f.File["file"][0], "test"))
	assert.NoError(t, c.SaveUploadedFile(f.File["file"][1], "test1"))
}

func TestExecuteDownloadFile(t *testing.T){
	EnvironmentalPreparation()
	ExecuteDownloadFile("草稿本.txt","/test2","E:/测试用的文件夹")
}