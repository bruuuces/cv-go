package statusbar

import (
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"gocv.io/x/gocv"
	"golang.org/x/image/font"
	"image"
	"image/color"
	"image/draw"
	"io/ioutil"
	"log"
	"math"
	"time"
)

type BarDrawer struct {
	lightThemeCache bool

	init             bool
	img              *gocv.Mat
	statusBarHeight  int
	statusBarPadding []int
	drawers          []IconDrawer
}

func NewBarDrawer(statusBarHeight int, statusBarPadding []int, drawers ...IconDrawer) *BarDrawer {
	return &BarDrawer{statusBarHeight: statusBarHeight, statusBarPadding: statusBarPadding, drawers: drawers}
}

func (d *BarDrawer) CheckRefresh(img *gocv.Mat) bool {
	if d.RefreshTheme(img) {
		return true
	}
	if !d.init {
		return true
	}
	for _, drawer := range d.drawers {
		if drawer.CheckRefresh() {
			return true
		}
	}
	return false
}

func (d *BarDrawer) RefreshTheme(img *gocv.Mat) bool {
	gocv.CvtColor(*img, img, gocv.ColorRGBAToBGR);
	gocv.CvtColor(*img, img, gocv.ColorBGRToRGBA);
	img.ConvertTo(img, gocv.MatTypeCV8UC4)

	roiImg := img.Region(image.Rect(0, 0, img.Cols(), d.statusBarHeight))
	gocv.CvtColor(roiImg, &roiImg, gocv.ColorBGRToGray);
	gocv.Threshold(roiImg, &roiImg, 128, 255, gocv.ThresholdBinary)
	numOfBlack := 0;
	for i := 0; i < roiImg.Rows(); i++ {
		for j := 0; j < roiImg.Cols(); j++ { //遍历图片的每一个像素点
			c := roiImg.GetUCharAt(i, j)
			if c < 128 {
				numOfBlack++
			}
		}
	}
	rate := float32(numOfBlack) / float32(roiImg.Rows()*roiImg.Cols())
	theme := rate > 0.5
	defer func() {
		d.lightThemeCache = theme
	}()
	return d.lightThemeCache == theme
}

func (d *BarDrawer) Refresh(img *gocv.Mat) {
	if !d.CheckRefresh(img) {
		return
	}
	gocv.CvtColor(*img, img, gocv.ColorRGBAToBGR);
	gocv.CvtColor(*img, img, gocv.ColorBGRToRGBA);
	img.ConvertTo(img, gocv.MatTypeCV8UC4)

	var fgColor color.RGBA
	var fgShadingColor color.RGBA
	var bgColor color.RGBA

	if d.lightThemeCache {
		fgColor = color.RGBA{60, 63, 65, 255}
		fgShadingColor = color.RGBA{200, 200, 200, 0}
		bgColor = color.RGBA{236, 236, 236, 0}
	} else {
		fgColor = color.RGBA{255, 255, 255, 255}
		fgShadingColor = color.RGBA{130, 130, 130, 0}
		bgColor = color.RGBA{60, 63, 65, 0}
	}

	statusBarIconHeight := d.statusBarHeight - d.statusBarPadding[0]*2
	statusBarImg := gocv.NewMatWithSize(d.statusBarHeight, img.Cols(), img.Type())
	gocv.Rectangle(&statusBarImg, image.Rect(0, 0, statusBarImg.Cols(), statusBarImg.Rows()), bgColor, -1)

	rowPos := d.statusBarPadding[1]
	colPos := d.statusBarPadding[0]
	for _, drawer := range d.drawers {
		iconImg := statusBarImg.Region(image.Rect(rowPos, colPos, statusBarImg.Cols(), statusBarIconHeight+colPos))
		rowPos += drawer.Draw(&iconImg, fgColor, fgShadingColor, bgColor)
		rowPos += d.statusBarPadding[1]
	}
	if d.img != nil {
		defer d.img.Close()
	}
	d.img = &statusBarImg
	d.init = true
}

func (d *BarDrawer) Draw(img *gocv.Mat) {
	gocv.CvtColor(*img, img, gocv.ColorRGBAToBGR);
	gocv.CvtColor(*img, img, gocv.ColorBGRToRGBA);
	gocv.Vconcat(*d.img, *img, img)
}

