package srs

// Konstanta batas bawah/atas ease factor, mengikuti konvensi umum
// algoritma SM-2 (dipakai Anki dkk).
const (
	minEaseFactor     = 1.3
	defaultEaseFactor = 2.5
)

// ScheduleNextReview menghitung ease factor & interval hari berikutnya
// untuk sebuah flashcard, berdasarkan rating kesulitan yang dipilih user
// ("hard" | "good" | "easy"). Ini murni algoritma penjadwalan — TIDAK
// memanggil AI/Gemini sama sekali (lihat Fase 4 di roadmap).
func ScheduleNextReview(easeFactor float64, intervalDays int, rating string) (newEaseFactor float64, newIntervalDays int) {
	if easeFactor <= 0 {
		easeFactor = defaultEaseFactor
	}
	if intervalDays <= 0 {
		intervalDays = 1
	}

	switch rating {
	case "hard":
		newEaseFactor = easeFactor - 0.15
		newIntervalDays = int(float64(intervalDays) * 1.2)
	case "easy":
		newEaseFactor = easeFactor + 0.15
		newIntervalDays = int(float64(intervalDays) * easeFactor * 1.3)
	default: // "good" atau rating tak dikenal dianggap netral
		newEaseFactor = easeFactor
		newIntervalDays = int(float64(intervalDays) * easeFactor)
	}

	if newEaseFactor < minEaseFactor {
		newEaseFactor = minEaseFactor
	}
	if newIntervalDays < 1 {
		newIntervalDays = 1
	}

	return newEaseFactor, newIntervalDays
}
