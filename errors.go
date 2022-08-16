package tiny

import (
	"fmt"
	"net/http"
)

type (
	Error struct {
		code int
		err  string
	}
)

func NewError(code int, format string, args ...interface{}) Error {
	return Error{
		code: code,
		err:  fmt.Sprintf(format, args...),
	}
}

func (err Error) Code() int {
	return err.code
}

func (err Error) Error() string {
	return err.err
}

func ErrorFromErr(err error) Error {
	if err, ok := err.(Error); ok {
		return err
	}
	if e, ok := err.(interface{ Code() int32 }); ok {
		return NewError(int(e.Code()), err.Error())
	} else if e, ok := err.(interface{ Code() uint32 }); ok {
		return NewError(int(e.Code()), err.Error())
	} else if e, ok := err.(interface{ Code() int64 }); ok {
		return NewError(int(e.Code()), err.Error())
	} else if e, ok := err.(interface{ Code() uint64 }); ok {
		return NewError(int(e.Code()), err.Error())
	} else if e, ok := err.(interface{ Code() uint }); ok {
		return NewError(int(e.Code()), err.Error())
	}
	return NewError(http.StatusInternalServerError, err.Error())
}
