package escpos

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
)

// text replacement map
var textReplaceMap = map[string]string{
	// horizontal tab
	"&#9;":  "\x09",
	"&#x9;": "\x09",

	// linefeed
	"&#10;": "\n",
	"&#xA;": "\n",

	// xml stuff
	"&apos;": "'",
	"&quot;": `"`,
	"&gt;":   ">",
	"&lt;":   "<",

	// ampersand must be last to avoid double decoding
	"&amp;": "&",
}

// replace text from the above map
func textReplace(data string) string {
	for k, v := range textReplaceMap {
		data = strings.Replace(data, k, v, -1)
	}
	return data
}

type Escpos struct {
	// destination
	dst io.Writer

	// font metrics
	width, height uint8

	// state toggles ESC[char]
	underline  uint8
	emphasize  uint8
	upsidedown uint8
	rotate     uint8

	// state toggles GS[char]
	reverse, smooth uint8
	s               string
	b               []byte
}

// reset toggles
func (e *Escpos) reset() {
	e.width = 1
	e.height = 1

	e.underline = 0
	e.emphasize = 0
	e.upsidedown = 0
	e.rotate = 0

	e.reverse = 0
	e.smooth = 0
}

// create Escpos printer
func New() (e *Escpos) {
	e = &Escpos{}
	e.reset()
	return
}

func (e *Escpos) SetPrintPic() {
	e.Write(fmt.Sprintf("\x1D*%c%c%v", 2, 2, "11111000001010101111100000101010"))
}

//打印下载位图
func (e *Escpos) PrintPic() {
	e.Write(fmt.Sprintf("\x1D/%c", 0))
}

//开钱箱
func (e *Escpos) OpenCashBox() {
	e.Write(fmt.Sprintf("\x1Bp%c%c%c", 0, 2, 4))
	e.Write(fmt.Sprintf("\x10\x14%c%c%c", 1, 0, 1))
}

//水平定位
func (e *Escpos) MoveBaseSize() {
	e.Write(fmt.Sprint("\x09"))
}

//设置横向和纵向移动单位
func (e *Escpos) SetMoveSize(x, y int) {
	e.Write(fmt.Sprintf("\x1D%c%c%c", 80, x, y))
	e.b = append(e.b, byte(x))
	e.b = append(e.b, byte(y))
}

//设置打印模式
func (e *Escpos) SetPrinterType(n int) {
	e.Write(fmt.Sprintf("\x1B!%c", n))
}

//设置行间距
func (e *Escpos) SetLineSpacing(n int) {
	e.Write(fmt.Sprintf("\x1B3%c", n))
}

//set location l+n*255
//设置绝对打印位置
func (e *Escpos) SetLocation(l, n int) {
	e.Write(fmt.Sprintf("\x1B$"))
	e.b = append(e.b, byte(l))
	e.b = append(e.b, byte(n))
}

//设置相对打印位置
func (e *Escpos) SetReLocation(l, n int) {
	e.Write(fmt.Sprintf("\x1B\\$"))
	e.b = append(e.b, byte(l))
	e.b = append(e.b, byte(n))
}

func (e *Escpos) PrintSplitLine(n int, s string) {
	t := s
	for i := 0; i < n; i++ {
		t += s
	}
	e.Write(t)
	e.Linefeed(1)
}

func (e *Escpos) Barcode(s string) {
	e.Write(fmt.Sprintf("\x1Dk%c%s%c", 4, s, 0))
	// e.Write(fmt.Sprintf("\x1Dk%c%c%s", 4, len(s), s))
	// e.Write(fmt.Sprintf("\x1Dk%c12345678910%c", 1, 0))
}

//set right space
func (e *Escpos) SendRightSpace(n int) {
	e.Write(fmt.Sprintf("\x1B%c%c", 32, n))
}

//print and feed paper
func (e *Escpos) PrintAndFeed(i int) {
	e.Write(fmt.Sprintf("\x1BJ%c", i))
}

//print barcode HRI
func (e *Escpos) BarcodeHRI(n int) {
	//0:no 1:top 2:down 3:top&&down
	e.Write(fmt.Sprintf("\x1DH%c", n))
}

//print barcode HRI font size
func (e *Escpos) BarcodeHRIFontSize(n int) {
	//0:A(12*24),1:B(9*17)
	e.Write(fmt.Sprintf("\x1Df%c", n))
}

