package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
)

const (
	userEmail = "username@gmail.com"
	password  = "passw0rd"
)

func main() {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	var currentAppointmentDate time.Time
	var findedDate string
	var currentAppointmentDateHTML string
	var findedDateHTML string

	err := chromedp.Run(ctx,
		chromedp.Navigate("https://ais.usvisa-info.com/tr-tr/niv/users/sign_in"),
		chromedp.Sleep(3*time.Second),
		chromedp.Click(`input[name="user[email]"]`),
		chromedp.SendKeys(`input[name="user[email]"]`, userEmail),
		chromedp.Click(`input[name="user[password]"]`),
		chromedp.SendKeys(`input[name="user[password]"]`, password),
		chromedp.Click(`label[for="policy_confirmed"]`),
		chromedp.Click(`input[name="commit"]`),
		chromedp.Sleep(5*time.Second),
		chromedp.WaitVisible(`img`, chromedp.ByQuery),
		chromedp.OuterHTML(`body`, &currentAppointmentDateHTML),
	)

	if err != nil {
		log.Fatal(err)
	}

	currentAppointmentDate, err = extractCurrentAppointmentDate(currentAppointmentDateHTML)
	if err != nil {
		log.Fatal(err)
	}

	err = chromedp.Run(ctx,
		chromedp.Click(`a:contains("Devam Et")`),
		chromedp.Sleep(2*time.Second),
		chromedp.Click(`a:contains("Randevuyu Yeniden Zamanla")`),
		chromedp.Sleep(3*time.Second),
		chromedp.Click(`a:contains("Randevuyu Yeniden Zamanla")`),
		chromedp.Sleep(3*time.Second),
		chromedp.Click(`input[name="appointments_consulate_appointment_date"]`),
		chromedp.Sleep(3*time.Second),
		chromedp.OuterHTML(`#ui-datepicker-div`, &findedDateHTML),
	)

	if err != nil {
		log.Fatal(err)
	}

	findedDate, err = extractEarliestAvailableDate(findedDateHTML)
	if err != nil {
		log.Fatal(err)
	}

	compareDatesAndNotify(currentAppointmentDate, findedDate)

	err = chromedp.Run(ctx,
		chromedp.Click(`a:contains("Eylemler")`),
		chromedp.Click(`a:contains("Oturumu Kapat")`),
	)

	if err != nil {
		log.Fatal(err)
	}
}

func extractCurrentAppointmentDate(html string) (time.Time, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return time.Time{}, err
	}

	givenStr := doc.Find(".consular-appt").Text()
	lines := strings.Split(givenStr, "\n")
	dateStr := strings.TrimSpace(lines[1])
	parts := strings.Split(dateStr, ",")
	dayMonth := strings.TrimSpace(parts[0])
	year := strings.TrimSpace(parts[1])

	dayMonthParts := strings.Split(dayMonth, " ")
	day := dayMonthParts[0]
	month := dayMonthParts[1]

	months := map[string]time.Month{
		"Ocak": time.January, "Şubat": time.February, "Mart": time.March,
		"Nisan": time.April, "Mayıs": time.May, "Haziran": time.June,
		"Temmuz": time.July, "Ağustos": time.August, "Eylül": time.September,
		"Ekim": time.October, "Kasım": time.November, "Aralık": time.December,
	}

	dayInt, err := strconv.Atoi(day)
	if err != nil {
		return time.Time{}, err
	}

	yearInt, err := strconv.Atoi(year)
	if err != nil {
		return time.Time{}, err
	}

	return time.Date(yearInt, months[month], dayInt, 0, 0, 0, 0, time.UTC), nil
}

func extractEarliestAvailableDate(html string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", err
	}

	tdElements := doc.Find("td.undefined")

	var dataList []string
	tdElements.Each(func(i int, s *goquery.Selection) {
		if s.AttrOr("data-event", "") == "click" && s.AttrOr("data-handler", "") == "selectDay" {
			date := s.Find("a").Text()
			month := s.AttrOr("data-month", "")
			year := s.AttrOr("data-year", "")

			if len(month) == 1 {
				month = "0" + month
			}

			dateFormatted := fmt.Sprintf("%02s/%s/%s", date, month, year)
			dataList = append(dataList, dateFormatted)
		}
	})

	if len(dataList) == 0 {
		return "", fmt.Errorf("no available dates found")
	}

	return dataList[0], nil
}

func compareDatesAndNotify(currentAppointmentDate time.Time, findedDate string) {
	date1, err := time.Parse("02/01/2006", findedDate)
	if err != nil {
		log.Fatal(err)
	}

	randTarihi := currentAppointmentDate.Format("02/01/2006")

	if date1.Before(currentAppointmentDate) {
		fmt.Printf("Daha erkene randevu buldum. %s tarihine randevu açıldı. Sizin güncel randevu tarihiniz: %s\n", findedDate, randTarihi)
	} else {
		fmt.Printf("En erken randevu tarihi %s. Sizin randevu tarihiniz: %s\n", findedDate, randTarihi)
	}
}
