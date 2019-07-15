package dailywriter

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// io.WriteCloser just for compile test
var _ io.WriteCloser = (*DailyWriter)(nil)

const (
	compressSuffix = ".gz"
	// golang's birth, just for memory
	timeFormat = "2006-01-02 15:04:05.000"

	defaultReserve = 30

	defaultFilenameSufffix = ".log"
	//must three %s,for prefix+time+suffix
	defaultFilenameFormat = "%s_%s%s"

	defaultDirMode  = os.FileMode(0744)
	defaultFileMode = os.FileMode(0644)
	//defaultCompressMode = os.FileMode(0444)
	// maybe can set more coarse
	defaultOpenNewFileFlag = os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	defaultOpenFileFlag    = os.O_WRONLY | os.O_APPEND
)

// "2006-01-02-15-04"=16 "2006-01-02"=10 "2006-01-02-15-04-05"=19 "2006-01-02-15"=13
var defaultFilenameDateFormat = "2006-01-02-15-04-05"
// must the same length to format
var defaultFilenameDateLength = 19

var defaultDir = os.TempDir()
var defaultFilenamePrefix = os.Args[0]
var defaultTimeType = "day"
var defaultTimeInterval = 24 * time.Hour

var errPrefix = errors.New("mismatched prefix")
var errSuffix = errors.New("mismatched Suffix")
var errDate = errors.New("mismatched date")

func init() {
	formatLength := time.Now().Local().Format(defaultFilenameDateFormat)
	if len(formatLength) != defaultFilenameDateLength {
		panic("FilenameDateFormat must have the same length with FilenameDateLength")
	}
}

type DailyWriter struct {
	// TODO : better var's placement
	dir             string
	filenamePrefix  string // for prefix
	filenameSufffix string // for ext,i.e. ".log" ".exe"

	curFilename string
	file        *os.File

	lock       sync.Mutex
	startClean sync.Once
	cleanCh    chan struct{}

	compress bool
	reserve  int
}

func New(dir, filenamePrefix, filenameSufffix string, compress bool, reserve int, timeType string) (*DailyWriter, error) {
	if dir == "" {
		dir = defaultDir
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	if filenamePrefix == "" {
		filenamePrefix = defaultFilenamePrefix
	}
	if filenameSufffix == "" {
		filenameSufffix = defaultFilenameSufffix
	}
	if timeType == "" {
		timeType = "day"
	}
	switch timeType {
	case "day", "d":
		defaultFilenameDateLength = 10
		defaultFilenameDateFormat = "2006-01-02"
	case "hour", "h":
		defaultTimeInterval = time.Hour
		defaultFilenameDateLength = 13
		defaultFilenameDateFormat = "2006-01-02-15"
	case "minute", "m":
		defaultTimeInterval = time.Minute
		defaultFilenameDateLength = 16
		defaultFilenameDateFormat = "2006-01-02-15-04"
	case "second", "s":
		defaultTimeInterval = time.Second
		defaultFilenameDateLength = 19
		defaultFilenameDateFormat = "2006-01-02-15-04-05"
	default:
		panic("error with FilenameDateLength and timeType(unknown)")
	}

	return &DailyWriter{
		dir:             dir,
		filenamePrefix:  filenamePrefix,
		filenameSufffix: filenameSufffix,
		compress:        compress,
		reserve:         reserve,
	}, nil
}

func (this *DailyWriter) Write(p []byte) (n int, err error) {
	// lock may slow, better to less scope
	this.lock.Lock()
	defer this.lock.Unlock()

	// this name can be set before lock.Lock()
	filename := this.filename()

	if this.file == nil {
		if err = this.openExistingOrNew(filename); err != nil {
			fmt.Fprintf(os.Stderr, "write fail, msg(%s)\n", err)
			return 0, err
		}
	}
	if this.curFilename != filename {
		this.rotate(filename)
	}
	return this.file.Write(p)
}

func (this *DailyWriter) WriteWithTime(p []byte, now time.Time) (n int, err error) {
	// lock may slow, better to less scope
	this.lock.Lock()
	defer this.lock.Unlock()

	// this name can be set before lock.Lock()
	filename := this.filenameWithTime(now)

	if this.file == nil {
		if err = this.openExistingOrNew(filename); err != nil {
			fmt.Fprintf(os.Stderr, "write fail, msg(%s)\n", err)
			return 0, err
		}
	}
	if this.curFilename != filename {
		this.rotate(filename)
	}
	return this.file.Write(p)
}

// Rorate rotates the writer
func (this *DailyWriter) Rotate() error {
	this.lock.Lock()
	defer this.lock.Unlock()
	return this.rotate(this.filename())
}

func (this *DailyWriter) RotateWithNewFileName(filename string) error {
	this.lock.Lock()
	defer this.lock.Unlock()
	return this.rotate(filename)
}

// must attention to the hold of the this.lock
func (this *DailyWriter) rotate(filename string) (err error) {
	if err = this.close(); err != nil {
		return
	}
	if err = this.openNew(filename); err != nil {
		return
	}
	this.clean()
	return nil
}

// Close closes the file
func (this *DailyWriter) Close() error {
	this.lock.Lock()
	defer this.lock.Unlock()
	return this.close()
}

// must attention to the hold of the this.lock
func (this *DailyWriter) close() error {
	if this.file == nil {
		return nil
	}
	err := this.file.Close()
	this.file = nil
	return err
}

// openExistingOrNew opens file, if not exists, create it
func (this *DailyWriter) openExistingOrNew(filename string) error {
	// lazy for create
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return this.openNew(filename)
	} else if err != nil {
		return err
	}
	// now,the file exists
	file, err := os.OpenFile(filename, defaultOpenFileFlag, defaultFileMode)
	if err != nil {
		// try again
		return this.openNew(filename)
	}
	this.curFilename = filename
	this.file = file
	return nil
}

