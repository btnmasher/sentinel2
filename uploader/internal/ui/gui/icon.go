//go:build !headless

package gui

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed assets/s2-uploader-icon.svg
var s2UploaderIconSVG []byte

func uploaderIconResource() fyne.Resource {
	return fyne.NewStaticResource("s2-uploader-icon.svg", s2UploaderIconSVG)
}
