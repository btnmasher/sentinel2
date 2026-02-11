package logging

import (
	_ "unsafe"

	pblogger "github.com/pocketbase/pocketbase/tools/logger"
)

// Prototype: replace PocketBase's internal dev printer without forking.
// This uses go:linkname to reach into the PocketBase core package.
// It is intentionally isolated and should be treated as a fragile hack.

//go:linkname pocketbasePrintLog github.com/pocketbase/pocketbase/core.printLog
var pocketbasePrintLog func(*pblogger.Log)

var pocketbasePrinterInstalled bool

func installPocketBasePrettyPrinter() {
	installPocketBasePrinterHook()
}

func installPocketBaseJSONWriter() {
	installPocketBasePrinterHook()
}

func installPocketBasePrinterHook() {
	if pocketbasePrinterInstalled || pocketbasePrintLog == nil {
		pocketbasePrinterInstalled = true
		return
	}

	original := pocketbasePrintLog
	pocketbasePrintLog = func(log *pblogger.Log) {
		if prettyEnabled {
			prettyPrintFromPB(log)
		}
		if jsonEnabled {
			writeJSONFromPB(log)
		}
		if original != nil {
			original(log)
		}
	}
	pocketbasePrinterInstalled = true
}
