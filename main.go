package main

import (
	"fmt"
	"os"
	"strings"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var (
	client *Client
	logger *logrus.Logger
)

const (
	VERSION = "v0.1.1"
)

// Direct configuration settings
var (
	endpoint   = "" // Set your Alist endpoint here, need http(https)
	username   = "" // Set your username here
	password   = "" // Set your password here
	skipVerify = true // Set to false if you want to verify SSL certificates
)

func main() {
	logger = logrus.New()
	logger.SetLevel(logrus.WarnLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
	app := cli.NewApp()
	app.Name = "alist-cli"
	app.Description = "alist command line interface"
	app.Version = VERSION
	app.EnableBashCompletion = true
	app.Flags = []cli.Flag{}
	app.HideVersion = true
	app.Commands = []*cli.Command{
		{
			Name:  "version",
			Usage: "show alist-cli version",
			Action: func(c *cli.Context) error {
				fmt.Printf("version: %s\n", VERSION)
				return nil
			},
		},
		{ // move
			Name:      "move",
			Aliases:   []string{"mv", "rename"},
			Usage:     "move file from src to dst, cannot cross mount point",
			UsageText: fmt.Sprintf(`%s move [-v] src-path dst-folder`, app.Name),
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:    "verbose",
					Aliases: []string{"v"},
					Usage:   "verbose mode, show more infomation",
					Value:   false,
					Action: func(c *cli.Context, v bool) error {
						if v {
							logger.SetLevel(logrus.InfoLevel)
							logger.Info("verbose mode enabled")
						}
						return nil
					},
				},
			},
			Action: func(c *cli.Context) error {
				client = NewClient(endpoint, username, password, skipVerify, 30)
				u, err := client.Login()
				if err != nil {
					return fmt.Errorf("login failed: %v", err)
				}
				logger.Infof("login success: %v", u.Username)
				src := c.Args().Get(0)
				dst := c.Args().Get(1)
				if !CheckPathIsExist(src) {
					return fmt.Errorf("path %s not exist", src)
				}
				if !CheckPathIsFolder(dst) {
					return fmt.Errorf("path %s is not a folder", dst)
				}
				//split src path to folder and filename
				pathParts := strings.Split(src, "/")
				folder := "/" + strings.Join(pathParts[:len(pathParts)-1], "/")
				filename := pathParts[len(pathParts)-1]
				err = client.Move(folder, dst, []string{filename})
				if err != nil {
					return fmt.Errorf("move %s to %s failed: %v", src, dst, err)
				}
				logger.Infof("move %s to %s success", src, dst)
				return nil
			},
		},
		{ // copy
			Name:      "copy",
			Aliases:   []string{"cp"},
			Usage:     "copy file from src to dst, can cross mount point",
			UsageText: fmt.Sprintf(`%s copy [-v] src-path dst-folder`, app.Name),
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:    "verbose",
					Aliases: []string{"v"},
					Usage:   "verbose mode, show more infomation",
					Value:   false,
					Action: func(c *cli.Context, v bool) error {
						if v {
							logger.SetLevel(logrus.InfoLevel)
						}
						return nil
					},
				},
			},
			Action: func(c *cli.Context) error {
				client = NewClient(endpoint, username, password, skipVerify, 30)
				u, err := client.Login()
				if err != nil {
					return fmt.Errorf("login failed: %v", err)
				}
				logger.Infof("login success: %v", u.Username)
				src := c.Args().Get(0)
				dst := c.Args().Get(1)
				dst = strings.TrimSuffix(dst, "/")
				if !CheckPathIsExist(src) {
					return fmt.Errorf("path %s not exist", src)
				}
				pathParts := strings.Split(src, "/")
				folder := "/" + strings.Join(pathParts[:len(pathParts)-1], "/")
				filename := pathParts[len(pathParts)-1]
				if CheckPathIsExist(dst + "/" + filename) {
					return fmt.Errorf("dst path %s exist", dst+"/"+filename)
				}
				if !CheckPathIsFolder(dst) {
					return fmt.Errorf("path %s is not a folder", dst)
				}
				err = client.Copy(folder, dst, []string{filename})
				if err != nil {
					return fmt.Errorf("copy %s to %s failed: %v", src, dst, err)
				}
				logger.Infof("copy %s to %s mission submit success", src, dst)
				return nil
			},
		},
		{ // list
			Name:      "list",
			Aliases:   []string{"ls"},
			Usage:     "list all files in path",
			UsageText: fmt.Sprintf(`%s list [-v] path`, app.Name),
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:    "verbose",
					Aliases: []string{"v"},
					Usage:   "verbose mode, show more infomation",
					Value:   false,
					Action: func(c *cli.Context, v bool) error {
						if v {
							logger.SetLevel(logrus.InfoLevel)
						}
						return nil
					},
				},
			},
			Action: func(c *cli.Context) error {
				client = NewClient(endpoint, username, password, skipVerify, 30)
				u, err := client.Login()
				if err != nil {
					return fmt.Errorf("login failed: %v", err)
				}
				logger.Infof("login success: %v", u.Username)
				path := c.Args().Get(0)
				files, err := client.List(path, "", 0, 0, true)
				if err != nil {
					return fmt.Errorf("list %s failed: %v", path, err)
				}
				fmt.Printf("Files in %s:\n", path)
				for _, f := range files {
					fmt.Printf("%-30.30s\t%6.6s\n", f.Name, func() string {
						if f.IsDir {
							return "folder"
						}
						return "file"
					}())
				}
				logger.Infof("list %s success, get %d files", path, len(files))
				return nil
			},
		},
		{ // delete
			Name:      "delete",
			Aliases:   []string{"rm"},
			Usage:     "delete file from path",
			UsageText: fmt.Sprintf(`%s delete [-v] path`, app.Name),
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:    "verbose",
					Aliases: []string{"v"},
					Usage:   "verbose mode, show more infomation",
					Value:   false,
					Action: func(c *cli.Context, v bool) error {
						if v {
							logger.SetLevel(logrus.InfoLevel)
						}
						return nil
					},
				},
			},
			Action: func(c *cli.Context) error {
				client = NewClient(endpoint, username, password, skipVerify, 30)
				u, err := client.Login()
				if err != nil {
					return fmt.Errorf("login failed: %v", err)
				}
				logger.Infof("login success: %v", u.Username)
				path := c.Args().Get(0)
				if !CheckPathIsExist(path) {
					return fmt.Errorf("path %s not exist", path)
				}
				pathParts := strings.Split(path, "/")
				folder := "/" + strings.Join(pathParts[:len(pathParts)-1], "/")
				filename := pathParts[len(pathParts)-1]
				err = client.Remove(folder, []string{filename})
				if err != nil {
					return fmt.Errorf("delete %s failed: %v", path, err)
				}
				logger.Infof("delete %s success", path)
				return nil
			},
		},
		{ // upload
			Name:      "upload",
			Usage:     "upload local file to alist server",
			UsageText: fmt.Sprintf(`%s upload [-v] local-file remote-path`, app.Name),
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:    "verbose",
					Aliases: []string{"v"},
					Usage:   "verbose mode, show more information",
					Value:   false,
					Action: func(c *cli.Context, v bool) error {
						if v {
							logger.SetLevel(logrus.InfoLevel)
						}
						return nil
					},
				},
			},
			Action: func(c *cli.Context) error {
				client = NewClient(endpoint, username, password, skipVerify, 30)
				_, err := client.Login()
				if err != nil {
					return fmt.Errorf("login failed: %v", err)
				}

				if c.NArg() < 2 {
					return fmt.Errorf("both local file and remote path are required")
				}

				localFile := c.Args().Get(0)
				remotePath := c.Args().Get(1)

				// Check if local file exists
				if _, statErr := os.Stat(localFile); statErr != nil {
					return fmt.Errorf("stat local file %s err: %v", localFile, statErr)
				}

				// Open the local file
				file, err := os.Open(localFile)
				if err != nil {
					return fmt.Errorf("failed to open local file: %v", err)
				}
				defer file.Close()

				if !CheckPathIsFolder(remotePath) {
					return fmt.Errorf("remote path %s invalid", remotePath)
				}

				remoteFilePath := filepath.Join(remotePath, filepath.Base(file.Name()))
				if CheckPathIsExist(remoteFilePath) {
					return fmt.Errorf("remote file %s is existed", remoteFilePath)
				}

				// Upload the file
				logger.Infof("uploading %s to %s", localFile, remotePath)
				err = client.Upload(remotePath, file)
				if err != nil {
					return fmt.Errorf("upload failed: %v", err)
				}

				logger.Infof("successfully uploaded %s to %s", file.Name(), remotePath)
				return nil
			},
		},
		{ // download
			Name:      "download",
			Usage:     "download file from alist server to local",
			UsageText: fmt.Sprintf(`%s download [-v] remote-file local-path`, app.Name),
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:    "verbose",
					Aliases: []string{"v"},
					Usage:   "verbose mode, show more information",
					Value:   false,
					Action: func(c *cli.Context, v bool) error {
						if v {
							logger.SetLevel(logrus.InfoLevel)
						}
						return nil
					},
				},
			},
			Action: func(c *cli.Context) error {
				client = NewClient(endpoint, username, password, skipVerify, 30)
				_, err := client.Login()
				if err != nil {
					return fmt.Errorf("login failed: %v", err)
				}

				if c.NArg() < 2 {
					return fmt.Errorf("both remote file and local path are required")
				}

				remoteFile := c.Args().Get(0)
				localPath := c.Args().Get(1)

				// Check if remote file exists
				if !CheckPathIsExist(remoteFile) {
					return fmt.Errorf("remote file %s does not exist", remoteFile)
				}

				// not support folder now
				if CheckPathIsFolder(remoteFile) {
					return fmt.Errorf("folder download not supported")
				}

				logger.Infof("downloading %s to %s", remoteFile, filepath.Dir(localPath))
				stat, err := os.Stat(localPath)
				if !stat.IsDir() {
					return fmt.Errorf("dst is invalid")
				}

				dstPath := filepath.Join(localPath, filepath.Base(remoteFile))
				// If localPath is a directory, use the remote filename
				if stat, serr := os.Stat(localPath); serr == nil && stat.IsDir() {
					localPath = dstPath
				}

				// Create the local file
				outFile, err := os.Create(localPath)
				if err != nil {
					return fmt.Errorf("failed to create local file: %v", err)
				}
				defer outFile.Close()

				// Download the file
				err = client.Download(remoteFile, outFile)
				if err != nil {
					return fmt.Errorf("download failed: %v", err)
				}

				logger.Infof("successfully downloaded %s to %s", remoteFile, localPath)
				return nil
			},
		},	
	}
	err := app.Run(os.Args)
	if err != nil {
		logger.Error(err)
	}
}

func CheckPathIsFolder(path string) bool {
	f, e := client.Get(path, "")
	if e != nil {
		logger.Errorf("get %s info failed with error: %v", path, e)
		return false
	}
	if !f.IsDir {
		return false
	}
	return true
}

func CheckPathIsExist(path string) bool {
	_, e := client.Get(path, "")
	if e != nil {
		return false
	}
	return true
}