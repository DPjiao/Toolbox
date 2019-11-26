package util

import (
	"bytes"
	"encoding/gob"
)

/**
深度拷贝dst是拷贝到的位置,src是需要拷贝的对象
 */
func DeepCopy(dst, src interface{}) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(src); err != nil {
		return err
	}
	return gob.NewDecoder(bytes.NewBuffer(buf.Bytes())).Decode(dst)
}
