package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func main() {
	f, err := os.Open("./filename_4.csv")
	checkError(err)
	r := csv.NewReader(f)

	records, err := r.ReadAll()
	checkError(err)

	transactions := []*Transaction{}
	for _, record := range records {
		transactions = append(transactions, FromRecord(record))
	}

	grouped := map[string][]Transaction{}
	for _, t := range transactions {
		switch t.TransactionType {
		case "ATM WITHDRAWAL":
			grouped[t.Description] = append(grouped[t.Description], *t)
		}
	}

	for k, v := range grouped {
		if k == "Categorized" {
			continue
		}
		for _, v := range v {
			fmt.Println(k, v.Amount)
			break
		}
	}

}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

type Transaction struct {
	PostDate        time.Time
	TransactionType string
	Description     string
	Amount          float64
	Balance         float64
}

func FromRecord(record []string) *Transaction {
	t, err := time.Parse("1/2/2006", record[0])
	checkError(err)
	val := record[3][2 : len(record[3])-2]
	if record[3][0] != '(' {
		val = record[3][1 : len(record[3])-2]
	}
	amt, err := strconv.ParseFloat(val, 64)
	if record[3][0] == '(' {
		amt = -amt
	}
	checkError(err)
	val = record[4][2 : len(record[4])-1]
	if record[4][0] != '(' {
		val = record[4][1 : len(record[4])-1]
	}
	bal, err := strconv.ParseFloat(val, 64)
	if record[4][0] == '(' {
		amt = -amt
	}
	checkError(err)
	return &Transaction{
		PostDate:        t,
		TransactionType: record[1],
		Description:     RemovePos(strings.TrimSpace(record[2])),
		Amount:          amt,
		Balance:         bal,
	}
}

var matchers = map[string]*regexp.Regexp{
	"FRYS FOOD AND DRUG":       regexp.MustCompile(`FRYS #\d{4} \w{2}`),
	"FRYS FUEL":                regexp.MustCompile(`FRYS FUEL #\d{4} \w{2}`),
	"CIRCLE K":                 regexp.MustCompile(`CIRCLE K ?#? ?\d{5}`),
	"Amazon Prime Memebership": regexp.MustCompile(`AMZN.COM/BILL`),
	"BURGER KING":              regexp.MustCompile(`BURGER KING`),
	"BEST TRUE VALUE":          regexp.MustCompile(`BEST TRUE VALUE`),
	"WALMART":                  regexp.MustCompile(`(WAL-MART #\w{4}|WAL-MART SUPER|WM SUPERCENTER)`),
	"AMAZON":                   regexp.MustCompile(`AMAZON\.COM\*\w{5}`),
	"PINAL COUNTY-K52":         regexp.MustCompile(`#\d{6} PINAL COUNTY-K52`),
	"NETFLIX":                  regexp.MustCompile(`NETFLIX\.?COM`),
	"SALLY BEAUTY":             regexp.MustCompile(`SALLY BEAUTY #\d\d\d?\d?`),
	"AUTO SPA":                 regexp.MustCompile(`AUTO SPA?`),
	"QT":                       regexp.MustCompile(`QT \w* `),
}

var contained = []string{
	"ADR HARDWARE",
	"SPEEDWAY",
	"JIMMY JOHNS",
	"COSTCO WHSE",
	"COSTCO GAS",
	"MCDONALD'S",
	"PANDA EXPRESS",
	"TOBACCO",
	"WENDY'S",
}

var replacer = strings.NewReplacer("0", "\\d", "1", "\\d", "2", "\\d", "3", "\\d", "4", "\\d", "5", "\\d", "6", "\\d", "7", "\\d", "8", "\\d", "9", "\\d", "#", "")

func RemovePos(des string) string {
	if strings.Contains(des, "*POS*") {
		des = des[strings.Index(des, "*POS*")+len("*POS*")+1:]
	}
	des = strings.TrimSpace(des)
	des = strings.ToUpper(des)
	for _, matcher := range matchers {
		if matcher.MatchString(des) {
			return "Categorized"
		}
	}

	for _, cat := range contained {
		if strings.Contains(des, cat) {
			return "Categorized"
		}
	}
	// des = replacer.Replace(des)
	return des
}
