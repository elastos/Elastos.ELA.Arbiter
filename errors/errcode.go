package errors

type ErrCode int

const (
	Error                   ErrCode = -1
	Success                 ErrCode = 0

	InvalidMethod           ErrCode = 42001
	InvalidParams           ErrCode = 42002
	InvalidToken            ErrCode = 42003
	InvalidTransaction      ErrCode = 43001
	UnknownTransaction      ErrCode = 44001
	UnknownBlock            ErrCode = 44003
	InternalError           ErrCode = 45002
)

var ErrMap = map[ErrCode]string{
	Error:                   "Unclassified error",
	Success:                 "Success",
	InvalidMethod:           "Invalid method",
	InvalidParams:           "Invalid Params",
	InvalidToken:            "Verify token error",
	InvalidTransaction:      "Invalid transaction",
	UnknownTransaction:      "Unknown Transaction",
	UnknownBlock:            "Unknown Block",
	InternalError:           "Internal error",
}

func (code ErrCode) Message() string {
	return ErrMap[code]
}
