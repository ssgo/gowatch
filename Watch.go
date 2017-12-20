package main

import (
	"time"
	"io/ioutil"
	"fmt"
	"os"
	"os/exec"
	"bufio"
	"io"
	"strings"
)

var filesModTime = make(map[string]int64)

func main() {
	basePath := "./"
	cmdArgs := make([]string, 1)
	cmdArgs[0] = "run"
	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "help":
		case "--help":
			fmt.Println("Usage:")
			fmt.Println("	watch [-p path] [-t] [-b|-bench name] [packages] [gofiles...]")
			fmt.Println("	-p	指定监视的路径，默认为 ./")
			fmt.Println("	-t	执行测试用例 默认执行 go test ./tests，未指定时将运行 go run *.go")
			fmt.Println("	-b	执行性能测试，默认执行所有，如需单独指定请使用 -bench name")
			return
		case "-t":
			cmdArgs[0] = "test"
		case "-p":
			i++
			basePath = os.Args[i]
			if []byte(basePath)[len(basePath)-1] != '/' {
				basePath += "/"
			}
		case "-b":
			cmdArgs[0] = "test"
			cmdArgs = append(cmdArgs, " -bench", "'.*'")
		default:
			cmdArgs = append(cmdArgs, os.Args[i])
		}
	}

	if len(cmdArgs) == 1 {
		if cmdArgs[0] == "run" {
			cmdArgs = append(cmdArgs, "*.go")
		} else {
			cmdArgs = append(cmdArgs, "./tests")
		}
	}

	runCommand("go", cmdArgs...)

	changed := make(chan bool)
	go func(changed chan bool) {
		for {
			if watchFiles() {
				changed <- true
			}
			time.Sleep(time.Millisecond * 100)
		}
	}(changed)

	go func() {
		for {
			watchPath(basePath)
			time.Sleep(time.Second * 3)
		}
	}()

	for {
		select {
		case <-changed:
			runCommand("go", cmdArgs...)
		}
	}
}

func runCommand(command string, args ...string) {
	os.Stdout.WriteString("\x1b[3;J\x1b[H\x1b[2J")
	fmt.Printf("[\033[35mgo %s\033[0m]\n\n", strings.Join(args, " "))
	cmd := exec.Command(command, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println(err)
		return
	}
	stderr, err := cmd.StderrPipe()

	cmd.Start()
	reader := bufio.NewReader(io.MultiReader(stdout, stderr))
	for {
		line, err2 := reader.ReadString('\n')
		if err2 != nil || io.EOF == err2 {
			break
		}
		line = strings.TrimRight(line, "\n")
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
				filesModTime[path+file.Name()] = file.ModTime().Unix()
			}
		}
	}
}
