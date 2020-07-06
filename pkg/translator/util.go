package translator

import "sort"

type translatedSequence struct {
	index int
	text  string
}

func sortTranslatedSequence(seq *[]translatedSequence) {
	sort.Slice(*seq, func(i, j int) bool {
		return (*seq)[i].index < (*seq)[j].index
	})
}
