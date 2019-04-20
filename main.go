package main

import (
	"fmt"
	"github.com/939999807/cv-go/statusbar"
	"gocv.io/x/gocv"
	"image"
	"image/color"
)

func main() {

	// open webcam
	webcam, err := gocv.VideoCaptureDevice(0)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer webcam.Close()

	webcam.Set(gocv.VideoCaptureFrameWidth, 640)
	webcam.Set(gocv.VideoCaptureFrameHeight, 480)

	// open display window
	window := gocv.NewWindow("Face Detect")
	defer window.Close()

	// prepare image matrix
	img := gocv.NewMat()
	defer img.Close()

	// color for the rect when faces detected
	blue := color.RGBA{0, 0, 255, 0}

	// load classifier to recognize faces
	classifier := gocv.NewCascadeClassifier()
	defer classifier.Close()

	if !classifier.Load("conf/haarcascade_frontalface_default.xml") {
		fmt.Printf("Error reading cascade file: %v\n", "conf/haarcascade_frontalface_default.xml")
		return
	}

	fmt.Printf("start reading camera device: %v\n", 0)

	signalIconDrawer := statusbar.NewSignalIconDrawer(16, 4)
	clockIconDrawer := statusbar.NewClockIconDrawer(72.0, 12.5)
	statusBarDrawer := statusbar.NewBarDrawer(20, []int{4, 8}, signalIconDrawer, clockIconDrawer)

	for {
		if ok := webcam.Read(&img); !ok {
			fmt.Printf("cannot read device %d\n", 0)
			return
		}
		if img.Empty() {
			continue
		}

		// detect faces
		rects := classifier.DetectMultiScale(img)
		fmt.Printf("found %d faces\n", len(rects))

		// draw a rectangle around each face on the original image,
		// along with text identifying as "Human"

		statusBarDrawer.Refresh(&img)
		statusBarDrawer.Draw(&img)

		for _, r := range rects {

			//gocv.DrawArc(&Img, Arc, StartPoint, EndPoint, 2);

			gocv.Rectangle(&img, r, blue, 3)

			size := gocv.GetTextSize("Human", gocv.FontHersheyPlain, 1.2, 2)
			pt := image.Pt(r.Min.X+(r.Min.X/2)-(size.X/2), r.Min.Y-2)
			gocv.PutText(&img, "Human", pt, gocv.FontHersheyPlain, 1.2, blue, 2)
		}

		// show the image in the window, and wait 1 millisecond

		window.IMShow(img)
		if window.WaitKey(1) >= 0 {
			break
		}
	}
}
