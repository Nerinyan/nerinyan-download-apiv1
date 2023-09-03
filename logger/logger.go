package logger

import (
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
)

var writer = &lumberjack.Logger{
	Filename:   "./logs/echo.log", // 로그 파일 경로
	MaxSize:    0,                 // 메가바이트 단위의 최대 크기
	MaxBackups: 0,                 // 보관할 이전 로그 파일의 최대 수
	MaxAge:     1,                 // 일 단위로 보관할 로그 파일의 최대 수명
	Compress:   true,              // 로그 파일을 gzip으로 압축할지 여부
}

func GetFileWriter() io.Writer {
	return writer
}
