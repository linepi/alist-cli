package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"crypto/tls"
	"os"
	"mime/multipart"
	"io"
	"net/http"
	"time"
	"path/filepath"
)

type Client struct {
	base     string //Alist API base url
	token    string //Alist API access token
	username string //Alist username
	password string //Alist password
	inscure  bool   //Skip TLS verification
	timeout  int    //Request timeout
}

const (
	DEFAULT_TIMEOUT   = 30
)

func do(method string, url string, body io.Reader, token string, options *RequestOptions) ([]byte, error) {
	client := &http.Client{
		Timeout: time.Duration(func() int {
			if options.Timeout > 0 {
				return options.Timeout
			} else {
				return DEFAULT_TIMEOUT
			}
		}()) * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: options.Insecure},
		},
	}
	request, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json; charset=utf-8")
	request.Header.Set("Authorization", token)
	resp, err := client.Do(request)
	if err != nil {
		if options.RetryTimes < options.MaxRetryTimes {
			options.RetryTimes++
			time.Sleep(time.Duration(options.RetryIntervalSecond) * time.Second)
			return do(method, url, body, token, options) // retry
		}
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// NewClient creates a new instance of the Client struct.
//
// Parameters:
// - endpoint: the API endpoint.
// - username: the username for authentication.
// - password: the password for authentication.
// - insecure: whether to ignore SSL certificate verification.
// - timeout: the timeout in seconds for API requests.
//
// Returns:
// - a pointer to the Client struct.
func NewClient(endpoint, username, password string, insecure bool, timeout int) *Client {
	return &Client{
		base:     endpoint,
		username: username,
		password: password,
		inscure:  insecure,
		timeout:  timeout,
	}
}

// NewClientWithToken creates a new instance of the Client struct with a token.
//
// Parameters:
// - endpoint: the API endpoint.
// - token: the access token for authentication.
// - insecure: whether to ignore SSL certificate verification.
// - timeout: the timeout in seconds for API requests.
//
// Returns:
// - a pointer to the Client struct.
func NewClientWithToken(endpoint, token string, insecure bool, timeout int) *Client {
	return &Client{
		base:    endpoint,
		token:   token,
		inscure: insecure,
		timeout: timeout,
	}
}

// Login performs the login operation for the Client.
//
// It sends a POST request to the "/api/auth/login" endpoint with the provided
// username and password in the request body as JSON. If successful, it
// extracts the JWT token from the response and sets it as the authorization
// header for future requests. Then, it sends a GET request to the "/api/me"
// endpoint to retrieve the user information. If successful, it returns the
// user information as a *User object.
//
// Returns:
// - *User: The user information if the login is successful.
// - error: An error if the login or user information retrieval fails.
func (c *Client) Login() (*User, error) {
	if !c.isLogin() {
		body := `{"username": "` + c.username + `","password": "` + c.password + `"}`
		respByts, err := do("POST", c.base+"/api/auth/login", bytes.NewBufferString(body), c.token, &RequestOptions{
			Insecure:            c.inscure,
			Timeout:             c.timeout,
			RetryTimes:          0,
			MaxRetryTimes:       3,
			RetryIntervalSecond: 5,
		})
		if err != nil {
			return nil, err
		}
		loginResp := &LoginResp{}
		err = json.Unmarshal(respByts, loginResp)
		if err != nil {
			return nil, err
		}
		if loginResp.Code != 200 {
			return nil, errors.New(loginResp.Message)
		}
		c.token = loginResp.Data.Token
	}

	respByts, err := do("GET", c.base+"/api/me", nil, c.token, &RequestOptions{
		Insecure:            c.inscure,
		Timeout:             c.timeout,
		RetryTimes:          0,
		MaxRetryTimes:       3,
		RetryIntervalSecond: 5,
	})
	if err != nil {
		return nil, err
	}
	userResp := &UserInfoResp{}
	err = json.Unmarshal(respByts, userResp)
	if err != nil {
		return nil, err
	}
	if userResp.Code != 200 {
		return nil, errors.New(userResp.Message)
	}
	u := &User{}
	u = userResp.Data
	return u, nil
}

func (c *Client) isLogin() bool {
	return c.token != ""
}

// MkDir creates a directory at the specified path.
//
// path: the path of the directory to be created.
// error: returns an error if the directory creation fails.
func (c *Client) MkDir(path string) error {
	if !c.isLogin() {
		return errors.New("not login yet")
	}
	body := `{
        "path": "` + path + `"
    }`

	respByts, err := do("POST", c.base+"/api/fs/mkdir", bytes.NewBufferString(body), c.token, &RequestOptions{
		Insecure:            c.inscure,
		Timeout:             c.timeout,
		RetryTimes:          0,
		MaxRetryTimes:       3,
		RetryIntervalSecond: 5,
	})
	if err != nil {
		return err
	}
	comResp := &CommonResp{}
	err = json.Unmarshal(respByts, comResp)
	if err != nil {
		return err
	}
	if comResp.Code != 200 {
		return errors.New(comResp.Message)
	}
	return nil
}

// Rename renames a file or directory to a new name.
//
// Parameters:
// - newName: the new name to rename to (string)
// - path: the path of the file or directory to rename (string)
//
// Returns:
// - error: an error if the renaming process fails (error)
func (c *Client) Rename(newName, path string) error {
	if !c.isLogin() {
		return errors.New("not login yet")
	}
	body := `{
        "name": "` + newName + `",
        "path": "` + path + `"
    }`
	respByts, err := do("POST", c.base+"/api/fs/rename", bytes.NewBufferString(body), c.token, &RequestOptions{
		Insecure:            c.inscure,
		Timeout:             c.timeout,
		RetryTimes:          0,
		MaxRetryTimes:       3,
		RetryIntervalSecond: 5,
	})
	if err != nil {
		return err
	}
	comResp := &CommonResp{}
	err = json.Unmarshal(respByts, comResp)
	if err != nil {
		return err
	}
	if comResp.Code != 200 {
		return errors.New(comResp.Message)
	}
	return nil
}

// Remove removes the specified files or directories from the client's filesystem.
//
// Parameters:
//   - dir (string): The directory where the files or directories are located.
//   - names ([]string): The names of the files or directories to be removed.
//
// Returns:
//   - error: An error if the removal operation fails, nil otherwise.
func (c *Client) Remove(dir string, names []string) error {
	if !c.isLogin() {
		return errors.New("not login yet")
	}
	body := `{
        "names": ["` + strings.Join(names, `","`) + `"],
        "dir": "` + dir + `"
    }`

	respByts, err := do("POST", c.base+"/api/fs/remove", bytes.NewBufferString(body), c.token, &RequestOptions{
		Insecure:            c.inscure,
		Timeout:             c.timeout,
		RetryTimes:          0,
		MaxRetryTimes:       3,
		RetryIntervalSecond: 5,
	})
	if err != nil {
		return err
	}
	comResp := &CommonResp{}
	err = json.Unmarshal(respByts, comResp)
	if err != nil {
		return err
	}
	if comResp.Code != 200 {
		return errors.New(comResp.Message)
	}
	return nil
}

// RemoveEmptyDir removes an empty directory.
//
// The `dir` parameter is a string that represents the directory path to be removed.
// It returns an error if there was a problem removing the directory.
func (c *Client) RemoveEmptyDir(dir string) error {
	if !c.isLogin() {
		return errors.New("not login yet")
	}
	body := `{
        "src_dir": "` + dir + `"
    }`

	respByts, err := do("POST", c.base+"/api/fs/remove_empty_directory", bytes.NewBufferString(body), c.token, &RequestOptions{
		Insecure:            c.inscure,
		Timeout:             c.timeout,
		RetryTimes:          0,
		MaxRetryTimes:       3,
		RetryIntervalSecond: 5,
	})
	if err != nil {
		return err
	}
	comResp := &CommonResp{}
	err = json.Unmarshal(respByts, comResp)
	if err != nil {
		return err
	}
	if comResp.Code != 200 {
		return errors.New(comResp.Message)
	}
	return nil
}

// Copy copies files from source directory to destination directory.
//
// srcDir: the source directory from which files will be copied.
// destDir: the destination directory to which files will be copied.
// names: a slice of string containing the names of the files to be copied.
// error: an error indicating any failure during the copy operation.
func (c *Client) Copy(srcDir, destDir string, names []string) error {
	if !c.isLogin() {
		return errors.New("not login yet")
	}
	body := `{
        "src_dir": "` + srcDir + `",
        "dst_dir": "` + destDir + `",
        "names": ["` + strings.Join(names, `","`) + `"]
    }`

	respByts, err := do("POST", c.base+"/api/fs/copy", bytes.NewBufferString(body), c.token, &RequestOptions{
		Insecure:            c.inscure,
		Timeout:             c.timeout,
		RetryTimes:          0,
		MaxRetryTimes:       3,
		RetryIntervalSecond: 5,
	})
	if err != nil {
		return err
	}
	comResp := &CommonResp{}
	err = json.Unmarshal(respByts, comResp)
	if err != nil {
		return err
	}
	if comResp.Code != 200 {
		return errors.New(comResp.Message)
	}
	return nil
}



// RecursiveMove 递归移动
// srcDir: 源目录
// destDir: 目标目录
func (c *Client) RecursiveMove(srcDir, destDir string) error {
	if !c.isLogin() {
		return errors.New("not login yet")
	}
	body := `{
        "src_dir": "` + srcDir + `",
        "dst_dir": "` + destDir + `"
    }`

	respByts, err := do("POST", c.base+"/api/fs/recursive_move", bytes.NewBufferString(body), c.token, &RequestOptions{
		Insecure:            c.inscure,
		Timeout:             c.timeout,
		RetryTimes:          0,
		MaxRetryTimes:       3,
		RetryIntervalSecond: 5,
	})
	if err != nil {
		return err
	}
	comResp := &CommonResp{}
	err = json.Unmarshal(respByts, comResp)
	if err != nil {
		return err
	}
	if comResp.Code != 200 {
		return errors.New(comResp.Message)
	}
	return nil
}

// Move 移动文件
// srcDir: 源目录
// destDir: 目标目录
// names: 文件名
func (c *Client) Move(srcDir, destDir string, names []string) error {
	if !c.isLogin() {
		return errors.New("not login yet")
	}
	body := `{
        "src_dir": "` + srcDir + `",
        "dst_dir": "` + destDir + `",
        "names": ["` + strings.Join(names, `","`) + `"]
    }`

	respByts, err := do("POST", c.base+"/api/fs/move", bytes.NewBufferString(body), c.token, &RequestOptions{
		Insecure:            c.inscure,
		Timeout:             c.timeout,
		RetryTimes:          0,
		MaxRetryTimes:       3,
		RetryIntervalSecond: 5,
	})
	if err != nil {
		return err
	}
	comResp := &CommonResp{}
	err = json.Unmarshal(respByts, comResp)
	if err != nil {
		return err
	}
	if comResp.Code != 200 {
		return errors.New(comResp.Message)
	}
	return nil
}

func (c *Client) batchRename(endpoint, srcDir string, nameKV map[string]string) error {
	if !c.isLogin() {
		return errors.New("not login yet")
	}
	kvs := make([]string, len(nameKV))
	for k, v := range nameKV {
		kvs = append(kvs, `"`+k+`":"`+v+`"`)
	}

	body := `{
        "src_dir": "` + srcDir + `",
        "rename_objects": [{` + strings.Join(kvs, `,`) + `}]
    }`

	respByts, err := do("POST", c.base+endpoint, bytes.NewBufferString(body), c.token, &RequestOptions{
		Insecure:            c.inscure,
		Timeout:             c.timeout,
		RetryTimes:          0,
		MaxRetryTimes:       3,
		RetryIntervalSecond: 5,
	})
	if err != nil {
		return err
	}
	comResp := &CommonResp{}
	err = json.Unmarshal(respByts, comResp)
	if err != nil {
		return err
	}
	if comResp.Code != 200 {
		return errors.New(comResp.Message)
	}
	return nil
}

// RegexRename 正则重命名
// srcDir: 源目录
// srcName: 源文件名正则匹配表达式
// newName: 新文件名正则引用表达式
func (c *Client) RegexRename(srcDir string, regexKV map[string]string) error {
	return c.batchRename("/api/fs/regex_rename", srcDir, regexKV)
}

// BatchRename 批量重命名
// BatchRename renames multiple files in a given source directory.
//
// Parameters:
// - srcDir: the source directory where the files are located.
// - batchKV: a map containing the source file names as keys and the new file names as values.
//
// Returns:
// - error: an error if the batch rename operation fails.
func (c *Client) BatchRename(srcDir string, batchKV map[string]string) error {
	return c.batchRename("/api/fs/batch_rename", srcDir, batchKV)
}

// Dirs 获取目录
// Dirs retrieves a list of directories from the server.
//
// It takes the following parameters:
// - path: the path of the directory to retrieve.
// - dirPassword: the password for the directory.
// - forceRoot: a flag indicating whether to the root directory.
//
// It returns a slice of Dir structs and an error.
func (c *Client) Dirs(path, dirPassword string, forceRoot bool) ([]Dir, error) {
	if !c.isLogin() {
		return nil, errors.New("not login yet")
	}
	body := `{
        "path": "` + path + `",
        "password": "` + dirPassword + `",
        "force_root": ` + strconv.FormatBool(forceRoot) + `
    }`

	respByts, err := do("POST", c.base+"/api/fs/dirs", bytes.NewBufferString(body), c.token, &RequestOptions{
		Insecure:            c.inscure,
		Timeout:             c.timeout,
		RetryTimes:          0,
		MaxRetryTimes:       3,
		RetryIntervalSecond: 5,
	})
	if err != nil {
		return nil, err
	}
	dirsResp := &DirsResp{}
	err = json.Unmarshal(respByts, dirsResp)
	if err != nil {
		return nil, err
	}
	if dirsResp.Code != 200 {
		return nil, errors.New(dirsResp.Message)
	}
	return dirsResp.Data, nil
}

// List 列出文件目录
// List returns a list of files in the specified directory.
//
// Parameters:
// - path: the path of the directory.
// - dirPassword: the password of the directory (if applicable).
// - pageNum: the page number for pagination.
// - pageSize: the number of files per page.
// - refresh: indicates whether to refresh the file list.
//
// Returns:
// - a slice of File structs representing the files in the directory.
// - an error if any error occurs.
func (c *Client) List(path, dirPassword string, pageNum, pageSize int, refresh bool) ([]File, error) {
	if !c.isLogin() {
		return nil, errors.New("not login yet")
	}
	body := `{
        "path": "` + path + `",
        "password": "` + dirPassword + `",
        "page_num": ` + strconv.Itoa(pageNum) + `,
        "per_page": ` + strconv.Itoa(pageSize) + `,
        "refresh": ` + strconv.FormatBool(refresh) + `
    }`

	respByts, err := do("POST", c.base+"/api/fs/list", bytes.NewBufferString(body), c.token, &RequestOptions{
		Insecure:            c.inscure,
		Timeout:             c.timeout,
		RetryTimes:          0,
		MaxRetryTimes:       3,
		RetryIntervalSecond: 5,
	})
	if err != nil {
		return nil, err
	}
	listResp := &ListResp{}
	err = json.Unmarshal(respByts, listResp)
	if err != nil {
		return nil, err
	}
	if listResp.Code != 200 {
		return nil, errors.New(listResp.Message)
	}
	return listResp.Data.Content, nil
}

// Get 获取某个文件/目录信息
// Get retrieves a file or directory from the server.
//
// Parameters:
// - path: the path of the file or directory.
// - dirPassword: the password of the directory (if applicable).
//
// Returns:
// - a File struct representing the file or directory.
// - an error if any error occurs.
func (c *Client) Get(path, dirPassword string) (*File, error) {
	if !c.isLogin() {
		return nil, errors.New("not login yet")
	}
	body := `{
        "path": "` + path + `",
        "password": "` + dirPassword + `"
    }`

	respByts, err := do("POST", c.base+"/api/fs/get", bytes.NewBufferString(body), c.token, &RequestOptions{
		Insecure:            c.inscure,
		Timeout:             c.timeout,
		RetryTimes:          0,
		MaxRetryTimes:       3,
		RetryIntervalSecond: 5,
	})
	if err != nil {
		return nil, err
	}
	getResp := &GetResp{}
	err = json.Unmarshal(respByts, getResp)
	if err != nil {
		return nil, err
	}
	if getResp.Code != 200 {
		return nil, errors.New(getResp.Message)
	}
	return getResp.Data, nil
}

// GetSettings 获取设置
// GetSettings retrieves the settings from the client.
//
// This function does not take any parameters.
// It returns a pointer to Settings and an error.
func (c *Client) GetSettings() (*Settings, error) {
	if !c.isLogin() {
		return nil, errors.New("not login yet")
	}
	respByts, err := do("GET", c.base+"/api/settings", nil, c.token, &RequestOptions{
		Insecure:            c.inscure,
		Timeout:             c.timeout,
		RetryTimes:          0,
		MaxRetryTimes:       3,
		RetryIntervalSecond: 5,
	})
	if err != nil {
		return nil, err
	}
	settingsResp := &SettingsResp{}
	err = json.Unmarshal(respByts, settingsResp)
	if err != nil {
		return nil, err
	}
	if settingsResp.Code != 200 {
		return nil, errors.New(settingsResp.Message)
	}
	return settingsResp.Data, nil
}

// Upload 方法用于通过 PUT 表单上传文件
func (c *Client) Upload(filePath string, file *os.File) error {
	if !c.isLogin() {
		return errors.New("not login yet")
	}

	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}

	method := "PUT"
	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)

	part, err := writer.CreateFormFile("file", fileInfo.Name())
    if err != nil {
        return err
    }
	_, err = io.Copy(part, file)
	if err != nil {
        return err
    }

	err = writer.Close()
	if err != nil {
		return err
	}

	client := &http.Client {}
	req, err := http.NewRequest(method, c.base+"/api/fs/form", payload)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", c.token)
	req.Header.Add("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))
	req.Header.Add("File-Path", filepath.Join(filePath, filepath.Base(file.Name())))
	req.Header.Add("As-Task", "true")
	req.Header.Add("Content-Type", writer.FormDataContentType())

	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	respByts, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	uploadResp := &UploadResp{}
	err = json.Unmarshal(respByts, uploadResp)
	if err != nil {
		return err
	}
	if uploadResp.Code != 200 {
		return errors.New(uploadResp.Message)
	}

	return nil
}

func (c *Client) Download(filePath string, file *os.File) error {
	if !c.isLogin() {
		return errors.New("not login yet")
	}

	method := "GET"
	client := &http.Client {}
	req, err := http.NewRequest(method, c.base+filePath, nil)
	if err != nil {
		return err
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

    _, err = io.Copy(file, res.Body)
    if err != nil {
        return err
    }

    if err := file.Sync(); err != nil {
        return err
    }
	return nil
}