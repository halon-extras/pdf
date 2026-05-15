package main

// #cgo CFLAGS: -I/opt/halon/include
// #cgo LDFLAGS: -Wl,--unresolved-symbols=ignore-all
// #include <HalonMTA.h>
// #include <stdlib.h>
// #include <dlfcn.h>
import "C"
import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime/cgo"
	"strconv"
	"strings"
	"sync"
	"unsafe"

	pdfcpuApi "github.com/pdfcpu/pdfcpu/pkg/api"
	pdfcpuModel "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

type PDFConstructorOptions struct {
	Format              string    `json:"format"`
	Protocols           *[]string `json:"protocols"`
	Media               string    `json:"media"`
	PresentationalHints *bool     `json:"presentationalhints"`
	Stylesheets         *[]string `json:"stylesheets"`
}

type PDFAddAttachmentOptions struct {
	Desc string `json:"desc"`
}

type PDFtoStringOptions struct {
	Password string `json:"password"`
}

type Config struct {
	Stylesheets map[string]string `json:"stylesheets"`
}

type PDF struct {
	ctx *pdfcpuModel.Context
}

var config *Config
var configLock sync.Mutex

func PDFFromHTML(data string, protocols []string, stylesheets []string, media string, presentationalHints bool) (*bytes.Buffer, error) {
	args := []string{}
	if len(protocols) > 0 {
		for _, protocol := range protocols {
			if protocol != "file" && protocol != "http" && protocol != "https" && protocol != "data" && protocol != "ftp" {
				return nil, fmt.Errorf("unsupported protocol: %s", protocol)
			}
		}
		args = append(args, "--allowed-protocols="+strings.Join(protocols, ","))
	}

	for _, stylesheet := range stylesheets {
		var path string
		configLock.Lock()
		if config != nil {
			p, ok := config.Stylesheets[stylesheet]
			if ok {
				path = p
			}

		}
		configLock.Unlock()
		if path == "" {
			return nil, fmt.Errorf("stylesheet not found: %s", stylesheet)
		}
		if _, err := os.Stat(path); err != nil {
			return nil, fmt.Errorf("stylesheet file not accessible: %s", stylesheet)
		}
		args = append(args, "--stylesheet="+path)
	}

	args = append(args, "--media-type="+media)

	if presentationalHints {
		args = append(args, "--presentational-hints")
	}

	args = append(args, "-", "-")

	cmd := exec.Command("weasyprint", args...)
	cmd.Stdin = bytes.NewBufferString(data)
	var buffer bytes.Buffer
	cmd.Stdout = &buffer
	var errBuffer bytes.Buffer
	cmd.Stderr = &errBuffer
	err := cmd.Run()
	return &buffer, err
}

func PDFFromText(data string) (*bytes.Buffer, error) {
	const charsPerLine = 65
	const linesPerPage = 36

	data = strings.ReplaceAll(strings.ReplaceAll(data, "\r\n", "\n"), "\r", "\n")

	var pages [][]string
	var page []string

	add := func(line string) {
		page = append(page, line)
		if len(page) == linesPerPage {
			pages = append(pages, page)
			page = nil
		}
	}

	for raw := range strings.SplitSeq(data, "\n") {
		if raw == "" {
			add(raw)
			continue
		}
		for len(raw) > charsPerLine {
			add(raw[:charsPerLine])
			raw = raw[charsPerLine:]
		}
		add(raw)
	}

	if len(page) > 0 || len(pages) == 0 {
		pages = append(pages, page)
	}

	doc := map[string]any{
		"paper":  "A4P",
		"origin": "upperleft",
		"pages":  map[string]any{},
	}

	m := doc["pages"].(map[string]any)
	for i, lines := range pages {
		m[strconv.Itoa(i+1)] = map[string]any{
			"content": map[string]any{
				"text": []any{
					map[string]any{
						"value": strings.Join(lines, "\n"),
						"pos":   []int{50, 50},
						"width": 495,
						"font": map[string]any{
							"name": "Courier",
							"size": 18,
						},
					},
				},
			},
		}
	}

	b, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	var outBuffer bytes.Buffer
	err = pdfcpuApi.Create(nil, bytes.NewReader(b), &outBuffer, nil)
	return &outBuffer, err
}

