package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/c-bata/go-prompt"
	"github.com/google/uuid"
	"github.com/hibooboo2/finanalyzer/storage"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/rivo/tview"

	"time"
)

const DEBIT = "ATM WITHDRAWAL"

var db = storage.MustNew(&Transaction{})

func main() {
	// importTransactions()

	// orig()

	app := tview.NewApplication()
	l := tview.NewList()
	l.AddItem("Quit", "", 'q', func() {
		app.Stop()
	})

	l.AddItem("Import transactions", "", 'i', func() {
		m := tview.NewModal()
		m.SetText("Imported")
		m.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			app.SetRoot(l, true)
		})
		m.AddButtons([]string{"Ok", "Cancel"})
		app.SetRoot(m, true)
	})
	app.SetRoot(l, true)
	if err := app.Run(); err != nil {
		panic(err)
	}
}

func importTransactions() {
	f, err := os.Open("./filename_4.csv")
	checkError(err)
	r := csv.NewReader(f)

	records, err := r.ReadAll()
	checkError(err)

	replacer = strings.NewReplacer(emptySpace(numSym...)...)
	for _, record := range records {
		t := FromRecord(record)
		t.CreateID()
		db.MustSave(t)
	}
}

func orig() {
	transactions := []*Transaction{}
	err := db.Find(&transactions).Error
	if err != nil {
		panic(err)
	}

	repeatedWords := map[string]int{}
	for _, t := range transactions {
		if t.TransactionType != DEBIT {
			continue
		}
		words := strings.Split(t.Description, " ")
		for _, word := range words {
			if word == " " || word == "" {
				continue
			}
			repeatedWords[word]++
		}
	}

	for word, count := range repeatedWords {
		occurrencePct := float64(count) / float64(len(transactions))
		if occurrencePct < 0.04 {
			continue
		}
		if occurrencePct > 0.15 {
			repeatedWordsList = append(repeatedWordsList, word)
			continue
		}

		fmt.Printf("Remove this word (%s)? It occurs %d times in %f%% of transactions\n", word, count, occurrencePct)
		keep := prompt.Input(">", none)
		if keep == "y" {
			word = fmt.Sprintf(" %s ", word)
			repeatedWordsList = append(repeatedWordsList, word)
		}

	}

	replacer = strings.NewReplacer(empty(repeatedWordsList...)...)

	for _, t := range transactions {
		t.Description = RemovePos(t.Description)
	}

	keys := map[string]int{}
	for _, t := range transactions {
		switch t.TransactionType {
		case DEBIT:
			keys[t.Description]++
		}
	}
	threshold := float32(0.99)
	alreadyCompared := map[string]map[string]int{}
	grouped := getGrouped(keys, alreadyCompared, nil, threshold)
	for threshold > 0.3 {
		getGrouped(keys, alreadyCompared, grouped, threshold)
		threshold -= 0.05
	}

	for k, dups := range grouped {
		fmt.Printf("%s [%v]\n", k, dups)
	}

}
func getGrouped(keys map[string]int, alreadyCompared map[string]map[string]int, grouped map[string][]string, threshold float32) map[string][]string {
	possibleDups := getPossibleDups(keys, threshold)

	if grouped == nil {
		grouped = map[string][]string{}
	}

	if len(possibleDups) == 0 {
		return grouped
	}
	fmt.Println("Are all of these companies equal?")
	for k := range possibleDups {
		i := 0
		for dup := range possibleDups[k] {
			if bothIn(k, dup, grouped) {
				continue
			}
			if alreadyCompared[k][dup] > 0 || alreadyCompared[dup][k] > 0 {
				continue
			}
			i++
		}
		if i == 0 {
			continue
		}

		fmt.Printf("%s\n", k)
		for dup := range possibleDups[k] {
			if bothIn(k, dup, grouped) {
				continue
			}
			if alreadyCompared[k][dup] > 0 || alreadyCompared[dup][k] > 0 {
				continue
			}
			fmt.Printf("\t%s", dup)
		}
		fmt.Printf("\n")
	}
	ans := prompt.Input(">", none)

	allGood := ans == "y" || ans == "yes"
	for k := range possibleDups {
		for dup := range possibleDups[k] {
			if bothIn(k, dup, grouped) {
				continue
			}
			if alreadyCompared[k][dup] > 0 || alreadyCompared[dup][k] > 0 {
				continue
			}
			_, ok := alreadyCompared[k]
			if !ok {
				alreadyCompared[k] = map[string]int{}
			}
			alreadyCompared[k][dup]++
			areSame := false
			if !allGood {
				fmt.Printf("Are : %s and %s the same company?\n", k, dup)
				ans := prompt.Input(">", none)
				switch strings.ToLower(ans) {
				case "y", "yes":
					areSame = true
				}
			}
			if !allGood && !areSame {
				continue
			}
			fmt.Printf("What is the name of this company? [%s] [%s]\n", k, dup)
			company := prompt.Input(">", autoComplete(grouped))

			grouped[company] = append(grouped[company], dup)
			grouped[company] = append(grouped[company], k)
		}
	}
	return grouped
}
func none(d prompt.Document) []prompt.Suggest {
	if d.CurrentLine() == "exit" {
		os.Exit(0)
	}
	return nil
}
func autoComplete(grouped map[string][]string) func(prompt.Document) []prompt.Suggest {
	return func(d prompt.Document) []prompt.Suggest {
		if d.CurrentLine() == "exit" {
			os.Exit(0)
		}
		suggestions := []prompt.Suggest{}
		for k := range grouped {
			suggestions = append(suggestions, prompt.Suggest{
				Text: k,
			})
		}

		return prompt.FilterFuzzy(suggestions, d.CurrentLineBeforeCursor(), true)
	}
}

