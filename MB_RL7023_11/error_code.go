package MB_RL7023_11

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// https://rabbit-note.com/wp-content/uploads/2016/12/50f67559796399098e50cba8fdbe6d0a.pdf
type ErrorCode uint8

const (
	// ER01 reserved
	ErrorCodeReserved1 ErrorCode = 1
	// ER02 reserved
	ErrorCodeReserved2 ErrorCode = 2
	// ER03 reserved
	ErrorCodeReserved3 ErrorCode = 3
	// ER04 指定されたコマンドがサポートされていない
	ErrorCodeCommandNotSupported ErrorCode = 4
	// ER05 指定されたコマンドの引数の数が正しくない
	ErrorCodeInvalidParameterLengh ErrorCode = 5
	// ER06 指定されたコマンドの引数形式や値域が正しくない
	ErrorCodeInvalidParameter ErrorCode = 6
	// ER07 reserved
	ErrorCodeReserved7 ErrorCode = 7
	// ER08 reserved
	ErrorCodeReserved8 ErrorCode = 8
	// ER09 UART 入力エラーが発生した
	ErrorCodeUARTInputError ErrorCode = 9
	// ER10 指定されたコマンドは受付けたが、実行結果が失敗した
	ErrorCodeCommandFailed ErrorCode = 10
)

var (
	// ER01 reserved
	ErrReserved1 = errors.New("reserved1")
	// ER02 reserved
	ErrReserved2 = errors.New("reserved2")
	// ER03 reserved
	ErrReserved3 = errors.New("reserved3")
	// ER04 指定されたコマンドがサポートされていない
	ErrCommandNotSupported = errors.New("command not supported")
	// ER05 指定されたコマンドの引数の数が正しくない
	ErrInvalidParameterLengh = errors.New("invalid parameter lengh")
	// ER06 指定されたコマンドの引数形式や値域が正しくない
	ErrInvalidParameter = errors.New("invalid parameter")
	// ER07 reserved
	ErrReserved7 = errors.New("reserved7")
	// ER08 reserved
	ErrReserved8 = errors.New("reserved8")
	// ER09 UART 入力エラーが発生した
	ErrUARTInputError = errors.New("uart input error")
	// ER10 指定されたコマンドは受付けたが、実行結果が失敗した
	ErrCommandFailed = errors.New("command failed")
	// unknown error code
	ErrUnknownErrorCode = errors.New("unknown error code")
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
		return 0, ErrUnknownErrorCode
	}

	val, err := strconv.ParseUint(s[len(errorCodePrefix):], 10, 8)
	if err != nil {
		return 0, err
	}

	return ErrorCode(val), nil
}

func (e ErrorCode) String() string {
	return fmt.Sprintf("ER%02d", uint8(e))
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
