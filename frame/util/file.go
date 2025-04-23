package util

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
)

var (
	// CurrentExecPath 当前程序路径
	CurrentExecPath, _ = GetCurrExecPath()
	// CurrentExecDir 当前程序目录
	CurrentExecDir, _ = GetCurrExecDir()
)

// GetCurrExecPath 获取当前Exec的绝对路径
func GetCurrExecPath() (string, error) {
	file, err := exec.LookPath(os.Args[0])
	if err == nil {
		return filepath.Abs(file)
	}
	return "", err
}

// GetCurrExecDir 获取当前Exec所在的绝对目录
func GetCurrExecDir() (string, error) {
	file, err := GetCurrExecPath()
	if err == nil {
		return filepath.Dir(file), nil
	}
	return "", err
}

// GetCurrFileName 获取当前Exec的名字
func GetCurrFileName() (string, error) {
	path, err := GetCurrExecPath()
	if err == nil {
		_, name := filepath.Split(path)

		for i := len(name) - 1; i >= 0; i-- {
			if name[i] == '.' {
				return name[:i], nil
			}
		}

		return name, nil
	}
	return "", err
}

// IsDirExists 判断指定路劲是否存在
func IsDirExists(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return os.IsExist(err)
	}
	return fi.IsDir()
}

// WriteFileAtomic Write file to temp and atomically move when everything else succeeds.
func WriteFileAtomic(filename string, data []byte, perm os.FileMode) error {
	dir, name := path.Split(filename)
	f, err := ioutil.TempFile(dir, name)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if err == nil {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if permErr := os.Chmod(f.Name(), perm); err == nil {
		err = permErr
	}
	if err == nil {
		err = os.Rename(f.Name(), filename)
	}
	// Any err should result in full cleanup.
	if err != nil {
		os.Remove(f.Name())
	}
	return err
}
