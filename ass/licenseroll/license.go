package licenseroll

import (
	"bytes"
	"io"
	"os"
)

func PrintLicense() {
	io.Copy(os.Stdout, bytes.NewBufferString(License))
}