type IconDrawer interface {
	CheckRefresh() bool
	Draw(*gocv.Mat, color.RGBA, color.RGBA, color.RGBA) int
}

type SignalIconDrawer struct {
	signalCache           int
	tmpCurrentSignalCache float64

	iconWidth      int
	maxSignalValue int
}

func NewSignalIconDrawer(iconWidth int, maxSignalValue int) *SignalIconDrawer {
	return &SignalIconDrawer{iconWidth: iconWidth, maxSignalValue: maxSignalValue}
}

func (d *SignalIconDrawer) CheckRefresh() bool {
	d.tmpCurrentSignalCache += 0.1
	signal := int(d.tmpCurrentSignalCache) % 5
	if signal != d.signalCache {
		return true
	}
	return false
}

func (d *SignalIconDrawer) Draw(img *gocv.Mat, fgColor color.RGBA, fgShadingColor color.RGBA, bgColor color.RGBA) int {
	signalImg := img.Region(image.Rect(0, 0, d.iconWidth, img.Rows()))
	d.tmpCurrentSignalCache += 0.1
	signal := int(d.tmpCurrentSignalCache) % 5
	for i := 0; i < d.maxSignalValue; i++ {
		startPt := image.Pt(1+i*(signalImg.Cols()/d.maxSignalValue), (signalImg.Rows()-4)/d.maxSignalValue*(d.maxSignalValue-i)-1)
		endPt := image.Pt(1+i*(signalImg.Cols()/d.maxSignalValue), signalImg.Rows()-1)
		if i < signal {
			gocv.Line(&signalImg, startPt, endPt, fgColor, 2)
		} else {
			gocv.Line(&signalImg, startPt, endPt, fgShadingColor, 2)
		}
	}
	d.signalCache = signal
	return d.iconWidth
}

type ClockIconDrawer struct {
	minuteCache int64

	dpi      float64
	fontSize float64
}

func NewClockIconDrawer(dpi float64, fontSize float64) *ClockIconDrawer {
	return &ClockIconDrawer{minuteCache: 0, dpi: dpi, fontSize: fontSize}
}

func (d *ClockIconDrawer) CheckRefresh() bool {
	minute := time.Now().Unix() / 60
	if minute != d.minuteCache {
		return true
	}
	return false
}

func (d *ClockIconDrawer) Draw(img *gocv.Mat, fgColor color.RGBA, fgShadingColor color.RGBA, bgColor color.RGBA) int {
	fontBytes, err := ioutil.ReadFile("conf/font/Nunito-Bold.ttf")
	if err != nil {
		log.Println(err)
		return 0
	}
	f, err := freetype.ParseFont(fontBytes)
	if err != nil {
		log.Println(err)
		return 0
	}
	opt := &truetype.Options{
		DPI:  d.dpi,
		Size: d.fontSize,
	}
	face := truetype.NewFace(f, opt)

	now := time.Now()
	minute := now.Unix() / 60
	text := now.Format("15:04")

	width := font.MeasureString(face, text).Ceil() + 2
	c := freetype.NewContext()
	c.SetDPI(d.dpi)
	c.SetFont(f)
	c.SetFontSize(d.fontSize)
	c.SetSrc(image.NewUniform(fgColor))

	rgba := image.NewRGBA(image.Rect(0, 0, width, int(math.Floor(d.fontSize))))
	draw.Draw(rgba, rgba.Bounds(), image.NewUniform(bgColor), image.ZP, draw.Src)
	c.SetClip(rgba.Bounds())
	c.SetDst(rgba)
	c.SetHinting(font.HintingFull)

	// Draw the text.
	pt := freetype.Pt(0, int(c.PointToFixed(d.fontSize)>>6))
	_, err = c.DrawString(text, pt)
	if err != nil {
		log.Println(err)
		return 0
	}
	mat, err := gocv.ImageToMatRGBA(rgba)

	roiImg := img.Region(rgba.Rect)

	gocv.Resize(mat, &roiImg, rgba.Rect.Size(), 0, 0, gocv.InterpolationLinear)

	d.minuteCache = minute
	return width
}