//print barcode HRI font H
func (e *Escpos) BarcodeHigth(n int) {
	e.Write(fmt.Sprintf("\x1Dh%c", n))
}

// '// Print
// barcode >>>
// '// Select justification: Centering
// ESC "a" 1
// "<< Bonus points : 14 >>"
// '// Print and feed paper: Paper feeding amount = 4.94 mm (35/180 inches)
// ESC "J" 35
// '// Set barcode height: in case TMT20,
// 6.25 mm (50/203 inches)
// GS "h" 50
// '// Select print position of HRI characters: Print position, below the barcode
// GS "H" 2
// '// Select font for HRI characters: Font B
// GS "f" 1
// '// Print barcode: (A) format, barcode system = CODE39
// GS "k" 4 "*00014*" 0
// '// Print
// barcode <<<

// write raw bytes to b
func (e *Escpos) WriteRaw(data []byte) (n int, err error) {
	e.b = append(e.b, []byte(data)...)
	return 0, nil
}

// write a string to the b
func (e *Escpos) Write(data string) (int, error) {
	e.s += data
	return e.WriteRaw([]byte(data))
}

//read data by buffer
func (e *Escpos) Readbyte() (int, []byte) {
	return len(e.b), e.b
}

//read data by buffer
func (e *Escpos) ReadString() string {
	return e.s
}

//send b to printer
func (e *Escpos) SendTo(w io.Writer) (int, error) {
	return w.Write(e.b)
}

// init/reset printer settings
func (e *Escpos) Init() {
	e.reset()
	e.Write("\x1B@")
	e.SendEmphasize()
	e.SetPrinterType(0)
	e.SendRotate()
}

// end output
func (e *Escpos) End() {
	e.Write("\xFA")
}

// send cut
func (e *Escpos) Cut() {
	e.Write("\x1DVA0")
}

// send linefeed
func (e *Escpos) Linefeed(n int) {
	e.FormfeedN(n)
	//for i := 0; i < n; i++ {
	//e.Write("\n")
	//}
}

// send N formfeeds
func (e *Escpos) FormfeedN(n int) {
	e.Write(fmt.Sprintf("\x1Bd%c", n))
}

// send formfeed
func (e *Escpos) Formfeed() {
	e.FormfeedN(1)
}

// set font
func (e *Escpos) SetFont(font string) {
	f := 0

	switch font {
	case "A":
		f = 0
	case "B":
		f = 1
	default:
		log.Fatal(fmt.Sprintf("Invalid font: '%s', defaulting to 'A'", font))
		f = 0
	}

	e.Write(fmt.Sprintf("\x1BM%c", f))
}

func (e *Escpos) SendFontSize() {
	e.Write(fmt.Sprintf("\x1D!%c", ((e.width-1)<<4)|(e.height-1)))
}

// set font size
func (e *Escpos) SetFontSize(width, height uint8) {
	if width > 0 && height > 0 && width <= 8 && height <= 8 {
		e.width = width
		e.height = height
		e.SendFontSize()
	} else {
		log.Fatal(fmt.Sprintf("Invalid font size passed: %d x %d", width, height))
	}
}

// send underline
func (e *Escpos) SendUnderline() {
	e.Write(fmt.Sprintf("\x1B-%c", e.underline))
}

// send emphasize / doublestrike
func (e *Escpos) SendEmphasize() {
	e.Write(fmt.Sprintf("\x1BG%c", e.emphasize))
}

// send upsidedown
func (e *Escpos) SendUpsidedown() {
	e.Write(fmt.Sprintf("\x1B{%c", e.upsidedown))
}

// send rotate
func (e *Escpos) SendRotate() {
	e.Write(fmt.Sprintf("\x1BR%c", e.rotate))
}

// send reverse
func (e *Escpos) SendReverse() {
	e.Write(fmt.Sprintf("\x1DB%c", e.reverse))
}

// send smooth
func (e *Escpos) SendSmooth() {
	e.Write(fmt.Sprintf("\x1Db%c", e.smooth))
}

// send move x
func (e *Escpos) SendMoveX(x uint16) {
	e.Write(string([]byte{0x1b, 0x24, byte(x % 256), byte(x / 256)}))
}

// send move y
func (e *Escpos) SendMoveY(y uint16) {
	e.Write(string([]byte{0x1d, 0x24, byte(y % 256), byte(y / 256)}))
}