//export PDF_free
func PDF_free(ptr *C.void) {
	h := cgo.Handle(uintptr(unsafe.Pointer(ptr)))
	h.Delete()
}

//export PDF_constructor
func PDF_constructor(hhc *C.HalonHSLContext, args *C.HalonHSLArguments, ret *C.HalonHSLValue) {
	var err error

	data, err := HSLArgumentGetString(args, 0, true)
	if err != nil {
		HSLValueSetException(hhc, err.Error())
		return
	}

	jsonOpts, err := HSLArgumentGetJSON(args, 1, false)
	if err != nil {
		HSLValueSetException(hhc, err.Error())
		return
	}

	opts := PDFConstructorOptions{}
	if jsonOpts != "" && jsonOpts != "[]" {
		err = json.Unmarshal([]byte(jsonOpts), &opts)
		if err != nil {
			HSLValueSetException(hhc, err.Error())
			return
		}
	}

	if opts.Format != "" {
		opts.Format = strings.ToLower(opts.Format)
		if opts.Format != "text/html" && opts.Format != "text/plain" {
			HSLValueSetException(hhc, "unsupported format: "+opts.Format)
			return
		}
	} else {
		opts.Format = "text/plain"
	}

	var buffer *bytes.Buffer
	switch opts.Format {
	case "text/html":
		protocols := []string{"data"}
		if opts.Protocols != nil {
			protocols = *opts.Protocols
		}

		stylesheets := []string{}
		if opts.Stylesheets != nil {
			stylesheets = *opts.Stylesheets
		}

		if opts.Media != "" {
			opts.Media = strings.ToLower(opts.Media)
			if opts.Media != "print" && opts.Media != "screen" {
				HSLValueSetException(hhc, "unsupported media: "+opts.Media)
				return
			}
		} else {
			opts.Media = "print"
		}

		if opts.PresentationalHints == nil {
			defaultPresentationalHints := true
			opts.PresentationalHints = &defaultPresentationalHints
		}

		buffer, err = PDFFromHTML(data, protocols, stylesheets, opts.Media, *opts.PresentationalHints)
	case "text/plain":
		buffer, err = PDFFromText(data)
	}
	if err != nil {
		HSLValueSetException(hhc, err.Error())
		return
	}

	reader := bytes.NewReader(buffer.Bytes())
	conf := pdfcpuModel.NewDefaultConfiguration()
	ctx, err := pdfcpuApi.ReadAndValidate(reader, conf)
	if err != nil {
		HSLValueSetException(hhc, err.Error())
		return
	}

	pdf := &PDF{ctx: ctx}

	hho := C.HalonMTA_hsl_object_new()
	HSLObjectTypeSet(hho, "PDF")
	h := cgo.NewHandle(pdf)
	C.HalonMTA_hsl_object_ptr_set(hho, *(*unsafe.Pointer)(unsafe.Pointer(&h)), HSLObjectFreeFunction(hho, "PDF_free"))
	HSLObjectRegisterFunction(hho, "addAttachment", "PDF_addAttachment")
	HSLObjectRegisterFunction(hho, "toString", "PDF_toString")
	C.HalonMTA_hsl_value_set(ret, C.HALONMTA_HSL_TYPE_OBJECT, unsafe.Pointer(hho), 0)
	C.HalonMTA_hsl_object_delete(hho)
}

