package jobs

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"time"
)

// Progress is a compact live record; full reports remain in message.txt and
// raw execution output remains in target.log. Each record repeats the latest
// message preview so status needs only a bounded tail, even after tool activity.
type Progress struct {
	At          *time.Time `json:"at,omitempty"`
	Event       string     `json:"event,omitempty"`
	LastMessage string     `json:"last_message,omitempty"`
}

func AppendProgress(path string, progress Progress) error {
	data, err := json.Marshal(progress)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	_, err = file.Write(append(data, '\n'))
	closeErr := file.Close()
	if err != nil {
		return err
	}
	return closeErr
}

const progressTailBytes = 8192

func ReadProgress(path string) (Progress, error) {
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return Progress{}, nil
	}
	if err != nil {
		return Progress{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return Progress{}, err
	}
	offset := max(int64(0), info.Size()-progressTailBytes)
	data := make([]byte, info.Size()-offset)
	n, err := file.ReadAt(data, offset)
	if err != nil && err != io.EOF {
		return Progress{}, err
	}
	data = data[:n]
	// Discard an incomplete first record and an in-flight final write.
	if offset > 0 {
		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			data = data[i+1:]
		} else {
			return Progress{}, nil
		}
	}
	lines := bytes.Split(data, []byte{'\n'})
	for i := len(lines) - 2; i >= 0; i-- {
		var progress Progress
		if json.Unmarshal(lines[i], &progress) == nil && progress.At != nil {
			return progress, nil
		}
	}
	return Progress{}, nil
}
