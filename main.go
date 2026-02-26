package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"os"

	"github.com/D4ario0/go-qrcode"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

func main() {
	qr, err := qrcode.New("https://pdf.elemen.id/4gJLEXjYgJXXJtrQuMjysPmFJV5PyJYh", qrcode.Medium)
	if err != nil {
		log.Fatal(err)
	}

	tempQR := "temp_qr.png"

	qr.Roundness = 0        // clamp occurs when rendering
	qr.QuietZoneSize = -1   // set -1 to restore version default, < -1 clamps to 0
	qr.DisableBorder = true // true removes the quiet zone entirely
	qr.ForegroundColor = color.Black
	qr.BackgroundColor = color.Transparent

	png, err := qr.PNG(120)
	if err != nil {
		log.Fatal(err)
	}

	if err := os.WriteFile(tempQR, png, 0o644); err != nil {
		log.Fatal(err)
	}

	defer os.Remove(tempQR)

	// 2. Combine with Background Logo
	log.Println("Adding background logo...")
	if err := addBackgroundToQR(tempQR, "logo_bsre.png", "hasil_qr_bsre.png"); err != nil {
		log.Fatalf("Gagal menggabungkan gambar: %v", err)
	}

	// 3. Add Text to the right of the QR code
	log.Println("Adding text to the right...")
	if err := addTextToRightSide("hasil_qr_bsre.png", "final_qr_with_text.png"); err != nil {
		log.Fatalf("Gagal menambahkan teks: %v", err)
	}

	fmt.Println("Sukses! QR Code final telah dibuat: final_qr_with_text.png")
}

func addBackgroundToQR(qrPath, logoPath, outPath string) error {
	// Open QR File
	qrFile, err := os.Open(qrPath)
	if err != nil {
		return fmt.Errorf("failed to open QR: %w", err)
	}
	defer qrFile.Close()
	qrImg, err := png.Decode(qrFile)
	if err != nil {
		return fmt.Errorf("failed to decode QR: %w", err)
	}

	// Open Logo File
	logoFile, err := os.Open(logoPath)
	if err != nil {
		return fmt.Errorf("failed to open logo: %w", err)
	}
	defer logoFile.Close()
	logoImg, _, err := image.Decode(logoFile)
	if err != nil {
		return fmt.Errorf("failed to decode logo: %w", err)
	}

	// Create output canvas same size as QR
	bounds := qrImg.Bounds()
	outputImg := image.NewRGBA(bounds)

	// 1. Resize Logo to fit canvas
	resizedLogo := image.NewRGBA(bounds)
	// Use CatmullRom for high quality resizing
	xdraw.CatmullRom.Scale(resizedLogo, bounds, logoImg, logoImg.Bounds(), xdraw.Over, nil)

	// 2. Draw Resized Logo with 50% Opacity (Alpha 128)
	mask := image.NewUniform(color.Alpha{128})
	draw.DrawMask(outputImg, bounds, resizedLogo, image.Point{}, mask, image.Point{}, draw.Over)

	// 3. Draw QR Code on top (Normal Opacity, but transparent BG)
	draw.Draw(outputImg, bounds, qrImg, image.Point{}, draw.Over)

	// Save Output
	outFile, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	return png.Encode(outFile, outputImg)
}

func addTextToRightSide(qrPath, outPath string) error {
	qrFile, err := os.Open(qrPath)
	if err != nil {
		return fmt.Errorf("failed to open qr: %w", err)
	}
	defer qrFile.Close()
	qrImg, err := png.Decode(qrFile)
	if err != nil {
		return fmt.Errorf("failed to decode qr: %w", err)
	}
	qrBounds := qrImg.Bounds()

	// Parse font
	f, err := opentype.Parse(goregular.TTF)
	if err != nil {
		return fmt.Errorf("failed to parse font: %w", err)
	}

	lines := []string{
		"Dokumen ini ditandatangani",
		"secara elektronik menggunakan",
		"sertifikat tanda tangan elektronik",
		"yang diterbitkan oleh BSrE-BSSN.",
	}

	var face font.Face
	var lineHeight, totalHeight, maxWidth int

	// Loop to find the maximum font size that lets the 4 lines fit completely within the QR code's exact height.
	for s := 4.0; s <= 200.0; s += 1.0 {
		tempFace, err := opentype.NewFace(f, &opentype.FaceOptions{
			Size:    s,
			DPI:     72,
			Hinting: font.HintingFull,
		})
		if err != nil {
			continue
		}

		metrics := tempFace.Metrics()
		ascent := metrics.Ascent.Ceil()
		descent := metrics.Descent.Ceil()
		lh := metrics.Height.Ceil()
		th := (len(lines)-1)*lh + ascent + descent

		if th > qrBounds.Dy() {
			tempFace.Close()
			break
		}
		if face != nil {
			face.Close()
		}
		face = tempFace
		lineHeight = lh
		totalHeight = th

		mw := 0
		for _, line := range lines {
			w := font.MeasureString(face, line).Ceil()
			if w > mw {
				mw = w
			}
		}
		maxWidth = mw
	}

	if face == nil {
		return fmt.Errorf("failed to find suitable font size")
	}
	defer face.Close()

	// Colors
	textColor := color.RGBA{48, 134, 198, 255} // Blue matching the BSrE logo

	spacing := int(float64(qrBounds.Dx()) * 0.08) // Spacing relative to QR size (approx 25px for 300px QR = ~10px for 120px QR)
	outWidth := qrBounds.Dx() + spacing + maxWidth
	outHeight := qrBounds.Dy()

	// image.NewRGBA defaults to fully transparent black, keeping background transparent
	outImg := image.NewRGBA(image.Rect(0, 0, outWidth, outHeight))

	// Draw QR code centered vertically on the left
	qrStartY := (outHeight - qrBounds.Dy()) / 2
	draw.Draw(outImg, image.Rect(0, qrStartY, qrBounds.Dx(), qrStartY+qrBounds.Dy()), qrImg, image.Point{}, draw.Over)

	// Draw text
	d := &font.Drawer{
		Dst:  outImg,
		Src:  image.NewUniform(textColor),
		Face: face,
	}

	// Calculate vertical START position so it sits perfectly matched in height
	startY := (outHeight-totalHeight)/2 + face.Metrics().Ascent.Ceil()

	for i, line := range lines {
		d.Dot = fixed.Point26_6{
			X: fixed.I(qrBounds.Dx() + spacing),
			Y: fixed.I(startY + i*lineHeight),
		}
		d.DrawString(line)
	}

	// Save Output
	outFile, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	return png.Encode(outFile, outImg)
}
