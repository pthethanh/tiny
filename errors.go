package tiny

import (
	"fmt"
	"net/http"
)

type (
	TinyError struct {
		code uint32
		err  string
	}
)

func Error(code uint32, format string, args ...interface{}) TinyError {
	return TinyError{
		code: code,
		err:  fmt.Sprintf(format, args...),
	}
}

func (err TinyError) Code() uint32 {
	return err.code
}

func (err TinyError) Error() string {
	return err.err
}

func ErrorFromErr(err error) TinyError {
	if err, ok := err.(TinyError); ok {
		return err
	}
	code := uint32(http.StatusInternalServerError)
	if err, ok := err.(interface{ Code() uint32 }); ok {
		code = err.Code()
	}
	return Error(code, err.Error())
}
