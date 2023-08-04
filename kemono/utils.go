package kemono

func isImage(ext string) bool {
	switch ext {
	case ".apng", ".avif", ".bmp", ".gif", ".ico", ".cur", ".jpg", ".jpeg", ".jfif", ".pjpeg", ".pjp", ".png", ".svg", ".tif", ".tiff", ".webp", ".jpe":
		return true
	default:
		return false
	}
}
