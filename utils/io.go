package utils

import (
	"bytes"
	"github.com/Nerinyan/nerinyan-download-apiv1/logger"
	"io"
)

// CaptureRW writer와 관련된 에러는 무시한다
func CaptureRW(reader io.Reader, writer io.Writer) (cap bytes.Buffer, err error) {
	wErr := false
	for {
		chunk := make([]byte, 16384) // 16384 단위로 응답이 들어오는듯함
		n, e := reader.Read(chunk)
		if n > 0 {
			cap.Write(chunk[:n])
			if !wErr { // 클리이언트의 연결 종료등 오류발생하는경우 전송하지 않는다
				_, err := writer.Write(chunk[:n]) // 청크 단위로 응답
				if err != nil {
					logger.Error(err)
					wErr = true
				}
			}
		}
		if e != nil {
			if e == io.EOF {
				break
			}
			logger.Error(e)
			err = e
			return
		}
	}
	return
}
