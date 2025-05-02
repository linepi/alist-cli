# Alist-cli  
Alist command line interface  
## 安装 

```
go install github.com/linepi/alist-cli@latest
```
## 使用

```
alist-cli.exe --help
NAME:
   alist-cli - A new cli application

USAGE:
   alist-cli [global options] command [command options] [arguments...]

DESCRIPTION:
   alist command line interface

COMMANDS:
   version           show alist-cli version
   move, mv, rename  move file from src to dst, cannot cross mount point
   copy, cp          copy file from src to dst, can cross mount point
   list, ls          list all files in path
   delete, rm        delete file from path
   upload            upload file to path, imcomplete not supported yet
   download          download file from alist server to local
   help, h           Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h  show help
```
## 作者
[@linepi](https://github.com/linepi)
## 感谢  
[Alist](https://github.com/alist-org/alist)
[@Alist-cli](https://github.com/imshuai/Alist-cli)
[@alistsdk-go](https://github.com/imshuai/alistsdk-go)