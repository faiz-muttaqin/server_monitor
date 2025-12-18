package utils

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// Fungsi untuk mengambil 1 kata positif secara random
func getRandomPositiveWord(words []string) string {
	rand.Seed(time.Now().UnixNano())
	return words[rand.Intn(len(words))]
}

func GetRandomWord() string {
	// Daftar kata positif
	positiveWords := []string{
		"bahagia",
		"semangat",
		"terbaik",
		"berjaya",
		"bersyukur",
		"sukses",
		"berani",
		"optimis",
		"ikhlas",
		"hebat",
		"berprestasi",
		"jujur",
		"peduli",
		"cerdas",
		"jago",
		"beruntung",
		"ramah",
		"terampil",
	}
	return getRandomPositiveWord(positiveWords)
}

// Fungsi untuk generate angka random dengan panjang n digit
func getRandomNumberString(n int) string {
	rand.Seed(time.Now().UnixNano())
	number := ""
	for i := 0; i < n; i++ {
		number += fmt.Sprintf("%d", rand.Intn(10))
	}
	return number
}

// Fungsi untuk mengambil satu karakter khusus random
func getRandomSpecialChar() string {
	specialChars := "!@#$%&*"
	rand.Seed(time.Now().UnixNano())
	return string(specialChars[rand.Intn(len(specialChars))])
}

// Fungsi utama untuk generate password
func GenerateRandomPassword(name string) string {
	// randomWord := GetRandomWord()
	randomNumber := getRandomNumberString(4) // 4 digit
	randomSpecial := getRandomSpecialChar()
	name = strings.ReplaceAll(name, " ", "_")
	return fmt.Sprintf("%s_%s%s", name, randomNumber, randomSpecial)
}
func GenerateRandomPasswordNameBOD(name string, bod time.Time) string {
	randomNumber := getRandomNumberString(4) // 4 digit
	randomSpecial := getRandomSpecialChar()
	name = strings.ReplaceAll(name, " ", "_")

	bodFormatted := bod.Format("02012006") // ddmmyyyy format

	return fmt.Sprintf("%s_%s_%s%s", name, bodFormatted, randomNumber, randomSpecial)
}
func GenerateRandomPasswordBOD(bod time.Time) string {
	randomNumber := getRandomNumberString(4) // 4 digit
	randomSpecial := getRandomSpecialChar()

	bodFormatted := bod.Format("02012006") // ddmmyyyy format

	return fmt.Sprintf("%s_%s%s", bodFormatted, randomNumber, randomSpecial)
}