//export PDF_addAttachment
func PDF_addAttachment(hhc *C.HalonHSLContext, args *C.HalonHSLArguments, ret *C.HalonHSLValue) {
	var err error

	id, err := HSLArgumentGetString(args, 0, true)
	if err != nil {
		HSLValueSetException(hhc, err.Error())
		return
	}

	data, err := HSLArgumentGetString(args, 1, true)
	if err != nil {
		HSLValueSetException(hhc, err.Error())
		return
	}

	jsonOpts, err := HSLArgumentGetJSON(args, 2, false)
	if err != nil {
		HSLValueSetException(hhc, err.Error())
		return
	}

	opts := PDFAddAttachmentOptions{}
	if jsonOpts != "" {
		err = json.Unmarshal([]byte(jsonOpts), &opts)
		if err != nil {
			HSLValueSetException(hhc, err.Error())
			return
		}
	}

	h := cgo.Handle(uintptr(C.HalonMTA_hsl_object_ptr_get(hhc)))
	pdf := h.Value().(*PDF)

	a := pdfcpuModel.Attachment{
		ID:     id,
		Desc:   opts.Desc,
		Reader: bytes.NewReader([]byte(data)),
	}

	err = pdf.ctx.AddAttachment(a, false)
	if err != nil {
		HSLValueSetException(hhc, err.Error())
		return
	}

	C.HalonMTA_hsl_value_set(ret, C.HALONMTA_HSL_TYPE_THIS, unsafe.Pointer(hhc), 0)
}

//export PDF_toString
func PDF_toString(hhc *C.HalonHSLContext, args *C.HalonHSLArguments, ret *C.HalonHSLValue) {
	var err error

	jsonOpts, err := HSLArgumentGetJSON(args, 0, false)
	if err != nil {
		HSLValueSetException(hhc, err.Error())
		return
	}

	opts := PDFtoStringOptions{}
	if jsonOpts != "" && jsonOpts != "[]" {
		err = json.Unmarshal([]byte(jsonOpts), &opts)
		if err != nil {
			HSLValueSetException(hhc, err.Error())
			return
		}
	}

	h := cgo.Handle(uintptr(C.HalonMTA_hsl_object_ptr_get(hhc)))
	pdf := h.Value().(*PDF)

	var val bytes.Buffer
	err = pdfcpuApi.WriteContext(pdf.ctx, &val)
	if err != nil {
		HSLValueSetException(hhc, err.Error())
		return
	}

	if opts.Password != "" {
		conf := pdfcpuModel.NewAESConfiguration(opts.Password, opts.Password, 256)
		conf.Permissions = pdfcpuModel.PermissionsAll
		var enc bytes.Buffer
		err = pdfcpuApi.Encrypt(bytes.NewReader(val.Bytes()), &enc, conf)
		if err != nil {
			HSLValueSetException(hhc, err.Error())
			return
		}
		val = enc
	}

	HSLValueSetString(ret, val.String())
}

//export Halon_init
func Halon_init(hic *C.HalonInitContext) C.bool {
	var raw_config *C.HalonConfig

	if C.HalonMTA_init_getinfo(hic, C.HALONMTA_INIT_APPCONFIG, nil, 0, unsafe.Pointer(&raw_config), nil) {
		cfg, err := GetConfigAsJSON(raw_config)
		if err != nil {
			if err.Error() != "failed to get config" {
				return false
			}
		} else {
			var parsed Config
			if err := json.Unmarshal([]byte(cfg), &parsed); err != nil {
				return false
			}

			configLock.Lock()
			config = &parsed
			configLock.Unlock()
		}
	}

	pdfcpuApi.DisableConfigDir()
	return true
}

//export Halon_config_reload
func Halon_config_reload(hc *C.HalonConfig) {
	cfg, err := GetConfigAsJSON(hc)

	if err != nil {
		if err.Error() != "failed to get config" {
			configLock.Lock()
			config = nil
			configLock.Unlock()
			return
		}
	}

	var parsed Config
	if err := json.Unmarshal([]byte(cfg), &parsed); err != nil {
		configLock.Lock()
		config = nil
		configLock.Unlock()
		return
	}

	configLock.Lock()
	config = &parsed
	configLock.Unlock()

}

//export Halon_hsl_register
func Halon_hsl_register(hhrc *C.HalonHSLRegisterContext) C.bool {
	HSLModuleRegisterFunction(hhrc, "PDF", "PDF_constructor")
	return true
}

//export Halon_version
func Halon_version() C.int {
	return C.HALONMTA_PLUGIN_VERSION
}

func main() {}
