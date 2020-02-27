package raasr

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

var (
	ch string = "aaaaaaaaa`"
)

type Conn struct {
	c    *http.Client
	conf *Conf
}

type Client struct {
	conn *Conn
}

type RespInfo struct {
	Ok     int    `json:"ok"`
	ErrNo  int    `json:"err_no"`
	Failed string `json:"failed"`
	Data   string `json:"data"`
}

//初始化
func New(appID, secretKey string) *Client {
	client := Client{}
	conf := getDefaultConf()
	conf.AppID = appID
	conf.SecretKey = secretKey

	conn := Conn{&http.Client{}, conf}
	client.conn = &conn

	return &client
}

func (c *Client) UploadAudio(filename, language string) (taskid string, err error) {
	filesize, sliceNum, err := c.conn.getSizeAndSiceNum(filename)
	if err != nil {
		return
	}

	taskid, err = c.initSliceUpload(filename, language, filesize, sliceNum)
	if err != nil {
		return
	}

	if err = c.performSliceUpload(filename, taskid, filesize, sliceNum); err != nil {
		return
	}

	if err = c.completeSliceUpload(taskid); err != nil {
		return
	}

	return
}

//预处理，初始化上传
func (c *Client) initSliceUpload(filename, language string, filesize, sliceNum int64) (taskid string, err error) {
	var info RespInfo
	params := c.getBaseAuthParam("")
	params.Add("file_len", strconv.FormatInt(filesize, 10))
	params.Add("file_name", filename)
	params.Add("language", language)
	params.Add("slice_num", strconv.FormatInt(sliceNum, 10))

	resp, err := c.conn.httpDo(c.conn.conf.Domain+"/prepare", nil, params, nil)
	if err != nil {
		return
	}

	if err = json.Unmarshal([]byte(resp), &info); err != nil {
		return
	}

	if info.Ok == 0 {
		taskid = info.Data
	} else {
		err = fmt.Errorf(info.Failed)
	}

	return
}
func (c *Client) performSliceUpload(filename, taskid string, filesize, sliceNum int64) (err error) {
	var info RespInfo
	fi, err := os.OpenFile(filename, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return
	}
	defer fi.Close()

	b := make([]byte, c.conn.conf.PartSize)
	for i := int64(1); i <= sliceNum; i++ {
		fi.Seek((i-1)*c.conn.conf.PartSize, 0)
		if len(b) > int(filesize-(i-1)*c.conn.conf.PartSize) {
			b = make([]byte, filesize-(i-1)*c.conn.conf.PartSize)
		}
		fi.Read(b)

		params := c.getBaseAuthParam(taskid)
		params.Add("slice_id", c.getNextSliceId())
		resp, err := c.conn.postMulti(c.conn.conf.Domain+"/upload", filename, b, params)
		if err != nil {
			return err
		}

		if err := json.Unmarshal([]byte(resp), &info); err != nil {
			return err
		}

		if info.Ok != 0 {
			return fmt.Errorf(info.Failed)
		}
	}
	return nil
}

func (c *Client) completeSliceUpload(taskid string) (err error) {
	params := c.getBaseAuthParam(taskid)
	resp, err := c.conn.httpDo(c.conn.conf.Domain+"/merge", nil, params, nil)
	if err != nil {
		return
	}
	var info RespInfo
	if err = json.Unmarshal([]byte(resp), &info); err != nil {
		return
	}

	if info.Ok != 0 {
		return fmt.Errorf(info.Failed)
	}

	return nil
}

func (c *Client) doWorker(filename, taskid string, b []byte) (err error) {
	params := c.getBaseAuthParam(taskid)
	params.Add("slice_id", c.getNextSliceId())
	resp, err := c.conn.postMulti(c.conn.conf.Domain+"/upload", filename, b, params)
	if err != nil {
		return err
	}
	var info RespInfo
	if err := json.Unmarshal([]byte(resp), &info); err != nil {
		return err
	}

	if info.Ok != 0 {
		return fmt.Errorf(info.Failed)
	}

	return
}

func (c *Client) GetProgress(taskid string) (content string, err error) {
	params := c.getBaseAuthParam(taskid)
	resp, err := c.conn.httpDo(c.conn.conf.Domain+"/getProgress", nil, params, nil)
	if err != nil {
		return
	}
	var info RespInfo
	if err = json.Unmarshal([]byte(resp), &info); err != nil {
		return
	}

	if info.Ok != 0 {
		return info.Data, fmt.Errorf(info.Failed)
	}

	return info.Data, nil
}

func (c *Client) GetResult(taskid string) (content string, err error) {
	params := c.getBaseAuthParam(taskid)
	resp, err := c.conn.httpDo(c.conn.conf.Domain+"/getResult", nil, params, nil)
	if err != nil {
		return
	}
	var info RespInfo
	if err = json.Unmarshal([]byte(resp), &info); err != nil {
		return
	}

	if info.Ok != 0 {
		return info.Data, fmt.Errorf(info.Failed)
	}

	return info.Data, nil
}

func (c *Conn) postMulti(uri, filename string, content []byte, params url.Values) ([]byte, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("content", filename+params.Get("slice_id"))
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, bytes.NewBuffer(content))

	for key, val := range params {
		_ = writer.WriteField(key, val[0])
	}
	err = writer.Close()
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest("POST", uri, body)
	request.Header.Set("Content-Type", writer.FormDataContentType())

	res, err := c.c.Do(request)
	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(res.Body)
}

func (c *Conn) httpDo(url string, body []byte, params url.Values, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	if params != nil {
		req.URL.RawQuery = params.Encode()
	}
	if headers != nil {
		for key, val := range headers {
			req.Header.Add(key, val)
		}
	}
	resp, err := c.c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

func (c *Conn) getSizeAndSiceNum(filename string) (filesize, num int64, err error) {
	filesize, err = fileSize(filename)
	if err != nil {
		return
	}
	num = int64(math.Ceil(float64(filesize) / float64(c.conf.PartSize)))
	return
}

func (c *Client) getBaseAuthParam(taskid string) url.Values {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	mac := hmac.New(sha1.New, []byte(c.conn.conf.SecretKey))
	strByte := []byte(c.conn.conf.AppID + ts)
	strMd5Byte := md5.Sum(strByte)
	strMd5 := fmt.Sprintf("%x", strMd5Byte)
	mac.Write([]byte(strMd5))
	signa := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	params := url.Values{}
	params.Add("app_id", c.conn.conf.AppID)
	params.Add("signa", signa)
	params.Add("ts", ts)
	if len(taskid) > 0 {
		params.Add("task_id", taskid)
	}

	return params
}

func (c *Client) getNextSliceId() string {
	j := len(ch) - 1
	for i := j; i >= 0; {
		cj := string(ch[i])
		if cj != "z" {
			ch = string(ch[:i]) + string(ch[i]+1) + string(ch[i+1:])
			break
		} else {
			ch = string(ch[:i]) + "a" + string(ch[i+1:])
			i--
		}
	}
	return ch
}

func fileSize(filename string) (int64, error) {
	info, err := os.Stat(filename)
	if err != nil && os.IsNotExist(err) {
		return 0, err
	}
	return info.Size(), nil
}
