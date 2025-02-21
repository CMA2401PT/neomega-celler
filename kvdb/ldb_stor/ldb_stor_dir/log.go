package ldb_stor_dir

import (
	"os"
	"path/filepath"
	"time"
)

func (fs *DirStorage) Log(str string) {
	if fs.readOnly {
		return
	}
	t := time.Now()
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if fs.open < 0 {
		return
	}
	fs.doLog(t, str)
}

func (fs *DirStorage) log(str string) {
	if fs.readOnly {
		return
	}
	fs.doLog(time.Now(), str)
}

func (fs *DirStorage) doLog(t time.Time, str string) {
	if fs.readOnly {
		return
	}
	if fs.logSize > logSizeThreshold {
		// Rotate log file.
		fs.logw.Close()
		fs.logw = nil
		fs.logSize = 0
		rename(filepath.Join(fs.path, "LOG"), filepath.Join(fs.path, "LOG.old"))
	}
	if fs.logw == nil {
		var err error
		fs.logw, err = os.OpenFile(filepath.Join(fs.path, "LOG"), os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return
		}
		// Force printDay on new log file.
		fs.day = 0
	}
	fs.printDay(t)
	hour, min, sec := t.Clock()
	msec := t.Nanosecond() / 1e3
	// time
	fs.buf = itoa(fs.buf[:0], hour, 2)
	fs.buf = append(fs.buf, ':')
	fs.buf = itoa(fs.buf, min, 2)
	fs.buf = append(fs.buf, ':')
	fs.buf = itoa(fs.buf, sec, 2)
	fs.buf = append(fs.buf, '.')
	fs.buf = itoa(fs.buf, msec, 6)
	fs.buf = append(fs.buf, ' ')
	// write
	fs.buf = append(fs.buf, []byte(str)...)
	fs.buf = append(fs.buf, '\n')
	n, _ := fs.logw.Write(fs.buf)
	fs.logSize += int64(n)
}

func (fs *DirStorage) printDay(t time.Time) {
	if fs.readOnly {
		return
	}
	if fs.day == t.Day() {
		return
	}
	fs.day = t.Day()
	fs.logw.Write([]byte("=============== " + t.Format("Jan 2, 2006 (MST)") + " ===============\n"))
}
