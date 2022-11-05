package output

import (
	"os"
	"sync"
)

type FileOutput struct {
	Path string
	f    *os.File
	lock sync.Mutex
}

func NewFileOutput(path string) (*FileOutput, error) {

	err := CheckFile(path)
	if err != nil {
		return nil, err
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return nil, err
	}

	return &FileOutput{
		Path: path,
		f:    f,
	}, nil
}

func (fo *FileOutput) WriteEntry(entry []byte) error {
	fo.lock.Lock()
	defer fo.lock.Unlock()
	_, err := fo.f.Write(entry)
	if err != nil {
		return err
	}
	return nil
}

func (fo *FileOutput) Close() error {
	return fo.f.Close()
}
