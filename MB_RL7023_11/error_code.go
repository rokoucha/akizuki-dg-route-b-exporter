package MB_RL7023_11

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var (
	ErrReserved1             = errors.New("reserved1")
	ErrReserved2             = errors.New("reserved2")
	ErrReserved3             = errors.New("reserved3")
	ErrCommandNotSupported   = errors.New("command not supported")
	ErrInvalidParameterLengh = errors.New("invalid parameter lengh")
	ErrInvalidParameter      = errors.New("invalid parameter")
	ErrReserved7             = errors.New("reserved7")
	ErrReserved8             = errors.New("reserved8")
	ErrUARTInputError        = errors.New("uart input error")
	ErrCommandFailed         = errors.New("command failed")
	ErrUnknownErrorCode      = errors.New("unknown error code")
)

type ErrorCode uint8

const (
	ErrorCodeReserved1             ErrorCode = 0x01
	ErrorCodeReserved2             ErrorCode = 0x02
	ErrorCodeReserved3             ErrorCode = 0x03
	ErrorCodeCommandNotSupported   ErrorCode = 0x04
	ErrorCodeInvalidParameterLengh ErrorCode = 0x05
	ErrorCodeInvalidParameter      ErrorCode = 0x06
	ErrorCodeReserved7             ErrorCode = 0x07
	ErrorCodeReserved8             ErrorCode = 0x08
	ErrorCodeUARTInputError        ErrorCode = 0x09
	ErrorCodeCommandFailed         ErrorCode = 0x10
)

func parseError(res []string, err error) error {
	if err == nil {
		return nil
	}
	if !errors.Is(err, ErrExecFailed) {
		return err
	}
	errCode, err := errorCodeFromString(res[0])
	if err != nil {
		return err
	}
	return errCode.Error()
}

const errorCodePrefix = "FAIL ER"

func errorCodeFromString(s string) (ErrorCode, error) {
	if !strings.HasPrefix(s, errorCodePrefix) {
		return 0, errors.New("invalid error code: " + s)
	}

	val, err := strconv.ParseUint(s[len(errorCodePrefix):], 16, 8)
	if err != nil {
		return 0, err
	}

	return ErrorCode(val), nil
}

func (e ErrorCode) String() string {
	return fmt.Sprintf("ER%02X", uint8(e))
}

func (e ErrorCode) Error() error {
	switch e {
	case ErrorCodeReserved1:
		return ErrReserved1
	case ErrorCodeReserved2:
		return ErrReserved2
	case ErrorCodeReserved3:
		return ErrReserved3
	case ErrorCodeCommandNotSupported:
		return ErrCommandNotSupported
	case ErrorCodeInvalidParameterLengh:
		return ErrInvalidParameterLengh
	case ErrorCodeInvalidParameter:
		return ErrInvalidParameter
	case ErrorCodeReserved7:
		return ErrReserved7
	case ErrorCodeReserved8:
		return ErrReserved8
	case ErrorCodeUARTInputError:
		return ErrUARTInputError
	case ErrorCodeCommandFailed:
		return ErrCommandFailed
	default:
		return ErrUnknownErrorCode
	}
}
