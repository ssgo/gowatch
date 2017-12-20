#### 监视文件并自动运行Go程序或测试用例



# 安装

```shell
go get github.com/ssgo/watch
go install github.com/ssgo/watch
```



# Usage

```shell
watch [ -p path ] [ -t ] [ -b | -bench name ] [ packages ] [ gofiles... ]
	-p	指定监视的路径，默认为 ./
	-t	执行测试用例 默认执行 go test ./tests，未指定时将运行 go run *.go
	-b	执行性能测试，默认执行所有，如需单独指定请使用 -bench name
```
