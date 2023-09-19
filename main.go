package main

import (
	"regexp"
	"sync"
	"os"
	"fmt"
	"runtime"
	"bufio"
	"path/filepath"
	"strconv"
)

func usage() {
	fmt.Fprintln(os.Stderr, "usage: spion [process PID] [target directory]")
	os.Exit(1)
}

func dumpWorker(offset, count int64, targetFile, memFile string) {
	targetFileHandle, err := os.Create(targetFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create file %s\n", targetFile)
	}
	defer targetFileHandle.Close()

	memFileHandle, err := os.Open(memFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open file %s\n", memFile)
	}
	defer memFileHandle.Close()

	memFileHandle.Seek(int64(offset), 0)
	byteChunk := make([]byte, count)

	_, err = memFileHandle.Read(byteChunk)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to dump %s\n", targetFile)
	}

	targetFileHandle.Write(byteChunk)
}

func DumpMemory(pid, targetDir string) (error){
	// Check if output directory exists, and create if necessary.
	err := os.MkdirAll(targetDir, os.ModePerm)
	if err != nil {
		return err
	}

	// Set up waitgroup
	var wg sync.WaitGroup

	// Open memory map
	memMapFile, err := os.Open(filepath.Join("/proc", pid, "/maps"))
	if err != nil {
		return err
	}
	defer memMapFile.Close()

	// Scan memory map with regex
	// This will SKIP unreadable sections.
	lineScanner := bufio.NewScanner(memMapFile)

	for lineScanner.Scan() {
		re := regexp.MustCompile(`([0-9A-Fa-f]+)-([0-9A-Fa-f]+) ([r])`)
		match := re.FindStringSubmatch(lineScanner.Text())

		if len(match) > 0 {
			var offset int64
			offset, err = strconv.ParseInt(match[1], 16, 64)
			if err != nil {
				return err
			}

			var end int64
			end, err = strconv.ParseInt(match[2], 16, 64)

			if err != nil {
				return err
			}

			count := end - offset

			filename := filepath.Join(targetDir, (match[1] + "-" + match[2]))

			wg.Add(1)
			go func() {
				defer wg.Done()
				dumpWorker(offset, count, filename, filepath.Join("/proc", pid , "/mem"))
			}()

		}
	}
	wg.Wait()
	return nil
}

func main() {
	if runtime.GOOS == "windows" {
		fmt.Fprintf(os.Stderr, "This tool only functions on *nix systems.")
		os.Exit(1)
	}
	if len(os.Args) < 3 {
		usage()
	}
	err := DumpMemory(os.Args[1], os.Args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	} else {
		os.Exit(0)
	}
}
