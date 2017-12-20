package main

import (
	"time"
	"io/ioutil"
	"fmt"
	"os"
	"os/exec"
	"bufio"
	"io"
)

var filesModTime = make(map[string]int64)

func main() {
	cmdArgs := make([]string, 1)
	cmdArgs[0] = "run"
	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "-t":
			cmdArgs[0] = "test"
		case "-b":
			cmdArgs = append(cmdArgs, " -bench", "'.*'")
		default:
			cmdArgs = append(cmdArgs, os.Args[i])
		}
	}

	if len(cmdArgs) == 1 && cmdArgs[0] == "run" {
		cmdArgs = append(cmdArgs, "*.go")
	}

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
			watchPath("./")
			time.Sleep(time.Second * 3)
		}
	}()

	for {
		select {
		case <-changed:
			runCommand("go", cmdArgs...)
		}
	}
	println("OK")
}

func runCommand(command string, args ...string) {
	cmd := exec.Command(command, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println(err)
		return
	}

	cmd.Start()
	reader := bufio.NewReader(stdout)
	for {
		line, err2 := reader.ReadString('\n')
		if err2 != nil || io.EOF == err2 {
			break
		}
		fmt.Println(line)
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
			fmt.Println(fileName, modTime)
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
