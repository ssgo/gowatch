#### 监视文件并自动运行Go程序或测试用例



# 安装

```shell
go get github.com/ssgo/gowatch
go install github.com/ssgo/gowatch
```


# Usage

```shell
gowatch [-p paths] [-t] [-b] [...]
-p	指定监视的路径，默认为 ./，支持逗号隔开的多个路径
-r	执行当前目录中的程序，相当于 go run *.go
-t	执行tests目录中的测试用例，相当于 go test ./tests
-b	执行性能测试，相当于 go -bench .*，需要额外指定 -t 或 test 参数
...	可以使用所有 go 命令的参数
```

# Samples:

```shell
gowatch -r
gowatch -t
gowatch -t -b
gowatch -p ../ -t
gowatch run start.go
gowatch run samePackages start.go
gowatch test
gowatch test ./testcase
```