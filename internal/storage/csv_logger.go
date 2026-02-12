package storage

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

type CSVLogger struct {
	BaseDir string
	mu      sync.Mutex
}

func (csvL *CSVLogger) endureDir() error {
	if csvL.BaseDir == "" {
		csvL.BaseDir = "logs"
	}
	return os.MkdirAll(csvL.BaseDir, 0755)
}

func (csvL *CSVLogger) LogStateChange(deviceName, ip, status, reason string, checkedAt time.Time, rtt int64) error {
	csvL.mu.Lock()
	defer csvL.mu.Unlock()

	if err := csvL.endureDir(); err != nil {
		return err
	}

	filename := time.Now().Format("2006-01-02") + "-events.csv"
	//filename := deviceName

	if filename == "" {
		filename = ip
	}
	//filename = sanitizeFileName(filename) + ".csv"
	path := filepath.Join(csvL.BaseDir, filename)

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)

	if err != nil {
		return err
	}
	defer f.Close()

	// If file doesn't exist, write header
	newfile := false
	if stat, err := f.Stat(); err == nil && stat.Size() == 0 {
		newfile = true
	}

	w := csv.NewWriter(f)
	defer w.Flush()

	if newfile {
		w.Write([]string{"Timestamp", "IP", "Status", "RTT_ms", "Reason"})
	}

	return w.Write([]string{
		checkedAt.Format("2006-01-02 15:04:05"),
		ip,
		status,
		formatInt(rtt),
		reason,
	})

}

/*
	func sanitizeFileName(filename string) string {
		// Replace any characters that are not allowed in file names with underscores
		// This is a simple implementation and may need to be expanded based on specific requirements
		invalidChars := []rune{'/', '\\', ':', '*', '?', '"', '<', '>', '|'}
		filenamerunes := []rune(filename)
		for i, r := range filenamerunes {
			for _, invalid := range invalidChars {
				if r == invalid {
					filenamerunes[i] = '_'
					break
				}
			}
		}
		return string(filenamerunes)
	}
*/
func formatInt(v int64) string {
	if v == 0 {
		return ""
	}
	return strconv.FormatInt(v, 10)
}