// set underline
func (e *Escpos) SetUnderline(v uint8) {
	e.underline = v
	e.SendUnderline()
}

// set emphasize
func (e *Escpos) SetEmphasize(u uint8) {
	e.emphasize = u
	e.SendEmphasize()
}

// set upsidedown
func (e *Escpos) SetUpsidedown(v uint8) {
	e.upsidedown = v
	e.SendUpsidedown()
}

// set rotate
func (e *Escpos) SetRotate(v uint8) {
	e.rotate = v
	e.SendRotate()
}

// set reverse
func (e *Escpos) SetReverse(v uint8) {
	e.reverse = v
	e.SendReverse()
}

// set smooth
func (e *Escpos) SetSmooth(v uint8) {
	e.smooth = v
	e.SendSmooth()
}

// pulse (open the drawer)
func (e *Escpos) Pulse() {
	// with t=2 -- meaning 2*2msec
	e.Write("\x1Bp\x02")
}

// set alignment
func (e *Escpos) SetAlign(align string) {
	a := 0
	switch align {
	case "left":
		a = 0
	case "center":
		a = 1
	case "right":
		a = 2
	default:
		log.Fatal(fmt.Sprintf("Invalid alignment: %s", align))
	}
	e.Write(fmt.Sprintf("\x1Ba%c", a))
}

// set language -- ESC R
func (e *Escpos) SetLang(lang string) {
	l := 0

	switch lang {
	case "en":
		l = 0
	case "fr":
		l = 1
	case "de":
		l = 2
	case "uk":
		l = 3
	case "da":
		l = 4
	case "sv":
		l = 5
	case "it":
		l = 6
	case "es":
		l = 7
	case "ja":
		l = 8
	case "no":
		l = 9
	default:
		log.Fatal(fmt.Sprintf("Invalid language: %s", lang))
	}
	e.Write(fmt.Sprintf("\x1BR%c", l))
}

// do a block of text
func (e *Escpos) Text(params map[string]string, data string) {

	// send alignment to printer
	if align, ok := params["align"]; ok {
		e.SetAlign(align)
	}

	// set lang
	if lang, ok := params["lang"]; ok {
		e.SetLang(lang)
	}

	// set smooth
	if smooth, ok := params["smooth"]; ok && (smooth == "true" || smooth == "1") {
		e.SetSmooth(1)
	}

	// set emphasize
	if em, ok := params["em"]; ok && (em == "true" || em == "1") {
		e.SetEmphasize(1)
	}

	// set underline
	if ul, ok := params["ul"]; ok && (ul == "true" || ul == "1") {
		e.SetUnderline(1)
	}

	// set reverse
	if reverse, ok := params["reverse"]; ok && (reverse == "true" || reverse == "1") {
		e.SetReverse(1)
	}

	// set rotate
	if rotate, ok := params["rotate"]; ok && (rotate == "true" || rotate == "1") {
		e.SetRotate(1)
	}

	// set font
	if font, ok := params["font"]; ok {
		e.SetFont(strings.ToUpper(font[5:6]))
	}

	// do dw (double font width)
	if dw, ok := params["dw"]; ok && (dw == "true" || dw == "1") {
		e.SetFontSize(2, e.height)
	}

	// do dh (double font height)
	if dh, ok := params["dh"]; ok && (dh == "true" || dh == "1") {
		e.SetFontSize(e.width, 2)
	}

	// do font width
	if width, ok := params["width"]; ok {
		if i, err := strconv.Atoi(width); err == nil {
			e.SetFontSize(uint8(i), e.height)
		} else {
			log.Fatal(fmt.Sprintf("Invalid font width: %s", width))
		}
	}

	// do font height
	if height, ok := params["height"]; ok {
		if i, err := strconv.Atoi(height); err == nil {
			e.SetFontSize(e.width, uint8(i))
		} else {
			log.Fatal(fmt.Sprintf("Invalid font height: %s", height))
		}
	}

	// do y positioning
	if x, ok := params["x"]; ok {
		if i, err := strconv.Atoi(x); err == nil {
			e.SendMoveX(uint16(i))
		} else {
			log.Fatal("Invalid x param %d", x)
		}
	}

	// do y positioning
	if y, ok := params["y"]; ok {
		if i, err := strconv.Atoi(y); err == nil {
			e.SendMoveY(uint16(i))
		} else {
			log.Fatal("Invalid y param %d", y)
		}
	}

	// do text replace, then write data
	data = textReplace(data)
	if len(data) > 0 {
		e.Write(data)
	}
}

