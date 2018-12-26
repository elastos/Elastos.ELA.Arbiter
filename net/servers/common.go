package servers

import (
	"github.com/elastos/Elastos.ELA.Arbiter/errors"
)

func ResponsePack(errCode errors.ErrCode, result interface{}) map[string]interface{} {
	if errCode != 0 && (result == "" || result == nil) {
		result = errors.ErrMap[errCode]
	}
	return map[string]interface{}{"Result": result, "Error": errCode}
}
