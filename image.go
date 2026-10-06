package main

import (
	"bytes"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"math"
	"net/http"
	"strconv"
)

func nativeResolution(w, h int) bool {
	for _, size := range [][2]int{{1920, 1200}, {1920, 1080}, {1366, 768}, {1280, 768}, {1080, 1920}, {1024, 768}, {800, 600}, {800, 480}, {768, 1366}, {768, 1280}, {768, 1024}, {720, 1280}, {640, 480}, {640, 360}, {480, 800}, {400, 240}, {320, 240}, {320, 180}, {240, 400}, {240, 320}, {240, 240}, {200, 200}, {150, 150}, {3840, 2160}} {
		if size[0] == w && size[1] == h {
			return true
		}
	}
	return false
}

func (a *App) transformedImage(w http.ResponseWriter, r *http.Request, p Photo, width, height, radius int) {
	select {
	case a.imageSlots <- struct{}{}:
		defer func() { <-a.imageSlots }()
	default:
		a.apiError(w, 503, "图片处理繁忙，请稍后重试", "")
		return
	}
	res, err := a.fetchImage(r, p, 1920, 1080)
	if err != nil {
		a.apiError(w, 502, err.Error(), "")
		return
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, (20<<20)+1))
	if err != nil || len(data) > 20<<20 {
		a.apiError(w, 502, "图片读取失败", "")
		return
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width < 1 || cfg.Height < 1 || cfg.Width > 8192 || cfg.Height > 8192 || cfg.Width*cfg.Height > 16000000 {
		a.apiError(w, 502, "无效或过大的源图片", "")
		return
	}
	source, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		a.apiError(w, 502, "图片解码失败", "")
		return
	}
	result := resizeCover(source, width, height)
	if radius > 0 {
		result = boxBlur(result, radius)
	}
	if r.Context().Err() != nil {
		return
	}
	var output bytes.Buffer
	if err = jpeg.Encode(&output, result, &jpeg.Options{Quality: 90}); err != nil {
		a.apiError(w, 500, "图片处理失败", "")
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Content-Length", strconv.Itoa(output.Len()))
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = output.WriteTo(w)
}

// Bilinear scaling with a centered crop provides arbitrary dimensions without an OSS service.
func resizeCover(src image.Image, w, h int) *image.RGBA {
	b := src.Bounds()
	scale := math.Max(float64(w)/float64(b.Dx()), float64(h)/float64(b.Dy()))
	ox := (float64(b.Dx()) - float64(w)/scale) / 2
	oy := (float64(b.Dy()) - float64(h)/scale) / 2
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		sy := max(0.0, min(float64(b.Dy()-1), oy+(float64(y)+0.5)/scale-0.5))
		y0 := int(sy)
		y1 := min(y0+1, b.Dy()-1)
		fy := sy - float64(y0)
		for x := 0; x < w; x++ {
			sx := max(0.0, min(float64(b.Dx()-1), ox+(float64(x)+0.5)/scale-0.5))
			x0 := int(sx)
			x1 := min(x0+1, b.Dx()-1)
			fx := sx - float64(x0)
			r0, g0, b0, _ := src.At(b.Min.X+x0, b.Min.Y+y0).RGBA()
			r1, g1, b1, _ := src.At(b.Min.X+x1, b.Min.Y+y0).RGBA()
			r2, g2, b2, _ := src.At(b.Min.X+x0, b.Min.Y+y1).RGBA()
			r3, g3, b3, _ := src.At(b.Min.X+x1, b.Min.Y+y1).RGBA()
			mix := func(a, b, c, d uint32) uint8 {
				return uint8(((float64(a)*(1-fx)+float64(b)*fx)*(1-fy) + (float64(c)*(1-fx)+float64(d)*fx)*fy) / 257)
			}
			i := dst.PixOffset(x, y)
			dst.Pix[i] = mix(r0, r1, r2, r3)
			dst.Pix[i+1] = mix(g0, g1, g2, g3)
			dst.Pix[i+2] = mix(b0, b1, b2, b3)
			dst.Pix[i+3] = 255
		}
	}
	return dst
}

// Three separable box passes approximate Gaussian blur, with O(width*height) work per pass.
func boxBlur(src *image.RGBA, radius int) *image.RGBA {
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	tmp := image.NewRGBA(src.Bounds())
	dst := image.NewRGBA(src.Bounds())
	den := 2*radius + 1
	for pass := 0; pass < 3; pass++ {
		for y := 0; y < h; y++ {
			var sums [3]int
			for x := -radius; x <= radius; x++ {
				i := src.PixOffset(max(0, min(w-1, x)), y)
				for c := 0; c < 3; c++ {
					sums[c] += int(src.Pix[i+c])
				}
			}
			for x := 0; x < w; x++ {
				i := tmp.PixOffset(x, y)
				for c := 0; c < 3; c++ {
					tmp.Pix[i+c] = uint8(sums[c] / den)
				}
				tmp.Pix[i+3] = 255
				left := src.PixOffset(max(0, x-radius), y)
				right := src.PixOffset(min(w-1, x+radius+1), y)
				for c := 0; c < 3; c++ {
					sums[c] += int(src.Pix[right+c]) - int(src.Pix[left+c])
				}
			}
		}
		for x := 0; x < w; x++ {
			var sums [3]int
			for y := -radius; y <= radius; y++ {
				i := tmp.PixOffset(x, max(0, min(h-1, y)))
				for c := 0; c < 3; c++ {
					sums[c] += int(tmp.Pix[i+c])
				}
			}
			for y := 0; y < h; y++ {
				i := dst.PixOffset(x, y)
				for c := 0; c < 3; c++ {
					dst.Pix[i+c] = uint8(sums[c] / den)
				}
				dst.Pix[i+3] = 255
				top := tmp.PixOffset(x, max(0, y-radius))
				bottom := tmp.PixOffset(x, min(h-1, y+radius+1))
				for c := 0; c < 3; c++ {
					sums[c] += int(tmp.Pix[bottom+c]) - int(tmp.Pix[top+c])
				}
			}
		}
		src, dst = dst, src
	}
	return src
}
