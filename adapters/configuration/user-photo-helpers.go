package configuration

import (
    //"github.com/nfnt/resize"
    "golang.org/x/image/draw"
    "image"
    _ "image/png"
    "image/jpeg"
    "io/ioutil"
    "errors"
    "bytes"
    //"fmt"
    "io"
)

const (
    minPhotoWidth = 300 // beward requirements
    maxPhotoWidth = 800
    minPhotoHeight = 300 // beward requirements
    maxPhotoHeight = 800
    maxPhotoBytes = 100 * 1024 - 1 // beward constraints
)

var photoSizeError = errors.New("Uploaded image dimensions are too small")

func makeUserPhoto(reader io.Reader) (photo []byte, err error) {
    data, err := ioutil.ReadAll(reader)
    if nil != err {return}
    reader = bytes.NewReader(data)
    im, imageType, err := image.DecodeConfig(reader)
    if err != nil {return}
    //fmt.Println("cfg.http.photo", imageType, im.Width, im.Height, im)
    if im.Width < minPhotoWidth || im.Height < minPhotoHeight {
        return nil, photoSizeError
    }

    if "jpg" != imageType || im.Width > maxPhotoWidth || im.Height > maxPhotoHeight || len(data) > maxPhotoBytes {
        reader = bytes.NewReader(data)
        photo, err = formatPhoto(reader)
    } else {
        photo = data
    }
    return
}

func formatPhoto(reader io.Reader) (data []byte, err error) {
    src, _, err := image.Decode(reader)
    if err != nil {return}

    dst := fitImage(src, maxPhotoWidth, maxPhotoHeight)
    
    if dst.Bounds().Max.X < minPhotoWidth || dst.Bounds().Max.Y < minPhotoHeight {
        return nil, photoSizeError
    }
    //fmt.Println("cfg.http.photoWH", dst.Bounds().Max.X, dst.Bounds().Max.Y)
    
    buf := new(bytes.Buffer)
    opt := new(jpeg.Options)
    for q := 85; q > 20; q-=5 {
        buf.Reset()
        opt.Quality = q
        jpeg.Encode(buf, dst, opt)
        if buf.Len() <= maxPhotoBytes {
            break
        }
    }
    //fmt.Println("cfg.http.photoSize", buf.Len(), opt.Quality)
    return buf.Bytes(), nil
}

func fitImage(img image.Image, maxWidth, maxHeight int) image.Image {
	origBounds := img.Bounds()
	origWidth := origBounds.Dx()
	origHeight := origBounds.Dy()
	newWidth, newHeight := origWidth, origHeight

	// Return original image if it have same or smaller size as constraints
	if maxWidth >= origWidth && maxHeight >= origHeight {
		return img
	}

	// Preserve aspect ratio
	if origWidth > maxWidth {
		newHeight = origHeight * maxWidth / origWidth
		if newHeight < 1 {
			newHeight = 1
		}
		newWidth = maxWidth
	}

	if newHeight > maxHeight {
		newWidth = newWidth * maxHeight / newHeight
		if newWidth < 1 {
			newWidth = 1
		}
		newHeight = maxHeight
	}

    // Set the expected size that you want:
    dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
    // Resize:
    draw.BiLinear.Scale(dst, dst.Rect, img, img.Bounds(), draw.Over, nil)
    
    return dst
}