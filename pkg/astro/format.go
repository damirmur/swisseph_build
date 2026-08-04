package astro

import "fmt"

var Signs = []string{
	"Ari", "Tau", "Gem", "Can", "Leo", "Vir",
	"Lib", "Sco", "Sag", "Cap", "Aqu", "Pis",
}

// FormatDegree formats absolute longitude (0-360) into degrees within a sign
func FormatDegree(lon float64) string {
	for lon >= 360 {
		lon -= 360
	}
	for lon < 0 {
		lon += 360
	}
	signIdx := int(lon / 30)
	degInSign := lon - float64(signIdx*30)

	deg := int(degInSign)
	min := int((degInSign - float64(deg)) * 60)

	return fmt.Sprintf("%02d°%02d' %s", deg, min, Signs[signIdx])
}

func FormatSpeed(speed float64) string {
	if speed < 0 {
		return fmt.Sprintf("%7.4f R", speed)
	}
	return fmt.Sprintf("%7.4f D", speed)
}
