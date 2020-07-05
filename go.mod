module nicotrans

go 1.14

require (
	github.com/op/go-logging v0.0.0-20160315200505-970db520ece7
	internal/translator v1.0.0
	internal/utils v1.0.0
)

replace internal/translator => ./internal/translator

replace internal/utils => ./internal/utils
