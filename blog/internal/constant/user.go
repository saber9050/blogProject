package constant

// ImageMaxLength 图片文件大小限制
const ImageMaxLength = 10 << 20 // 10 MB

// AllowedImageExt 图片文件格式限制
var AllowedImageExt = map[string]struct{}{
	".jpg":  {},
	".jpeg": {},
	".png":  {},
	".webp": {},
}
