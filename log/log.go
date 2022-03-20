package log

import (
    "fmt"
    "os"
    "runtime"
    stdlog "log"
    "strings"
    "sync"
    "time"
)

type Level int
const (
    LevelDebug Level = iota
    LevelInfo
    LevelWarn
    LevelError
    LevelFatal //error + exit
)
const EndColor = "\033[0m"

type levelInfo struct {
    name  string
    color string
}
var levels = map[Level]levelInfo {
    LevelDebug: levelInfo { name: "debug", color: "\033[36m" },
    LevelInfo:  levelInfo { name: "info",  color: "\033[32m" },
    LevelWarn:  levelInfo { name: "warn",  color: "\033[93m" },
    LevelError: levelInfo { name: "error", color: "\033[91m" },
    LevelFatal: levelInfo { name: "fatal", color: "\033[91m" },
}
const eraseLine        = "\033[2K"
const windowTitleBegin = "\033]0;"
const windowTitleEnd   = "\007"

func ParseLevel(name string) (Level, error) {
    name = strings.ToLower(name)
    for level, info := range levels {
        if name == info.name {
            return level, nil
        }
    }
    return LevelFatal, fmt.Errorf("Invalid log level '%s'", name)
}

type Logger struct {
    buf         []byte
    extraFrames int
    mu          sync.Mutex
    minLevel    Level
    tag         string
}

// used to track progress updates (which use \r to replace the previous one)
// but if something got logged after the last update, a new line has to be printed
var progress struct {
    mu           sync.Mutex
    buf          []byte
    title        string
    lastProgress []byte
    hasProgress  bool
}

var DefaultLogger *Logger

func init() {
    DefaultLogger = &Logger {
        extraFrames: 1,
    }
    stdlog.SetFlags(stdlog.Ldate | stdlog.Lmicroseconds | stdlog.Lshortfile)
    stdlog.SetOutput(stdLogProxy {})
}

func doWrite(isProgress bool, title string, data []byte) (int, error) {
    progress.mu.Lock()
    defer progress.mu.Unlock()

    progress.buf = progress.buf[:0]
    if progress.hasProgress {
        progress.buf = append(progress.buf, eraseLine...)
        progress.buf = append(progress.buf, '\r')
        progress.buf = append(progress.buf, data...)
    } else {
        progress.buf = append(progress.buf, data...)
    }

    if isProgress {
        progress.lastProgress = append(progress.lastProgress[:0], data...)
        progress.hasProgress = true
        if len(title) > 0 && title != progress.title {
            progress.title = title
            progress.buf = append(progress.buf, windowTitleBegin...)
            progress.buf = append(progress.buf, title...)
            progress.buf = append(progress.buf, windowTitleEnd...)
        }
        os.Stderr.Write(progress.buf)
    } else {
        progress.buf = append(progress.buf, progress.lastProgress...)
        os.Stderr.Write(progress.buf)
    }
    return len(data), nil
}

type stdLogProxy struct {}

func (_ stdLogProxy) Write(p []byte) (int, error) {
    return doWrite(false, "", p)
}

func Progress(title string, line string) {
    doWrite(true, title, []byte(line))
}

func SetDefaultLevel(level Level) {
    DefaultLogger.minLevel = level
}

func New(tag string) *Logger {
    return &Logger {
        minLevel: DefaultLogger.minLevel,
        tag:      tag,
    }
}

func (l *Logger) SubLogger(tag string) *Logger {
    return New(fmt.Sprintf("%s.%s", l.tag, tag))
}

func (l *Logger) output(level Level, calldepth int, s string) {
    now := time.Now().UTC()
    var file string
    var line int

    if len(l.tag) == 0 {
        var ok bool
        _, file, line, ok = runtime.Caller(calldepth + l.extraFrames)
        if !ok {
            file = "???"
            line = 0
        }
    }
    l.mu.Lock()
    defer l.mu.Unlock()

    l.buf = l.buf[:0]

    info := levels[level]
    l.buf = append(l.buf, info.color...)
    formatTime(&l.buf, now)
    l.buf = append(l.buf, info.name...)
    l.buf = append(l.buf, ": "...)
    for i := len(info.name); i < 5; i++ {
        l.buf = append(l.buf, ' ')
    }

    formatHeader(&l.buf, l.tag, file, line)
    l.buf = append(l.buf, s...)
    if len(s) == 0 || s[len(s)-1] != '\n' {
        l.buf = append(l.buf, '\n')
    }
    l.buf = append(l.buf, EndColor...)
    doWrite(false, "", l.buf)
}

func (l *Logger) logf(level Level, format string, v ...interface{}) {
    if int(level) >= int(l.minLevel) {
        l.output(level, 3, fmt.Sprintf(format, v...))
    }
    if level == LevelFatal {
        os.Exit(1)
    }
}

func (l *Logger) log(level Level, v ...interface{}) {
    if int(level) >= int(l.minLevel) {
        l.output(level, 3, fmt.Sprint(v...))
    }
    if level == LevelFatal {
        os.Exit(1)
    }
}

func (l *Logger) Debug(v ...interface{}) {
    l.log(LevelDebug, v...)
}
func (l *Logger) Debugf(format string, v ...interface{}) {
    l.logf(LevelDebug, format, v...)
}

func (l *Logger) Info(v ...interface{}) {
    l.log(LevelInfo, v...)
}
func (l *Logger) Infof(format string, v ...interface{}) {
    l.logf(LevelInfo, format, v...)
}

func (l *Logger) Warn(v ...interface{}) {
    l.log(LevelWarn, v...)
}
func (l *Logger) Warnf(format string, v ...interface{}) {
    l.logf(LevelWarn, format, v...)
}

func (l *Logger) Error(v ...interface{}) {
    l.log(LevelError, v...)
}
func (l *Logger) Errorf(format string, v ...interface{}) {
    l.logf(LevelError, format, v...)
}

func (l *Logger) Fatal(v ...interface{}) {
    l.log(LevelFatal, v...)
}
func (l *Logger) Fatalf(format string, v ...interface{}) {
    l.logf(LevelFatal, format, v...)
}

func Debug(v ...interface{}) {
    DefaultLogger.Debug(v...)
}
func Debugf(format string, v ...interface{}) {
    DefaultLogger.Debugf(format, v...)
}

func Info(v ...interface{}) {
    DefaultLogger.Info(v...)
}
func Infof(format string, v ...interface{}) {
    DefaultLogger.Infof(format, v...)
}

func Warn(v ...interface{}) {
    DefaultLogger.Warn(v...)
}
func Warnf(format string, v ...interface{}) {
    DefaultLogger.Warnf(format, v...)
}

func Error(v ...interface{}) {
    DefaultLogger.Error(v...)
}
func Errorf(format string, v ...interface{}) {
    DefaultLogger.Errorf(format, v...)
}

func Fatal(v ...interface{}) {
    DefaultLogger.Fatal(v...)
}
func Fatalf(format string, v ...interface{}) {
    DefaultLogger.Fatalf(format, v...)
}

