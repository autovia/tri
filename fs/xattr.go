package fs

import (
	"fmt"

	"golang.org/x/sys/unix"
)

func Setxattr(file, key, value string) error {
	return unix.Setxattr(file, fmt.Sprintf("user.%s", key), []byte(value), 0)
}

func Getxattr(file, key string) (string, error) {
	size, err := unix.Getxattr(file, fmt.Sprintf("user.%s", key), nil)
	if err != nil {
		return "", err
	}

	buf := make([]byte, size)
	_, err = unix.Getxattr(file, fmt.Sprintf("user.%s", key), buf)
	if err != nil {
		return "", err
	}

	return string(buf), nil
}
