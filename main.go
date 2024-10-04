package main

import (
	"embed"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"runtime"
	"syscall"
)

//go:embed center.conf hub.conf node.conf
var configFiles embed.FS

const cacheDir = ".cache"

// TODO ftype: conf, sock
func getPath(fname string, ftype string) string {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		panic(err)
	}
	// NOTE: even on Win32, it's in cygwin env
	tempFilePath := path.Join(cacheDir, fmt.Sprintf("%s.%s", fname, ftype))
	return tempFilePath
}

func usage() {
	fmt.Println("Supported Commands: center, hub, node and reset.")
}

func startSession(name string) {
    fmt.Printf("[CTMUX] Starting %s...\n", name)
	tempConfigPath := getPath(name, "conf")
	// Created by tmux
	tempSockPath := getPath(name, "sock")

    // Write embedded config to a temporary file
	if _, err := os.Stat(tempConfigPath); os.IsNotExist(err) {
		configFileContent, err := configFiles.ReadFile(fmt.Sprintf("%s.conf", name))
		if err != nil { panic(err) }

		if err := ioutil.WriteFile(tempConfigPath, configFileContent, 0644); err != nil {
			panic(err)
		}
		// defer os.Remove(tempConfigPath)
	}

	tmuxBinary, err := exec.LookPath("tmux")
	if err != nil { panic(err) }

	if runtime.GOOS == "windows" {
		fmt.Println("On Windows")

		cmd := exec.Command("tmux", "-S", tempSockPath, "-f", tempConfigPath, "at", )
		// Redirect stdin, stdout, and stderr to the current process's streams
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error running tmux session on Windows: %v\n", err)
			panic(err)
		}
	} else {
		// Probably use the exec.Command as well.
		args := []string{"tmux", "-S", tempSockPath, "-f", tempConfigPath, "at"}
		if err := syscall.Exec(tmuxBinary, args, os.Environ()); err != nil {
			panic(err)
		}
	}
}

func resetCache() {
    if err := os.RemoveAll(cacheDir); err != nil {
        panic(err)
    }
	fmt.Printf("Removed %s\n", cacheDir);
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Error: No arguments provided.")
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
	case "reset":
		resetCache()
	default:
		fmt.Printf("Invalid argument %s.\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}
