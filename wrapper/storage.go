package wrapper

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"sync"
	"time"
)

var bufPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

type FrameLogger interface {
	Log(timestamp time.Time, frame []byte) error
	Close() error
}

type EchoReplayFrameLogger struct {
	zipFile    *os.File
	zipWriter  *zip.Writer
	fileWriter io.Writer
}

func NewEchoReplayFrameLogger(filename string) (*EchoReplayFrameLogger, error) {
	var err error

	// Create a new zip file
	zipFile, err := os.Create(filename)
	if err != nil {
		return nil, err
	}

	// Create a new zip writer
	zipWriter := zip.NewWriter(zipFile)

	// Create a new file inside the zip
	fileWriter, err := zipWriter.Create(filename)
	if err != nil {
		return nil, err
	}

	return &EchoReplayFrameLogger{
		zipWriter:  zipWriter,
		fileWriter: fileWriter,
	}, nil
}

func (r *EchoReplayFrameLogger) Log(timestamp time.Time, frame []byte) error {
	b := bufPool.Get().(*bytes.Buffer)
	b.Reset()
	b.WriteString(timestamp.Format("2006/01/02 15:04:05.000"))
	b.WriteByte('\t')
	b.Write(frame)
	b.WriteByte('\n')

	if _, err := r.fileWriter.Write(b.Bytes()); err != nil {
		return err
	}
	bufPool.Put(b)
	return nil
}

func (r *EchoReplayFrameLogger) Close() error {
	if err := r.zipWriter.Flush(); err != nil {
		return err
	}
	if err := r.zipWriter.Close(); err != nil {
		return err
	}
	if err := r.zipFile.Close(); err != nil {
		return err
	}
	return nil
}