func bothIn(a, b string, vals map[string][]string) bool {
	return in(a, vals) && in(b, vals)
}

func in(a string, vals map[string][]string) bool {
	for k, v := range vals {
		if k == a {
			return true
		}
		for _, v := range v {
			if v == a {
				return true
			}
		}
	}
	return false
}

func getPossibleDups(keys map[string]int, threshold float32) map[string]map[string]int {
	possibleDups := map[string]map[string]int{}
	for k := range keys {
		for t := range keys {
			if k == t {
				continue
			}
			if possibleDups[k][t] > 0 {
				continue
			}
			if possibleDups[t][k] > 0 {
				continue
			}
			d := CompareTwoStrings(k, t)
			if d > threshold {
				if possibleDups[k] == nil {
					if possibleDups[t] != nil {
						possibleDups[t][k]++
						continue
					}
					possibleDups[k] = map[string]int{}
				}
				possibleDups[k][t]++
			}
		}
	}
	return possibleDups
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

type Transaction struct {
	ID              string `gorm:"primary_key"`
	PostDate        time.Time
	TransactionType string
	Description     string
	Orig            string
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
		Description:     replacer.Replace(record[2]),
		Orig:            record[2],
		Amount:          amt,
		Balance:         bal,
	}
}

func (t *Transaction) CreateID() {
	t.ID = uuid.NewMD5(uuid.Nil, []byte(fmt.Sprintf("%s", t.TransactionType, t.Amount, t.Description))).String()
}

var repeatedWordsList = []string{}
var numSym = emptySpace("0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "#", "*", ".", "&", "/", "\\", "@", "-", "'")

var replacer *strings.Replacer

func empty(args ...string) []string {
	vals := []string{}

	for _, arg := range args {
		vals = append(vals, arg)
		vals = append(vals, "")
	}
	return vals
}

func emptySpace(args ...string) []string {
	vals := []string{}

	for _, arg := range args {
		vals = append(vals, arg)
		vals = append(vals, " ")
	}
	return vals
}

func RemovePos(des string) string {
	des = strings.ReplaceAll(des, " ", "\r")
	des = strings.TrimSpace(des)
	des = strings.ToUpper(des)
	des = replacer.Replace(des)
	des = strings.ReplaceAll(des, "\r", " ")
	for des != strings.ReplaceAll(des, "  ", " ") {
		des = strings.ReplaceAll(des, "  ", " ")
	}
	repeated := map[string]int{}
	order := []string{}

	words := strings.Split(des, " ")
	for _, word := range words {
		if repeated[word] == 0 {
			order = append(order, word)
		}
		repeated[word]++
	}

	return strings.Join(order, " ")
}
