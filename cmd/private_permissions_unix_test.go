//go:build !windows

package cmd

import "os"

func makeFileBroadlyReadable(path string) error { return os.Chmod(path, 0644) }