// feed the printer
func (e *Escpos) Feed(params map[string]string) {
	// handle lines (form feed X lines)
	if l, ok := params["line"]; ok {
		if i, err := strconv.Atoi(l); err == nil {
			e.FormfeedN(i)
		} else {
			log.Fatal(fmt.Sprintf("Invalid line number %d", l))
		}
	}

	// handle units (dots)
	if u, ok := params["unit"]; ok {
		if i, err := strconv.Atoi(u); err == nil {
			e.SendMoveY(uint16(i))
		} else {
			log.Fatal(fmt.Sprintf("Invalid unit number %d", u))
		}
	}

	// send linefeed
	e.Linefeed(1)

	// reset variables
	e.reset()

	// reset printer
	e.SendEmphasize()
	e.SendRotate()
	e.SendSmooth()
	e.SendReverse()
	e.SendUnderline()
	e.SendUpsidedown()
	e.SendFontSize()
	e.SendUnderline()
}

// feed and cut based on parameters
func (e *Escpos) FeedAndCut(params map[string]string) {
	if t, ok := params["type"]; ok && t == "feed" {
		e.Formfeed()
	}

	e.Cut()
}

// used to send graphics headers
func (e *Escpos) gSend(m byte, fn byte, data []byte) {
	l := len(data) + 2

	e.Write("\x1b(L")
	e.WriteRaw([]byte{byte(l % 256), byte(l / 256), m, fn})
	e.WriteRaw(data)
}

// write an image
func (e *Escpos) Image(params map[string]string, data string) {
	// send alignment to printer
	if align, ok := params["align"]; ok {
		e.SetAlign(align)
	}

	// get width
	wstr, ok := params["width"]
	if !ok {
		log.Fatal("No width specified on image")
	}

	// get height
	hstr, ok := params["height"]
	if !ok {
		log.Fatal("No height specified on image")
	}

	// convert width
	width, err := strconv.Atoi(wstr)
	if err != nil {
		log.Fatal("Invalid image width %s", wstr)
	}

	// convert height
	height, err := strconv.Atoi(hstr)
	if err != nil {
		log.Fatal("Invalid image height %s", hstr)
	}

	// decode data frome b64 string
	dec, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Image len:%d w: %d h: %d\n", len(dec), width, height)

	// $imgHeader = self::dataHeader(array($img -> getWidth(), $img -> getHeight()), true);
	// $tone = '0';
	// $colors = '1';
	// $xm = (($size & self::IMG_DOUBLE_WIDTH) == self::IMG_DOUBLE_WIDTH) ? chr(2) : chr(1);
	// $ym = (($size & self::IMG_DOUBLE_HEIGHT) == self::IMG_DOUBLE_HEIGHT) ? chr(2) : chr(1);
	//
	// $header = $tone . $xm . $ym . $colors . $imgHeader;
	// $this -> graphicsSendData('0', 'p', $header . $img -> toRasterFormat());
	// $this -> graphicsSendData('0', '2');

	header := []byte{
		byte('0'), 0x01, 0x01, byte('1'),
	}

	a := append(header, dec...)

	e.gSend(byte('0'), byte('p'), a)
	e.gSend(byte('0'), byte('2'), []byte{})

}

// write a "node" to the printer
func (e *Escpos) WriteNode(name string, params map[string]string, data string) {
	cstr := ""
	if data != "" {
		str := data[:]
		if len(data) > 40 {
			str = fmt.Sprintf("%s ...", data[0:40])
		}
		cstr = fmt.Sprintf(" => '%s'", str)
	}
	log.Printf("Write: %s => %+v%s\n", name, params, cstr)

	switch name {
	case "text":
		e.Text(params, data)
	case "feed":
		e.Feed(params)
	case "cut":
		e.FeedAndCut(params)
	case "pulse":
		e.Pulse()
	case "image":
		e.Image(params, data)
	}
}

//print Qrcode
func (e *Escpos) QrCode() {
	e.Write(fmt.Sprintf("\x1BZ\x02%c%c\x00%chttp://www.baidu.com", 100, 1, 20))
}