// openNew opens file
// must attention to the hold of the this.lock
func (this *DailyWriter) openNew(newname string) (err error) {
	if err = os.MkdirAll(this.dir, defaultDirMode); err != nil {
		return
	}
	file, err := os.OpenFile(newname, defaultOpenNewFileFlag, defaultFileMode)
	if err != nil {
		return
	}
	this.curFilename = newname
	this.file = file
	return nil
}

// filename returns the filename now.
func (this *DailyWriter) filename() string {
	return this.filenameWithTime(time.Now().Local())
}

func (this *DailyWriter) filenameWithTime(now time.Time) string {
	//year, month, day := now.Date()
	//date := fmt.Sprintf(defaultFilenameDateFormat, year, month, day)
	date := now.Format(defaultFilenameDateFormat)
	name := fmt.Sprintf(defaultFilenameFormat, this.filenamePrefix, date, this.filenameSufffix)
	return filepath.Join(this.dir, name)
}

func (this *DailyWriter) clean() {
	this.startClean.Do(func() {
		this.cleanCh = make(chan struct{}, 1)
		go this.cleanRun()
	})
	select {
	case this.cleanCh <- struct{}{}:
	default:
	}
}

func (this *DailyWriter) cleanRun() {
	for range this.cleanCh {
		//fmt.Println("****", "do clean")
		this.doClean()
	}
}

func (this *DailyWriter) doClean() error {
	if this.reserve == 0 && !this.compress {
		return nil
	}
	files, err := this.oldLogFiles()
	if err != nil {
		return err
	}

	var compress, remove []logInfo
	if this.reserve > 0 {
		diff := defaultTimeInterval * time.Duration(int64(this.reserve))
		cutoff := time.Now().Local().Add(-1 * diff)
		var remaining []logInfo
		// because files's sorted, so there's more convinient method to get remaining
		//fmt.Println("zzzzzz")
		for _, f := range files {
			//fmt.Println(f.timestamp.Format(defaultFilenameDateFormat),
			//	f.timestamp.Before(cutoff), cutoff.Format(defaultFilenameDateFormat))
			if f.timestamp.Before(cutoff) {
				remove = append(remove, f)
			} else {
				remaining = append(remaining, f)
			}
		}
		//fmt.Println("zzzzzz")
		files = remaining
	}
	if this.compress {
		for _, f := range files {
			if !strings.HasSuffix(f.Name(), compressSuffix) {
				compress = append(compress, f)
			}
		}
	}
	//fmt.Println("*****", len(remove))
	for _, f := range remove {
		errRemove := os.Remove(filepath.Join(this.dir, f.Name()))
		if err == nil && errRemove != nil {
			err = errRemove
		}
	}
	for _, f := range compress {
		fn := filepath.Join(this.dir, f.Name())
		//fmt.Println("*********", fn)
		errCompress := compressLogFile(fn, fn+compressSuffix)
		if err == nil && errCompress != nil {
			err = errCompress
		}
	}
	return err
}

