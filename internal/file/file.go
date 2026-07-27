package file

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
)

func ParseFilePath(fp string) (string, error) {
	if fp == "" {
		return "", fmt.Errorf("empty path")
	}

	// ~ to home
	if fp == "~" {
		return xdg.Home, nil
	}

	// ~/path to home/path
	if strings.HasPrefix(fp, "~/") {
		return filepath.Join(xdg.Home, fp[2:]), nil
	}

	// ~username unsupported, SSH config shouldn't need this probably
	if strings.HasPrefix(fp, "~") && fp[1] != '/' {
		return "", fmt.Errorf("unsupported: ~username expansion")
	}

	// clean it up
	return filepath.Clean(fp), nil
}

func GetFileInfo(fp string) (os.FileInfo, error) {
	parsedPath, err := ParseFilePath(fp)
	if err != nil {
		return nil, err
	}
	return os.Stat(parsedPath)
}

func Exists(fp string) (string, error) {
	file, err := ParseFilePath(fp)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(file)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("path is a directory")
	}
	return file, nil
}
