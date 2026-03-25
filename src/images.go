package src

import (
	b64 "encoding/base64"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

func uploadLogo(input, filename string) (logoURL string, err error) {

	if strings.ContainsAny(filename, "/\\") || filename != filepath.Base(filename) || filename == "." || filename == ".." {
		err = errors.New("invalid filename")
		return
	}

	b64data := input[strings.IndexByte(input, ',')+1:]

	sDec, err := b64.StdEncoding.DecodeString(b64data)
	if err != nil {
		return
	}

	var file = System.Folder.ImagesUpload + filename

	err = writeByteToFile(file, sDec)
	if err != nil {
		return
	}

	logoURL = fmt.Sprintf("%s://%s/data_images/%s", System.ServerProtocol.XML, System.Domain, filename)

	return

}
