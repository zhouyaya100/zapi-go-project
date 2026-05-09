package core

import (
	"bytes"
	cryptoRand "crypto/rand"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"strings"
	"sync"
	"time"
)

var (
	captchaStore   = map[string]captchaEntry{}
	captchaStoreMu sync.RWMutex
)
type captchaEntry struct { Code string; Expires time.Time }

func GenerateCaptcha() (string, []byte) {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 32); randBytes := make([]byte, 32); cryptoRand.Read(randBytes); for i := range b { b[i] = chars[int(randBytes[i])%len(chars)] }
	id := string(b)
	code := make([]byte, 4); codeRand := make([]byte, 4); cryptoRand.Read(codeRand); for i := range code { code[i] = chars[int(codeRand[i])%len(chars)] }
	captchaStoreMu.Lock()
	if len(captchaStore) > 1000 { now := time.Now(); for k, v := range captchaStore { if v.Expires.Before(now) { delete(captchaStore, k) } } }
	captchaStore[id] = captchaEntry{Code: string(code), Expires: time.Now().Add(5 * time.Minute)}
	captchaStoreMu.Unlock()
	return id, genCaptchaImg(string(code))
}

func VerifyCaptcha(id, code string) bool {
	captchaStoreMu.Lock(); defer captchaStoreMu.Unlock()
	e, ok := captchaStore[id]; if !ok { return false }
	delete(captchaStore, id)
	return time.Now().Before(e.Expires) && strings.EqualFold(e.Code, code)
}

var bmpFont = map[byte][]string{
	'A':{"01110","10001","10001","11111","10001","10001","10001"},
	'B':{"11110","10001","10001","11110","10001","10001","11110"},
	'C':{"01110","10001","10000","10000","10000","10001","01110"},
	'D':{"11100","10010","10001","10001","10001","10010","11100"},
	'E':{"11111","10000","10000","11110","10000","10000","11111"},
	'F':{"11111","10000","10000","11110","10000","10000","10000"},
	'G':{"01110","10001","10000","10111","10001","10001","01110"},
	'H':{"10001","10001","10001","11111","10001","10001","10001"},
	'J':{"00111","00010","00010","00010","00010","10010","01100"},
	'K':{"10001","10010","10100","11000","10100","10010","10001"},
	'L':{"10000","10000","10000","10000","10000","10000","11111"},
	'M':{"10001","11011","10101","10101","10001","10001","10001"},
	'N':{"10001","11001","10101","10011","10001","10001","10001"},
	'P':{"11110","10001","10001","11110","10000","10000","10000"},
	'Q':{"01110","10001","10001","10001","10101","10010","01101"},
	'R':{"11110","10001","10001","11110","10100","10010","10001"},
	'S':{"01110","10001","10000","01110","00001","10001","01110"},
	'T':{"11111","00100","00100","00100","00100","00100","00100"},
	'U':{"10001","10001","10001","10001","10001","10001","01110"},
	'V':{"10001","10001","10001","10001","01010","01010","00100"},
	'W':{"10001","10001","10001","10101","10101","11011","10001"},
	'X':{"10001","10001","01010","00100","01010","10001","10001"},
	'Y':{"10001","10001","01010","00100","00100","00100","00100"},
	'Z':{"11111","00001","00010","00100","01000","10000","11111"},
	'2':{"01110","10001","00001","00010","00100","01000","11111"},
	'3':{"01110","10001","00001","00110","00001","10001","01110"},
	'4':{"00010","00110","01010","10010","11111","00010","00010"},
	'5':{"11111","10000","11110","00001","00001","10001","01110"},
	'6':{"00110","01000","10000","11110","10001","10001","01110"},
	'7':{"11111","00001","00010","00100","01000","01000","01000"},
	'8':{"01110","10001","10001","01110","10001","10001","01110"},
	'9':{"01110","10001","10001","01111","00001","00010","01100"},
}

func genCaptchaImg(code string) []byte {
	w, h := 120, 40; img := image.NewRGBA(image.Rect(0,0,w,h))
	for y:=0;y<h;y++{for x:=0;x<w;x++{img.Set(x,y,color.RGBA{240,240,240,255})}}
	for i,ch:=range code {
		rows,ok:=bmpFont[byte(ch)]; if !ok{continue}
		ox:=10+i*26; cr:=uint8(30+rand.Intn(90)); cg:=uint8(30+rand.Intn(90)); cb:=uint8(30+rand.Intn(90))
		for row,line:=range rows{for col,c:=range line{if c=='1'{for dy:=0;dy<4;dy++{for dx:=0;dx<4;dx++{img.Set(ox+col*4+dx,4+row*4+dy,color.RGBA{cr,cg,cb,255})}}}}}
	}
	for i:=0;i<4;i++{x1,y1:=rand.Intn(w),rand.Intn(h);x2,y2:=rand.Intn(w),rand.Intn(h);steps:=iMax(iAbs(x2-x1),iAbs(y2-y1))+1;for s:=0;s<=steps;s++{t:=float64(s)/float64(steps);x:=int(float64(x1)+t*float64(x2-x1));y:=int(float64(y1)+t*float64(y2-y1));if x>=0&&x<w&&y>=0&&y<h{img.Set(x,y,color.RGBA{180,180,180,255})}}}
	for i:=0;i<50;i++{img.Set(rand.Intn(w),rand.Intn(h),color.RGBA{150,150,150,255})}
	var buf bytes.Buffer; png.Encode(&buf, img); return buf.Bytes()
}
func iMax(a,b int)int{if a>b{return a};return b}
func iAbs(a int)int{if a<0{return -a};return a}