func (this *DailyWriter) oldLogFiles() ([]logInfo, error) {
	var t = time.Time{}

	files, err := ioutil.ReadDir(this.dir)
	if err != nil {
		return nil, err
	}
	logFiles := []logInfo{}
	for _, fileinfo := range files {

		if fileinfo.IsDir() {
			continue
		}
		//fmt.Println("index--", fileinfo.Name(), filepath.Base(this.curFilename))
		if fileinfo.Name() == filepath.Base(this.curFilename) {
			continue
		}
		//fmt.Println("index++", fileinfo.Name(), filepath.Base(this.curFilename))
		t, err = timeForName(fileinfo.Name(), this.filenamePrefix, this.filenameSufffix)
		if err == nil {
			//fmt.Println("normal--", fileinfo.Name())
			logFiles = append(logFiles, logInfo{t, fileinfo})
			continue
		} else {
			// just pass

			//fmt.Fprintf(os.Stderr, "%s-%s-%s\n",fileinfo.Name(),this.filenamePrefix,this.filenameSufffix)
			//fmt.Fprintf(os.Stderr, "parse time err(%s)\n", err)
		}
		t, err = timeForName(fileinfo.Name(), this.filenamePrefix, this.filenameSufffix+compressSuffix)
		if err == nil {
			//fmt.Println("normal--", fileinfo.Name())
			logFiles = append(logFiles, logInfo{t, fileinfo})
			continue
		} else {
			// just pass

			//fmt.Fprintf(os.Stderr, "parse time err(%s)\n", err)
		}
	}
	sort.Sort(byFormatTime(logFiles))
	return logFiles, nil
}

func timeForName(filename, prefix, ext string) (time.Time, error) {
	//fmt.Println("parse---", filename, prefix, ext)
	if !strings.HasPrefix(filename, prefix) {
		return time.Time{}, errPrefix
	}
	if !strings.HasSuffix(filename, ext) {
		return time.Time{}, errSuffix
	}
	ts := filename[len(prefix)+1 : len(filename)-len(ext)]
	//fmt.Println("parse++", filename, prefix, ext, ts, len(ts), defaultFilenameDateLength)

	if len(ts) != defaultFilenameDateLength {
		return time.Time{}, errDate
	}
	//fmt.Println("parse**", filename, prefix, ext)
	return time.ParseInLocation(defaultFilenameDateFormat, ts, time.Local)

	// Here is consider the time.Location

	//if year, err := strconv.ParseInt(ts[0:4], 10, 64); err != nil {
	//	return time.Time{}, fmt.Errorf("mismatched year: %v", err)
	//} else if month, err := strconv.ParseInt(ts[4:6], 10, 64); err != nil {
	//	return time.Time{}, fmt.Errorf("mismatched month: %v", err)
	//} else if day, err := strconv.ParseInt(ts[6:8], 10, 64); err != nil {
	//	return time.Time{}, fmt.Errorf("mismatched day: %v", err)
	//} else {
	//	timeStr := fmt.Sprintf("%04d-%02d-%02d 00:00:00.000", year, month, day)
	//	if location, err := time.LoadLocation("Local"); err != nil {
	//		return time.Time{}, err
	//	} else if t, err := time.ParseInLocation(timeFormat, timeStr, location); err != nil {
	//		return time.Time{}, err
	//	} else {
	//		return t, nil
	//	}
	//}
}

func compressLogFile(oldfile, newfile string) error {
	f, err := os.Open(oldfile)
	if err != nil {
		return err
	}
	defer f.Close()

	fi, err := os.Stat(oldfile)
	if err != nil {
		return err
	}

	gzf, err := os.OpenFile(newfile, defaultOpenNewFileFlag, fi.Mode())
	if err != nil {
		return err
	}
	defer gzf.Close()

	gz := gzip.NewWriter(gzf)
	defer gz.Close()

	defer func() {
		if err != nil {
			os.Remove(newfile)
			err = fmt.Errorf("failed to compress log file: %v", err)
		}
	}()

	if _, err := io.Copy(gz, f); err != nil {
		return err
	}
	// double check to insure close with no error
	if err := gz.Close(); err != nil {
		return err
	}
	if err := gzf.Close(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Remove(oldfile); err != nil {
		return err
	}
	return nil
}

type logInfo struct {
	timestamp time.Time
	os.FileInfo
}

// byFormatTime sorts by newest time formatted in the name.
type byFormatTime []logInfo

func (b byFormatTime) Less(i, j int) bool {
	return b[i].timestamp.After(b[j].timestamp)
}

func (b byFormatTime) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func (b byFormatTime) Len() int {
	return len(b)
}
