package translator

import "sort"

// TranslatedSequence 번역된 시퀀스
type TranslatedSequence struct {
	index      int
	Original   string
	Translated string
}

// SortTranslatedSequence 번역된 시퀀스를 요청한 순으로 나열합니다
func SortTranslatedSequence(seq *[]TranslatedSequence) {
	sort.Slice(*seq, func(i, j int) bool {
		return (*seq)[i].index < (*seq)[j].index
	})
}
