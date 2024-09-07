package prompt

type ModelFormat struct {
	SystemStart string
	SystemEnd   string
	UserStart   string
	UserEnd     string
	ModelStart  string
	ModelEnd    string
}

var MISTRAL_V2 = ModelFormat{
	SystemStart: "[INST]",
	SystemEnd:   "[/INST]Understood.</s>",
	UserStart:   "[INST]",
	UserEnd:     "[/INST]",
	ModelStart:  "",
	ModelEnd:    "</s>",
}
