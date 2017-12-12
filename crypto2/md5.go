package crypto2

import (
	"os"
	"bufio"
	"crypto/md5"
	"io"
	"fmt"
	"encoding/hex"
)

func Md5File(file string) (string , error){
	f, err := os.Open(file)
	if err != nil {
		return "",err
	}
	defer f.Close()
	r := bufio.NewReader(f)
	h := md5.New()
	_, err = io.Copy(h, r)
	if err != nil {
		return "",err
	}
	return fmt.Sprintf("%x", h.Sum(nil)),nil
}

func Md5String(str string)  string{
	md5Ctx := md5.New()
	md5Ctx.Write([]byte(str))
	return hex.EncodeToString(md5Ctx.Sum(nil))
}