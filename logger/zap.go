package logger

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"log"
	"os"
	"time"
)

var customTimeEncoder = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
}
var rotatedLogFile = &lumberjack.Logger{
	Filename:   "./logs/application.log", // 로그 파일 경로
	MaxSize:    0,                        // 메가바이트 단위의 최대 크기
	MaxBackups: 0,                        // 보관할 이전 로그 파일의 최대 수
	MaxAge:     0,                        // 일 단위로 보관할 로그 파일의 최대 수명
	Compress:   true,                     // 로그 파일을 gzip으로 압축할지 여부
}
var _ZAP *zap.Logger

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	log.SetOutput(io.MultiWriter(os.Stdout, rotatedLogFile))

	// Encoder 설정: yyyy-mm-ddThh:mm:ss | file:line | LEVEL | message
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     customTimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		zapcore.NewMultiWriteSyncer(zapcore.AddSync(io.MultiWriter(os.Stdout, rotatedLogFile))),
		zapcore.DebugLevel,
	)

	// 로거 생성 및 caller 정보 추가 (파일과 라인 정보를 위해)
	_ZAP = zap.New(core, zap.AddStacktrace(zap.ErrorLevel), zap.AddCaller(), zap.AddCallerSkip(1))

}
func Error(a ...any) {
	_ZAP.Error(fmt.Sprint(a...))
}
func Info(a ...any) {
	_ZAP.Info(fmt.Sprint(a...))
}
func Debug(a ...any) {
	_ZAP.Debug(fmt.Sprint(a...))
}
func Warn(a ...any) {
	_ZAP.Warn(fmt.Sprint(a...))
}
func Fatal(a ...any) {
	_ZAP.Fatal(fmt.Sprint(a...))
}

// ===========================================
func Errorf(format string, a ...any) {
	_ZAP.Error(fmt.Sprintf(format, a...))
}
func Infof(format string, a ...any) {
	_ZAP.Info(fmt.Sprintf(format, a...))
}
func Debugf(format string, a ...any) {
	_ZAP.Debug(fmt.Sprintf(format, a...))
}
func Warnf(format string, a ...any) {
	_ZAP.Warn(fmt.Sprintf(format, a...))
}
func Fatalf(format string, a ...any) {
	_ZAP.Fatal(fmt.Sprintf(format, a...))
}
