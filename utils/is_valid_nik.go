package utils

import (
	"errors"
	"fmt"
	"strconv"
	"time"
	"unicode"

	"github.com/sirupsen/logrus"
)

var validProvinceCodes = map[string]bool{
	"11": true, // Aceh
	"12": true, // Sumatera Utara
	"13": true, // Sumatera Barat
	"14": true, // Riau
	"15": true, // Jambi
	"16": true, // Sumatera Selatan
	"17": true, // Bengkulu
	"18": true, // Lampung
	"19": true, // Bangka Belitung
	"21": true, // Kepulauan Riau
	"31": true, // DKI Jakarta
	"32": true, // Jawa Barat
	"33": true, // Jawa Tengah
	"34": true, // DI Yogyakarta
	"35": true, // Jawa Timur
	"36": true, // Banten
	"51": true, // Bali
	"52": true, // Nusa Tenggara Barat
	"53": true, // Nusa Tenggara Timur
	"61": true, // Kalimantan Barat
	"62": true, // Kalimantan Tengah
	"63": true, // Kalimantan Selatan
	"64": true, // Kalimantan Timur
	"65": true, // Kalimantan Utara
	"71": true, // Sulawesi Utara
	"72": true, // Sulawesi Tengah
	"73": true, // Sulawesi Selatan
	"74": true, // Sulawesi Tenggara
	"75": true, // Gorontalo
	"76": true, // Sulawesi Barat
	"81": true, // Maluku
	"82": true, // Maluku Utara
	"91": true, // Papua
	"92": true, // Papua Barat
}

func IsValidNIK(nik string) bool {
	if len(nik) != 16 {
		return false
	}

	// Harus semua digit
	for _, r := range nik {
		if !unicode.IsDigit(r) {
			return false
		}
	}

	// Cek kode provinsi (2 digit pertama)
	provinceCode := nik[:2]
	if !validProvinceCodes[provinceCode] {
		return false
	}

	// Ambil tanggal lahir: DDMMYY (posisi 7–12)
	dayStr := nik[6:8]
	monthStr := nik[8:10]
	yearStr := nik[10:12]

	// Konversi ke int
	day, _ := strconv.Atoi(dayStr)
	month, _ := strconv.Atoi(monthStr)
	year, _ := strconv.Atoi(yearStr)

	// Koreksi untuk perempuan (jika day > 40)
	if day > 40 {
		day -= 40
	}

	// Tambahkan tahun (asumsi 00–99 jadi 1900–2099, kamu bisa sesuaikan)
	yearFull := 1900 + year
	if yearFull < 1950 {
		yearFull += 100 // anggap tahun 2000-an
	}

	// Cek validitas tanggal
	dateStr := fmt.Sprintf("%04d-%02d-%02d", yearFull, month, day)
	_, err := time.Parse("2006-01-02", dateStr)
	return err != nil
}
func IsNIKValid(nik string) error {
	if len(nik) != 16 {
		return errors.New("digit NIK/KTP Harus 16")
	}

	// Harus semua digit
	for _, r := range nik {
		if !unicode.IsDigit(r) {
			return errors.New("digit NIK/KTP mengandung huruf")
		}
	}

	// Cek kode provinsi (2 digit pertama)
	provinceCode := nik[:2]
	if !validProvinceCodes[provinceCode] {
		return errors.New("kode provinsi tidak valid ")
	}

	// Ambil tanggal lahir: DDMMYY (posisi 7–12)
	dayStr := nik[6:8]
	monthStr := nik[8:10]
	yearStr := nik[10:12]

	// Konversi ke int
	day, _ := strconv.Atoi(dayStr)
	month, _ := strconv.Atoi(monthStr)
	year, _ := strconv.Atoi(yearStr)

	// Koreksi untuk perempuan (jika day > 40)
	if day > 40 {
		day -= 40
	}

	// Tambahkan tahun (asumsi 00–99 jadi 1900–2099, kamu bisa sesuaikan)
	yearFull := 1900 + year
	if yearFull < 1950 {
		yearFull += 100 // anggap tahun 2000-an
	}

	// Cek validitas tanggal
	dateStr := fmt.Sprintf("%04d-%02d-%02d", yearFull, month, day)
	_, err := time.Parse("2006-01-02", dateStr)
	return err
}

// IsValidNIKWithGenderDOB validates NIK and returns gender and date of birth if valid
func IsValidNIKWithGenderDOB(nik string) (gender string, dob string, err error) {
	if len(nik) != 16 {
		return "", "", errors.New("NIK harus terdiri dari 16 digit")
	}

	for _, r := range nik {
		if !unicode.IsDigit(r) {
			return "", "", errors.New("NIK hanya boleh berisi angka")
		}
	}

	provinceCode := nik[:2]
	if !validProvinceCodes[provinceCode] {
		return "", "", fmt.Errorf("kode provinsi tidak valid: %s", provinceCode)
	}

	dayStr := nik[6:8]
	monthStr := nik[8:10]
	yearStr := nik[10:12]

	day, err := strconv.Atoi(dayStr)
	if err != nil {
		logrus.Error(err)
		return "", "", errors.New("format tanggal lahir tidak valid (hari)")
	}
	month, err := strconv.Atoi(monthStr)
	if err != nil {
		logrus.Error(err)
		return "", "", errors.New("format tanggal lahir tidak valid (bulan)")
	}
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		logrus.Error(err)
		return "", "", errors.New("format tanggal lahir tidak valid (tahun)")
	}

	if day > 40 {
		day -= 40
		gender = "wanita"
	} else {
		gender = "pria"
	}

	// Asumsikan tahun lahir di rentang 1950–2049
	yearFull := 1900 + year
	if yearFull < 1950 {
		yearFull += 100
	}

	// Cek validitas tanggal
	tanggalLahir := fmt.Sprintf("%04d-%02d-%02d", yearFull, month, day)
	_, err = time.Parse("2006-01-02", tanggalLahir)
	if err != nil {
		logrus.Error(err)
		return "", "", errors.New("tanggal lahir tidak valid")
	}

	return gender, tanggalLahir, nil
}
