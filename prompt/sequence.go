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

var GEMMA_V2 = ModelFormat{
	SystemStart: "<start_of_turn>system\n",
	SystemEnd:   "<end_of_turn>",
	UserStart:   "<start_of_turn>user\n",
	UserEnd:     "<end_of_turn>",
	ModelStart:  "<start_of_turn>model\n",
	ModelEnd:    "<end_of_turn>",
}
