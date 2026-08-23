package service

import (
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	"gotravel/internal/apperr"
	"gotravel/internal/geo"
	"gotravel/internal/model"
	"gotravel/internal/repository"
	"gotravel/internal/timeutil"
)

type Photo struct {
	Repo      *repository.Repos
	Trip      *Trip
	UploadDir string
}

func (s *Photo) Save(ctx context.Context, userID, tripID int64, sessionID *int64, lat, lng float64, caption, filename string, r io.Reader, size int64) (*model.Photo, error) {
	if _, err := s.Trip.mustTripMember(ctx, userID, tripID); err != nil {
		return nil, err
	}
	if !geo.ValidCoord(lat, lng) {
		return nil, apperr.New(http.StatusBadRequest, apperr.Validation, "photo coordinate invalid")
	}
	if size <= 0 || size > 10<<20 {
		return nil, apperr.New(http.StatusBadRequest, apperr.PayloadTooBig, "image must be 1B-10MB")
	}
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
	default:
		return nil, apperr.New(http.StatusBadRequest, apperr.Validation, "only jpg/png/gif/webp")
	}
	if err := os.MkdirAll(filepath.Join(s.UploadDir, "full"), 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(s.UploadDir, "thumb"), 0o755); err != nil {
		return nil, err
	}
	name := fmt.Sprintf("%d_%d%s", timeutil.Now().UnixNano(), userID, ext)
	fullRel := filepath.Join("full", name)
	thumbRel := filepath.Join("thumb", name)
	fullAbs := filepath.Join(s.UploadDir, fullRel)
	thumbAbs := filepath.Join(s.UploadDir, thumbRel)
	f, err := os.Create(fullAbs)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(f, io.LimitReader(r, 10<<20+1)); err != nil {
		f.Close()
		return nil, err
	}
	f.Close()
	if err := makeThumb(fullAbs, thumbAbs); err != nil {
		_ = copyFile(fullAbs, thumbAbs)
	}
	p := &model.Photo{
		TripID: tripID, SessionID: sessionID, UserID: userID,
		Lat: lat, Lng: lng, FilePath: fullRel, ThumbPath: thumbRel,
		Caption: strings.TrimSpace(caption), TakenAt: timeutil.Now(),
	}
	if err := s.Repo.InsertPhoto(ctx, p); err != nil {
		return nil, err
	}
	p.URL = "/uploads/" + filepath.ToSlash(fullRel)
	p.ThumbURL = "/uploads/" + filepath.ToSlash(thumbRel)
	return p, nil
}

func (s *Photo) List(ctx context.Context, userID, tripID int64) ([]model.Photo, error) {
	if _, err := s.Trip.mustTripMember(ctx, userID, tripID); err != nil {
		return nil, err
	}
	ps, err := s.Repo.Photos(ctx, tripID)
	if err != nil {
		return nil, err
	}
	for i := range ps {
		ps[i].URL = "/uploads/" + filepath.ToSlash(ps[i].FilePath)
		ps[i].ThumbURL = "/uploads/" + filepath.ToSlash(ps[i].ThumbPath)
	}
	return ps, nil
}

func makeThumb(src, dst string) error {
	img, err := imaging.Open(src, imaging.AutoOrientation(true))
	if err != nil {
		f, e2 := os.Open(src)
		if e2 != nil {
			return err
		}
		defer f.Close()
		decoded, _, e3 := image.Decode(f)
		if e3 != nil {
			return err
		}
		img = decoded
	}
	th := imaging.Fit(img, 480, 480, imaging.Lanczos)
	return imaging.Save(th, dst)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
