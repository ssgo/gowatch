package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var filesModTime = make(map[string]int64)

func main() {
	if len(os.Args) == 1 {
		printUsage()
		return
	}

	basePaths := make([]string, 0)
	cmd := "go"
	cmdArgs := make([]string, 0)
	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "help":
		case "--help":
			printUsage()
			return
		case "-p":
			i++
			tmpPaths := strings.Split(os.Args[i], ",")
			for _, path := range tmpPaths {
				if []byte(path)[len(path)-1] != '/' {
					path += "/"
				}
				basePaths = append(basePaths, path)
			}
		case "-sh":
			if i < len(os.Args)-1 && os.Args[i+1][0] != '-' {
				i++
				cmd = os.Args[i]
			} else {
				cmd = "sh"
			}
		case "-r":
			cmdArgs = append(cmdArgs, "run")
			files, err := ioutil.ReadDir("./")
			if err == nil {
				for _, file := range files {
					if !strings.HasPrefix(file.Name(), ".") && !strings.HasSuffix(file.Name(), "_test.go") && strings.HasSuffix(file.Name(), ".go") {
						cmdArgs = append(cmdArgs, "./"+file.Name())
					}
				}
			}
		case "-t":
			cmdArgs = append(cmdArgs, "test", "./tests")
		case "-b":
			cmdArgs = append(cmdArgs, "-bench", ".*")
		default:
			cmdArgs = append(cmdArgs, os.Args[i])
		}
	}

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		if lastCmd != nil {
			fmt.Println("killing ", lastCmd.Process.Pid)
			syscall.Kill(-lastCmd.Process.Pid, syscall.SIGKILL)
		}
		fmt.Println("\nExit")
		os.Exit(0)
	}()

	if len(basePaths) == 0 {
		basePaths = append(basePaths, "./")
	}

	//os.Stdout.WriteString("\x1b[3;J\x1b[H\x1b[2J")
	//fmt.Printf("[Watching \033[36m%s\033[0m] [Running \033[36mgo %s\033[0m]\n\n", strings.Join(basePaths, " "), strings.Join(cmdArgs, " "))
	//runCommand("go", cmdArgs...)

	changed := make(chan bool)
	go func(changed chan bool) {
		for {
			if watchFiles() {
				if lastCmd != nil {
					fmt.Println("killing ", lastCmd.Process.Pid)
					syscall.Kill(-lastCmd.Process.Pid, syscall.SIGKILL)
				}
				changed <- true
			}
			time.Sleep(time.Millisecond * 100)
		}
	}(changed)

	go func() {
		for {
			for _, path := range basePaths {
				watchPath(path)
			}
			time.Sleep(time.Second * 3)
		}
	}()

	for {
		select {
		case <-changed:
			os.Stdout.WriteString("\x1b[3;J\x1b[H\x1b[2J")
			fmt.Printf("[Watching \033[36m%s\033[0m] [Running \033[36m%s %s\033[0m]\n\n", strings.Join(basePaths, " "), cmd, strings.Join(cmdArgs, " "))
			runCommand(cmd, cmdArgs...)
		}
	}
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("	gowatch \033[37m[-p paths] [-t] [-b] [...]\033[0m")
	fmt.Println("	\033[36m-p\033[0m	\033[37m指定监视的路径，默认为 ./，支持逗号隔开的多个路径\033[0m")
	fmt.Println("	\033[36m-sh\033[0m	\033[37m指定执行的命令，默认为 go\033[0m")
	fmt.Println("	\033[36m-r\033[0m	\033[37m执行当前目录中的程序，相当于 go run *.go\033[0m")
	fmt.Println("	\033[36m-t\033[0m	\033[37m执行tests目录中的测试用例，相当于 go test ./tests\033[0m")
	fmt.Println("	\033[36m-b\033[0m	\033[37m执行性能测试，相当于 go -bench .*，需要额外指定 -t 或 test 参数\033[0m")
	fmt.Println("	\033[36m...\033[0m	\033[37m可以使用所有 go 命令的参数\033[0m")
	fmt.Println("")
	fmt.Println("Samples:")
	fmt.Println("	\033[36mgowatch -r\033[0m")
	fmt.Println("	\033[36mgowatch -t\033[0m")
	fmt.Println("	\033[36mgowatch -t -b\033[0m")
	fmt.Println("	\033[36mgowatch -p ../ -t\033[0m")
	fmt.Println("	\033[36mgowatch run start.go\033[0m")
	fmt.Println("	\033[36mgowatch run samePackages start.go\033[0m")
	fmt.Println("	\033[36mgowatch test\033[0m")
	fmt.Println("	\033[36mgowatch test ./testcase\033[0m")
	fmt.Println("")
}

var lastCmd *exec.Cmd = nil

func runCommand(command string, args ...string) {
	cmd := exec.Command(command, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	lastCmd = cmd
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println(err)
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Println(err)
		return
	}

	cmd.Start()
	reader := bufio.NewReader(io.MultiReader(stdout, stderr))
	for {
		lineBuf, _, err2 := reader.ReadLine()

		if err2 != nil || io.EOF == err2 {
			break
		}
		line := strings.TrimRight(string(lineBuf), "\r\n")
		if strings.HasPrefix(line, "ok ") {
			fmt.Println("\033[42m", line, "\033[0m")
		} else if strings.HasPrefix(line, "FAIL	") {
			fmt.Println("\033[41m", line, "\033[0m")
		} else if strings.Index(line, ".go:") != -1 {
			if strings.Index(line, "/usr") != -1 {
				fmt.Println(line)
			} else {
				fmt.Println("\033[36m", line, "\033[0m")
			}
		} else if strings.HasPrefix(line, "	") {
			fmt.Println(line)
		} else {
			fmt.Println("\033[37m", line, "\033[0m")
		}
	}

	cmd.Wait()
}

func watchFiles() bool {
	changed := false
	for fileName, modTime := range filesModTime {
		info, err := os.Stat(fileName)
		if err != nil {
			delete(filesModTime, fileName)
			continue
		}
		if info.ModTime().Unix() != modTime {
			filesModTime[fileName] = info.ModTime().Unix()
			changed = true
		}
	}
	return changed
}

func watchPath(path string) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return
	}
	for _, file := range files {
		fileBytes := []byte(file.Name())
		if fileBytes[0] == '.' {
			continue
		}
		if file.IsDir() {
			watchPath(path + file.Name() + "/")
		} else {
			l := len(fileBytes)
			if l < 4 || fileBytes[l-3] != '.' || fileBytes[l-2] != 'g' || fileBytes[l-1] != 'o' {
				continue
			}
			if filesModTime[path+file.Name()] == 0 {
				filesModTime[path+file.Name()] = 1
			}
		}
	}
}
