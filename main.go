package main

import (
	"embed"
	"fmt"
    "io/ioutil"
	"os"
	"os/exec"
	"syscall"
	"path/filepath"
)

//go:embed center.conf hub.conf node.conf
var configFiles embed.FS

// TODO ftype: conf, sock
func getPath(fname string, ftype string) string {
	const cacheDir = ".cache"
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		panic(err)
	}
	tempFilePath := filepath.Join(cacheDir, fmt.Sprintf("%s.%s", fname, ftype))
	return tempFilePath
}

func usage() {
	fmt.Println("Please specify: center, hub or node.")
}

func startSession(name string) {
    fmt.Printf("Starting %s...\n", name)
	tempConfigPath := getPath(name, "conf")
	tempSockPath := getPath(name, "sock")

    // Write embedded config to a temporary file
	configFileContent, err := configFiles.ReadFile(fmt.Sprintf("%s.conf", name))
	if err != nil { panic(err) }

    if err := ioutil.WriteFile(tempConfigPath, configFileContent, 0644); err != nil {
        panic(err)
    }
    defer os.Remove(tempConfigPath)

	tmuxBinary, err := exec.LookPath("tmux")
	if err != nil { panic(err) }

	if _, err := os.Create(tempSockPath); err != nil {
		panic(err)
	}
    defer os.Remove(tempSockPath)
	args := []string{"tmux", "-S", tempSockPath, "-f", tempConfigPath, "at"}
	if err := syscall.Exec(tmuxBinary, args, os.Environ()); err != nil {
        panic(err)
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("No arguments provided.")
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "center":
		startSession("center")
	case "hub":
		startSession("hub")
	case "node":
		startSession("node")
	default:
		fmt.Printf("Invalid argument %s.\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}
